package fan

import (
	"github.com/stianeikeland/go-rpio/v4"
	"github.com/sirupsen/logrus"
)

// Fan ...
type Fan struct {
	pin rpio.Pin
}

// NewFan ...
func NewFan(pin uint8) *Fan {
	f := &Fan{
		pin: rpio.Pin(pin),
	}
	f.pin.Output()
	f.pin.Low()
	return f
}

// On ...
func (f *Fan) On() {
	logrus.Infoln("Fan is turned on")
	f.pin.High()
}

// Off ...
func (f *Fan) Off() {
	logrus.Infoln("Fan is turned off")
	f.pin.Low()
}