package geocode

import (
	"context"
	"errors"
	"fmt"

	"billiard_master/internal/model"
)

const (
	ErrorKindAuth            = "auth"
	ErrorKindTemporary       = "temporary"
	ErrorKindInvalidResponse = "invalid_response"
)

const (
	GeocodeAccountStatusDisabled = model.GeocodeAccountStatusDisabled
	GeocodeAccountStatusEnabled  = model.GeocodeAccountStatusEnabled
)

var ErrNoAvailableAccount = errors.New("no available geocode account")

type Result struct {
	Longitude float64
	Latitude  float64
	Score     int
	Level     string
	RawCode   int
	Source    string
}

type Geocoder interface {
	Geocode(ctx context.Context, account model.GeocodeAccount, address string) (*Result, error)
}

type ProviderError struct {
	Kind    string
	Message string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("geocode %s error: %s", e.Kind, e.Message)
}

func NewProviderError(kind, message string) *ProviderError {
	return &ProviderError{
		Kind:    kind,
		Message: message,
	}
}
