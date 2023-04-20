package internal

import (
	"bytes"
	"embed"
	"io/ioutil"
	"net/http"
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
	"golang.org/x/crypto/bcrypt"
)

// StaticFS is the embed for the static files.
var StaticFS embed.FS

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

	// Temperatures
	intempsensor := temperature.NewTemperature(viper.GetString("temperature.inside.name"), viper.GetString("temperature.inside.type"), viper.GetInt("temperature.inside.pin"), viper.GetBool("temperature.inside.boost"))
	outtempsensor := temperature.NewTemperature(viper.GetString("temperature.outside.name"), viper.GetString("temperature.outside.type"), viper.GetInt("temperature.outside.pin"), viper.GetBool("temperature.outside.boost"))

	// Create the coop instance
	c, err := coop.New(viper.GetFloat64("coop.latitude"), viper.GetFloat64("coop.longitude"), d, viper.GetString("coop.opening.mode"), 
	                   viper.GetString("coop.opening.value"), viper.GetString("coop.closing.mode"), viper.GetString("coop.closing.value"), 
					   notifiers, true, false)
	if err != nil {
		logrus.WithError(err).Fatalln("Error while creating the coop instance")
	}

	// Initialize Web controllers
	logrus.Infoln("Initializing the services")
	coopService := services.NewCoopService(c, intempsensor, outtempsensor)
	logrus.Infoln("Successfully initialized the services")

	// Initialize Web controllers
	logrus.Infoln("Initializing the Web controllers")
	//coopCtrl := routes.NewCoopController(coopService)
	miscCtrl := routes.NewMiscController(coopService)
	//securityCtrl := routes.NewSecurityController(viper.GetString("general.gui_username"), viper.GetString("general.gui_password"))
	logrus.Infoln("Successfully initialized the Web controllers")

	// Set the Basic authenticator
	authenticator := auth.NewBasicAuthenticator("example.com", Secret)

	// Static files
	var staticFS = http.FS(StaticFS)
	fs := http.FileServer(staticFS)

	// Handlers
	router := mux.NewRouter()
	router.PathPrefix("/static/").Handler(fs)
	router.HandleFunc("/", authenticator.Wrap(miscCtrl.Index))
	router.HandleFunc("/configuration", authenticator.Wrap(miscCtrl.Configuration))
	router.HandleFunc("/coop/open", authenticator.Wrap(miscCtrl.OpenCoopDoorManually))
	router.HandleFunc("/coop/close", authenticator.Wrap(miscCtrl.CloseCoopDoorManually))
	router.HandleFunc("/coop/stop", authenticator.Wrap(miscCtrl.StopCoopDoorManually))
	//http.Handle("/static/", fs)
	//http.HandleFunc("/", authenticator.Wrap(miscCtrl.Index))
	//http.HandleFunc("/configuration", authenticator.Wrap(miscCtrl.Configuration))

	// Serve
	addr := ":8000"
	s := &http.Server{
		Addr:           addr,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	logrus.WithFields(logrus.Fields{
		"port": addr,
	}).Infoln("Starting the Web server")
	err = s.ListenAndServe()
	if err != nil {
		logrus.WithError(err).Fatalln("Error while starting the Web server")
	}
}

// Secret holds the secret password.
func Secret(user, realm string) string {
	if user == viper.GetString("general.gui_username") {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(viper.GetString("general.gui_username")), bcrypt.DefaultCost)
		if err == nil {
			return string(hashedPassword)
		}
	}

	return ""
}
