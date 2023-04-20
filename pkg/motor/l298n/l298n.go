package l298n

import (
	"github.com/fallais/gocoop/pkg/motor"
)

//------------------------------------------------------------------------------
// Structure
//------------------------------------------------------------------------------

// l298n is a motor driver.
type l298n struct {
	pinInput1  int
	pinInput2  int
	pinEnable1 int
	pinInput3  int
	pinInput4  int
	pinEnable2 int
}

//------------------------------------------------------------------------------
// Factory
//------------------------------------------------------------------------------

// Newl298n returns a new l298n.
func NewL298N(pinInput1, pinInput2, pinEnable1 int) motor.Motor {
	return &l298n{
		pinInput1:  pinInput1,
		pinInput2:  pinInput2,
		pinEnable1: pinEnable1,
	}
}
