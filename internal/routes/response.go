package routes

import "time"

// ConditionResponse is the response for a condition.
type ConditionResponse struct {
	Mode  string
	Value string
}

// CoopResponse is the response for coop.
type CoopResponse struct {
	OpeningCondition ConditionResponse
	ClosingCondition ConditionResponse
	Latitude         float64
	Longitude        float64
	Status           string
	IsAutomatic      bool
	NextOpeningTime  time.Time
	NextClosingTime  time.Time
	OutsideTemp      float32
	OutsideHumidity	 float32
	InsideTemp       float32
	InsideHumidity	 float32
	Cameras          map[string]string
}
