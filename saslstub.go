//go:build !sasl
// +build !sasl

package mgo

import (
	"errors"
)

func saslNew(cred Credential, host string) (saslStepper, error) {
	return nil, errors.New("SASL support not enabled during build (-tags sasl)")
}
