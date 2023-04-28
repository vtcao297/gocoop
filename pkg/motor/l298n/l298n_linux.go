//go:build linux
// +build linux

package l298n

import (
	"context"
	"fmt"
	"time"
	"errors"

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
	pinEnable1.Freq(50 * 100)
	pinEnable1.DutyCycle(0, 100)
	
	// Set the motor rotation
	logrus.Infoln("Set the motor rotation")
	pinInput1.High()
	pinInput2.Low()

	// Enable the motor
	motorDutyCycle := uint32(viper.GetInt("door.motor.pwm_open_dutycycle"))
	logrus.Infof("Start the motor: PWM DutyCycle=%v", motorDutyCycle)
	pinEnable1.DutyCycle(motorDutyCycle, 100)

	// Wait
	until, isDeadlineSet := ctx.Deadline()
	if isDeadlineSet == true {
		logrus.Infoln("Wait until", until)
	}

	for {
		select {
		case <-ctx.Done():
			logrus.Infoln("Motor stopped due to context cancellation")
			pinEnable1.DutyCycle(0, 100)
			logrus.Infoln("Motor is stopped")
			return ctx.Err()
		default:
			if limitPin.Read() == rpio.Low {
				logrus.Infoln("Hit the Door Top Limit switch")
				pinEnable1.DutyCycle(0, 100)
				logrus.Infoln("Motor is stopped")
				return nil
			}

			if time.Now().After(until) {
				logrus.Infoln("Motor timeout, need to shutdown motor")
				pinEnable1.DutyCycle(0, 100)
				logrus.Infoln("Motor is stopped")
				return errors.New("motor timeout")
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
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
	pinEnable1.Freq(50 * 100)
	pinEnable1.DutyCycle(0, 100)

	// Set the motor rotation
	logrus.Infoln("Set the motor rotation")
	pinInput1.Low()
	pinInput2.High()

	// Enable the motor
	motorDutyCycle := uint32(viper.GetInt("door.motor.pwm_close_dutycycle"))
	logrus.Infof("Start the motor: PWM DutyCycle=%v", motorDutyCycle)
	pinEnable1.DutyCycle(motorDutyCycle, 100)

	// Wait
	until, isDeadlineSet := ctx.Deadline()
	if isDeadlineSet == true {
		logrus.Infoln("Wait until", until)
	}

	for {
		select {
		case <-ctx.Done():
			logrus.Infoln("Motor stopped due to context cancellation")
			pinEnable1.DutyCycle(0, 100)
			logrus.Infoln("Motor is stopped")
			return ctx.Err()
		default:
			if limitPin.Read() == rpio.Low {
				logrus.Infoln("Hit the Door Bottom Limit switch")
				pinEnable1.DutyCycle(0, 100)
				logrus.Infoln("Motor is stopped")
				return nil
			}

			if time.Now().After(until) {
				logrus.Infoln("Motor timeout, need to shutdown motor")
				pinEnable1.DutyCycle(0, 100)
				logrus.Infoln("Motor is stopped")
				return errors.New("motor timeout")
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
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

	logrus.Infoln("Stop the motor")
	pinEnable1.Pwm()
	pinEnable1.Freq(50 * 100)
	pinEnable1.DutyCycle(0, 100)
	logrus.Infoln("Motor has been stopped")

	return nil
}
