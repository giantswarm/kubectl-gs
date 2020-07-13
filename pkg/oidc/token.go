package oidc

import (
	"github.com/giantswarm/microerror"
	"golang.org/x/oauth2"
)

func ConvertTokenToRawIDToken(token *oauth2.Token) (string, error) {
	// Extract the raw ID Token.
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return "", microerror.Mask(cannotDecodeTokenError)
	}

	return rawIDToken, nil
}
