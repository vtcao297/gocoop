//go:build linux
// +build linux

package l298n

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stianeikeland/go-rpio/v4"
	"github.com/spf13/viper"
)

// Forward turns the motor forward.
func (motor *l298n) Forward(ctx context.Context) error {
	logrus.Infoln("Turn motor forward")

	// Access the pins
	err := rpio.Open()
	if err != nil {
		return fmt.Errorf("error while accessing the pins: %s", err)
	}
	defer rpio.Close()

	// Configure limit switch
	openDoorLimitPin := viper.GetInt("door.stoplimit.open_pin")
	limitPin := rpio.Pin(openDoorLimitPin)
	limitPin.Input()

	// Open the pinInput1 and set OUT mode
	logrus.WithFields(logrus.Fields{
		"pin_number": motor.pinInput1,
		"mode":       "out",
	}).Infoln("Open the pin")
	pinInput1 := rpio.Pin(motor.pinInput1)
	pinInput1.Output()

	// Open the pinInput2 and set OUT mode
	logrus.WithFields(logrus.Fields{
		"pin_number": motor.pinInput2,
		"mode":       "out",
	}).Infoln("Open the pin")
	pinInput2 := rpio.Pin(motor.pinInput2)
	pinInput2.Output()

	// Open the pinEnable1 and set OUT mode
	logrus.WithFields(logrus.Fields{
		"pin_number": motor.pinEnable1,
		"mode":       "out",
	}).Infoln("Open the pin")
	pinEnable1 := rpio.Pin(motor.pinEnable1)
	pinEnable1.Output()
	pinEnable1.Pwm()
	pinEnable1.Freq(1000)
	pinEnable1.DutyCycle(0, 100)
	
	// Set the motor rotation
	logrus.Infoln("Set the motor rotation")
	pinInput1.High()
	pinInput2.Low()

	// Enable the motor
	logrus.Infoln("Start the motor")
	//pinEnable1.High()
	pinEnable1.DutyCycle(60, 100)

	// Wait
	until, _ := ctx.Deadline()
	logrus.Infoln("Wait until", until)
	for {
		select {
		case <-ctx.Done():
			break;
		default:
			if limitPin.Read() == rpio.Low {
				logrus.Infoln("Hit the Door Top Limit switch")
				break;
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	
	// Disable the motor
	logrus.Infoln("Stop the motor")
	pinEnable1.DutyCycle(0, 100)
	//pinEnable1.StopPwm()
	//pinEnable1.Low()

	logrus.Infoln("Door has been stopped")

	return nil
}

// Backward turns the motor backward.
func (motor *l298n) Backward(ctx context.Context) error {
	logrus.Infoln("Turn motor backward")

	// Access the pins
	err := rpio.Open()
	if err != nil {
		return fmt.Errorf("error while accessing the pins: %s", err)
	}
	defer rpio.Close()

	// Configure limit switch
	closeDoorLimitPin := viper.GetInt("door.stoplimit.close_pin")
	limitPin := rpio.Pin(closeDoorLimitPin)
	limitPin.Input()

	// Open the pinInput1 and set OUT mode
	logrus.WithFields(logrus.Fields{
		"pin_number": motor.pinInput1,
		"mode":       "out",
	}).Infoln("Open the pin")
	pinInput1 := rpio.Pin(motor.pinInput1)
	pinInput1.Output()

	// Open the pinInput2 and set OUT mode
	logrus.WithFields(logrus.Fields{
		"pin_number": motor.pinInput2,
		"mode":       "out",
	}).Infoln("Open the pin")
	pinInput2 := rpio.Pin(motor.pinInput2)
	pinInput2.Output()

	// Open the pinEnable1 and set OUT mode
	logrus.WithFields(logrus.Fields{
		"pin_number": motor.pinEnable1,
		"mode":       "out",
	}).Infoln("Open the pin")
	pinEnable1 := rpio.Pin(motor.pinEnable1)
	pinEnable1.Output()
	pinEnable1.Pwm()
	pinEnable1.Freq(1000)
	pinEnable1.DutyCycle(0, 100)

	// Set the motor rotation
	logrus.Infoln("Set the motor rotation")
	pinInput1.Low()
	pinInput2.High()

	// Enable the motor
	logrus.Infoln("Start the motor")
	//pinEnable1.High()
	pinEnable1.DutyCycle(60, 100)


	// Wait
	until, _ := ctx.Deadline()
	logrus.Infoln("Wait until", until)
	for {
		select {
		case <-ctx.Done():
			break;
		default:
			limitPin := rpio.Pin(closeDoorLimitPin)
			if limitPin.Read() == rpio.Low {
				logrus.Infoln("Hit the Door Bottom Limit switch")
				break;
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Disable the motor
	logrus.Infoln("Stop the motor")
	pinEnable1.DutyCycle(0, 100)
	//pinEnable1.StopPwm()
	//pinEnable1.Low()

	logrus.Infoln("Motor is stopped")

	return nil
}

// Stop the motor.
func (motor *l298n) Stop() error {
	logrus.Infoln("Stopping the motor")

	// Access the pins
	err := rpio.Open()
	if err != nil {
		return fmt.Errorf("error while accessing the pins: %s", err)
	}
	defer rpio.Close()

	// Open the pinEnable1 and set OUT mode
	logrus.WithFields(logrus.Fields{
		"pin_number": motor.pinEnable1,
		"mode":       "out",
	}).Infoln("Open the pin")
	pinEnable1 := rpio.Pin(motor.pinEnable1)
	pinEnable1.Output()

	// Set pinEnable1 to LOW
	logrus.Infoln("Stop the motor")
	pinEnable1.Low()

	logrus.Infoln("Motor has been stopped")

	return nil
}
