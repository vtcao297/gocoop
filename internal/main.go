package internal

import (
	"bytes"
	"fmt"
	"embed"
	"io/ioutil"
	"net/http"
	"crypto/tls"
	"time"

	"github.com/fallais/gocoop/internal/routes"
	"github.com/fallais/gocoop/internal/services"
	"github.com/fallais/gocoop/internal/system"
	"github.com/fallais/gocoop/pkg/coop"
	"github.com/fallais/gocoop/pkg/door"
	"github.com/fallais/gocoop/pkg/motor"
	"github.com/fallais/gocoop/pkg/motor/bts7960"
	"github.com/fallais/gocoop/pkg/motor/l293d"
	"github.com/fallais/gocoop/pkg/motor/l298n"
	"github.com/fallais/gocoop/pkg/temperature"

	auth "github.com/abbot/go-http-auth"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/gorilla/mux"
	"github.com/stianeikeland/go-rpio/v4"
	"golang.org/x/crypto/bcrypt"
)

// StaticFS is the embed for the static files.
var StaticFS embed.FS

const (
	maxFailedAttempts = 5
	throttleInterval  = 1 * time.Minute
)

var failedAttempts = make(map[string]int)

// Run is a convenient function for Cobra.
func Run(cmd *cobra.Command, args []string) {
	// Flags
	configFile, err := cmd.Flags().GetString("config")
	if err != nil {
		logrus.WithError(err).Fatalln("Error while getting the flag for configuration data")
	}

	// Read configuration file
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		logrus.WithError(err).Fatalln("Error while reading configuration file")
	}

	// Initialize configuration values with Viper
	viper.SetConfigType("yaml")
	err = viper.ReadConfig(bytes.NewBuffer(data))
	if err != nil {
		logrus.WithError(err).Fatalln("Error when reading configuration data")
	}

	// Initialize RPIO
	err = rpio.Open()
	if err != nil {
		logrus.WithError(err).Fatalln("Error opening GPIO")
	}

	// Motor
	var motor motor.Motor
	logrus.WithFields(logrus.Fields{
		"type": viper.GetString("door.motor.type"),
	}).Infoln("Creating the motor")
	switch viper.GetString("door.motor.type") {
	case "l298n":
		motor = l298n.NewL298N(viper.GetInt("door.motor.pin_1A"), viper.GetInt("door.motor.pin_1B"), viper.GetInt("door.motor.pin_enable1"))
	case "l293d":
		motor = l293d.NewL293D(viper.GetInt("door.motor.pin_1A"), viper.GetInt("door.motor.pin_1B"), viper.GetInt("door.motor.pin_enable1"))
	case "bts7960":
		motor = bts7960.NewBTS7960(viper.GetInt("door.motor.forward_PWM"), viper.GetInt("door.motor.backward_PWM"), viper.GetInt("door.motor.forward_enable"), viper.GetInt("door.motor.backward_enable"))
	default:
		logrus.Fatalln("Motor type does not exist")
	}
	logrus.Infoln("Successfully created the motor")

	// Door
	logrus.Infoln("Creating the door")
	d := door.NewDoor(motor, viper.GetDuration("door.opening_duration"), viper.GetDuration("door.closing_duration"))
	logrus.Infoln("Successfully created the door")

	// Notifiers
	notifiers := system.SetupNotifiers()

	// Temperatures and Fan
	intempsensor := temperature.NewTemperature(viper.GetString("temperature.inside.name"), viper.GetString("temperature.inside.type"), viper.GetInt("temperature.inside.pin"))
	outtempsensor := temperature.NewTemperature(viper.GetString("temperature.outside.name"), viper.GetString("temperature.outside.type"), viper.GetInt("temperature.outside.pin"))

	// Create the coop instance
	isAutomaticAtStartup := false
	notifyAtStartup := false
	c, err := coop.New(viper.GetFloat64("coop.latitude"), viper.GetFloat64("coop.longitude"), d, viper.GetString("coop.opening.mode"), 
	                   viper.GetString("coop.opening.value"), viper.GetString("coop.closing.mode"), viper.GetString("coop.closing.value"), 
					   notifiers, isAutomaticAtStartup, notifyAtStartup)
	if err != nil {
		logrus.WithError(err).Fatalln("Error while creating the coop instance")
	}

	// Initialize Service controllers
	logrus.Infoln("Initializing the services")
	coopService := services.NewCoopService(c, intempsensor, outtempsensor)
	logrus.Infoln("Successfully initialized the services")

	// Initialize Web controllers
	logrus.Infoln("Initializing the Web controllers")
	miscCtrl := routes.NewMiscController(coopService)
	logrus.Infoln("Successfully initialized the Web controllers")

	// Set the Basic authenticator
	authenticator := auth.NewBasicAuthenticator(viper.GetString("general.site"), Secret)

	// Create a logout handler function that will be used to handle requests to the logout URL
	logoutHandler := func(w http.ResponseWriter, r *http.Request) {
		authRealm := viper.GetString("general.site")

		// Send a 401 Unauthorized response to the client with a
		// WWW-Authenticate header containing the authentication realm
		w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, authRealm))
		w.WriteHeader(http.StatusUnauthorized)

		// Redirect the user to a blank page after logging out
		http.Redirect(w, r, "about:blank", http.StatusSeeOther)
	}

	// Static files
	var staticFS = http.FS(StaticFS)
	fs := http.FileServer(staticFS)

	// Handlers
	router := mux.NewRouter()
	router.PathPrefix("/static/").Handler(fs)
	router.HandleFunc("/logout", logoutHandler)
	router.HandleFunc("/", authenticator.Wrap(miscCtrl.Index))
	router.HandleFunc("/configuration", authenticator.Wrap(miscCtrl.Configuration))
	router.HandleFunc("/coop/open", authenticator.Wrap(miscCtrl.OpenCoopDoorManually))
	router.HandleFunc("/coop/close", authenticator.Wrap(miscCtrl.CloseCoopDoorManually))
	router.HandleFunc("/coop/stop", authenticator.Wrap(miscCtrl.StopCoopDoorManually))
	router.HandleFunc("/coop/temperature", authenticator.Wrap(miscCtrl.GetCoopTemperature))
	router.HandleFunc("/coop/camera/still", authenticator.Wrap(miscCtrl.ProcessCaptureRequest))

	// Load TLS certificate and private key
	cert, err := tls.LoadX509KeyPair(viper.GetString("general.tls_cert"), viper.GetString("general.tls_key"))
	if err != nil {
		logrus.WithError(err).Fatalln("failed to load TLS certificate and key")
	}

	// Create a new TLS configuration with the loaded certificate and key
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	// Serve
	addr := ":443"
	s := &http.Server{
		Addr:           addr,
		TLSConfig: 		config,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	logrus.WithFields(logrus.Fields{
		"port": addr,
	}).Infoln("Starting the Web server")
	err = s.ListenAndServeTLS("", "")
	if err != nil {
		logrus.WithError(err).Fatalln("Error while starting the Web server")
	}
}

// Secret holds the secret password.
func Secret(user, realm string) string {
	if user == viper.GetString("general.gui_username") {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(viper.GetString("general.gui_password")), bcrypt.DefaultCost)
		if err == nil {
			return string(hashedPassword)
		}
	}

	return ""
}
