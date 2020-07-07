package template

import (
	"io"

	"github.com/giantswarm/microerror"
	"github.com/markbates/pkger"
)

func GetSuccessHTMLTemplateReader() (io.ReadSeeker, error) {
	f, err := pkger.Open("/cmd/login/template/internal/sso_complete.html")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f, nil
}

func GetFailedHTMLTemplateReader() (io.ReadSeeker, error) {
	f, err := pkger.Open("/cmd/login/template/internal/sso_failed.html")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return f, nil
}
