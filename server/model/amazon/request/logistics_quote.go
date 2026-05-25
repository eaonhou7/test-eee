package request

import (
	"errors"
)

type LogisticsQuoteForm struct {
	WeightKG        float64  `form:"weight_kg" json:"weight_kg"`
	ContainsBattery bool     `form:"contains_battery" json:"contains_battery"`
	Platform        string   `form:"platform" json:"platform"`
	LengthCM        *float64 `form:"length_cm" json:"length_cm,omitempty"`
	WidthCM         *float64 `form:"width_cm" json:"width_cm,omitempty"`
	HeightCM        *float64 `form:"height_cm" json:"height_cm,omitempty"`
}

func (f LogisticsQuoteForm) Validate() error {
	if f.WeightKG <= 0 {
		return errors.New("weight_kg must be greater than 0")
	}
	hasAnyDimension := f.LengthCM != nil || f.WidthCM != nil || f.HeightCM != nil
	hasAllDimensions := f.LengthCM != nil && f.WidthCM != nil && f.HeightCM != nil
	if hasAnyDimension && !hasAllDimensions {
		return errors.New("length_cm, width_cm, height_cm must be provided together")
	}
	if hasAllDimensions {
		if *f.LengthCM <= 0 || *f.WidthCM <= 0 || *f.HeightCM <= 0 {
			return errors.New("dimensions must be greater than 0")
		}
	}
	return nil
}
