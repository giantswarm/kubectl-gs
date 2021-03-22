package template

import (
	"embed"
	"io"

	"github.com/giantswarm/microerror"
)

var (
	//go:embed internal/*
	templates embed.FS
)

func GetSuccessHTMLTemplateReader() (io.ReadSeeker, error) {
	f, err := templates.Open("internal/sso_complete.html")
	if err != nil {
		return nil, microerror.Mask(err)
	}
	defer f.Close()

	return f.(io.ReadSeeker), nil
}

func GetFailedHTMLTemplateReader() (io.ReadSeeker, error) {
	f, err := templates.Open("internal/sso_failed.html")
	if err != nil {
		return nil, microerror.Mask(err)
	}
	defer f.Close()

	return f.(io.ReadSeeker), nil
}
