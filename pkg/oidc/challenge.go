package oidc

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/giantswarm/microerror"
)

func GenerateChallenge() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", microerror.Mask(err)
	}

	state := base64.StdEncoding.EncodeToString(b)

	return state, nil
}
