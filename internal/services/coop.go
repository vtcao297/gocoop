package services

import (
	"fmt"

	"github.com/fallais/gocoop/pkg/coop"
	"github.com/fallais/gocoop/pkg/coop/conditions"
	"github.com/fallais/gocoop/pkg/coop/conditions/sunbased"
	"github.com/fallais/gocoop/pkg/coop/conditions/timebased"
	"github.com/fallais/gocoop/pkg/temperature"
	"github.com/fallais/gocoop/pkg/fan"
	"github.com/spf13/viper"
)

//------------------------------------------------------------------------------
// Constants
//------------------------------------------------------------------------------

// ErrCoopOpened ...
const ErrCoopOpened = "The coop is already opened"

// ErrCoopOpening ...
const ErrCoopOpening = "The coop is opening"

// ErrCoopClosed ...
const ErrCoopClosed = "The coop is already closed"

// ErrCoopClosing ...
const ErrCoopClosing = "The coop is closing"

//------------------------------------------------------------------------------
// Structure
//------------------------------------------------------------------------------

type coopService struct {
	coop *coop.Coop
	InTempSensor temperature.Temperature
	OutTempSensor temperature.Temperature
}

//------------------------------------------------------------------------------
// Factory
//------------------------------------------------------------------------------

// NewCoopService returns a new CoopService.
func NewCoopService(coop *coop.Coop, indoorTemp temperature.Temperature, outsideTemp temperature.Temperature) CoopService {
	return &coopService {
		coop: coop,
		InTempSensor: indoorTemp,
		OutTempSensor: outsideTemp,
	}
}

//------------------------------------------------------------------------------
// Services
//------------------------------------------------------------------------------

// Get returns the the coop.
func (service *coopService) GetCoop() *coop.Coop {
	return service.coop
}

// Update updates the coop.
func (service *coopService) Update(input CoopUpdateRequest) error {
	// Create the opening condition
	var openingCondition conditions.Condition
	switch input.OpeningCondition.Mode {
	case "time_based":
		oc, err := timebased.NewTimeBasedCondition(input.OpeningCondition.Value)
		if err != nil {
			return fmt.Errorf("Error while creating the opening condition: %s", err)
		}

		openingCondition = oc
	case "sun_based":
		oc, err := sunbased.NewSunBasedCondition(input.OpeningCondition.Value, viper.GetFloat64("coop.latitude"), viper.GetFloat64("coop.longitude"))
		if err != nil {
			return fmt.Errorf("Error while creating the opening condition")
		}

		openingCondition = oc
	default:
		return fmt.Errorf("opening mode is incorrect: %s", input.OpeningCondition.Mode)
	}

	// Create the closing condition
	var closingCondition conditions.Condition
	switch input.ClosingCondition.Mode {
	case "time_based":
		cc, err := timebased.NewTimeBasedCondition(input.ClosingCondition.Value)
		if err != nil {
			return fmt.Errorf("Error when creating the closing condition")
		}

		closingCondition = cc
	case "sun_based":
		cc, err := sunbased.NewSunBasedCondition(input.ClosingCondition.Value, viper.GetFloat64("coop.latitude"), viper.GetFloat64("coop.longitude"))
		if err != nil {
			return fmt.Errorf("Error when creating the closing condition")
		}

		closingCondition = cc
	default:
		return fmt.Errorf("closing mode is incorrect: %s", input.ClosingCondition.Mode)
	}

	// Update the coop
	service.coop.Status = input.Status
	service.coop.IsAutomatic = input.IsAutomatic
	service.coop.OpeningCondition = openingCondition
	service.coop.ClosingCondition = closingCondition

	return nil
}

// Open the Coop
func (service *coopService) Open() error {
	// Get the status of the coop
	status := service.coop.Status

	// Check if coop is opened
	if status == coop.Opened {
		return fmt.Errorf(ErrCoopOpened)
	}

	// Check if coop is opening
	if status == coop.Opening {
		return fmt.Errorf(ErrCoopOpening)
	}

	return service.coop.Open()
}

// Close the Coop
func (service *coopService) Close() error {
	// Get the status of the coop
	status := service.coop.Status

	// Check if coop is closed
	if status == coop.Closed {
		return fmt.Errorf(ErrCoopClosed)
	}

	// Check if coop is closing
	if status == coop.Closing {
		return fmt.Errorf(ErrCoopClosing)
	}

	return service.coop.Close()
}

// Close the Coop
func (service *coopService) Stop() error {
	return service.coop.Stop()
}

func coopTempFanHandler(tempInsideCoop float32) {
	coopfan := fan.NewFan(uint8(viper.GetInt("fan.pin")))
	insideTempLimit := viper.GetInt("fan.temp_limit")

	if tempInsideCoop > float32(insideTempLimit) {
		coopfan.On()
	} else {
		coopfan.Off()
	}
}

func (service *coopService) GetTemp() (float32, float32, float32, float32, error) {
	InsideTemp, InsideHumidity, err := service.InTempSensor.ReadTemp()
    if err != nil {
        return -1,-1,-1,-1,fmt.Errorf("Error reading temperature: %s\n", err.Error())
    }

	go coopTempFanHandler(InsideTemp)

	OutsideTemp, OutsideHumidity, err := service.OutTempSensor.ReadTemp()
    if err != nil {
        return -1,-1,-1,-1,fmt.Errorf("Error reading temperature: %s\n", err.Error())
    }

	return InsideTemp, InsideHumidity, OutsideTemp, OutsideHumidity, nil
}
