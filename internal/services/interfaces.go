package services

import (
	"github.com/fallais/gocoop/pkg/coop"
)

//------------------------------------------------------------------------------
// Interfaces
//------------------------------------------------------------------------------

// CoopService is the interface
type CoopService interface {
	GetCoop() *coop.Coop
	Update(CoopUpdateRequest) error
	Open() error
	Close() error
	Stop() error
	GetTemp() (float32, float32, float32, float32, error)
}
