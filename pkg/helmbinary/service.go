package helmbinary

import (
	"context"
	"io/ioutil"
	"net/url"
	"os/exec"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
)

var _ Interface = &Service{}

// Config represent the input parameters that New takes to produce a valid helmbinary Service.
type Config struct {
	Client *client.Client
}

// Service is the object we'll hang the helmbinary methods on.
type Service struct {
	client *client.Client
}

// New returns a new helmbinary Service.
func New(config Config) (Interface, error) {
	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	s := &Service{
		client: config.Client,
	}

	return s, nil
}

// Pull uses the helm binary to fetch a Chart and extract it to a temporary directory
// Why not just download it directly? There are actually a lot of little security
// details that go into unpacking this tarball. Checkout https://github.com/helm/helm/blob/master/pkg/chart/loader/archive.go#L101
func (s *Service) Pull(ctx context.Context, options PullOptions) (tmpDir string, err error) {

	parsedURL, err := url.Parse(options.URL)
	if err != nil {
		return "", microerror.Maskf(argumentError, "unable to parse %q. Is it a valid URL?", options.URL)
	}

	tmpDir, err = ioutil.TempDir("", "kubectl-gs-validate-apps-")
	if err != nil {
		return "", microerror.Mask(err)
	}

	// nosec is applied here, the URL in question can be traced back to the
	// app catalog index entry. We verified this is actually a URL earlier in this
	// method.
	cmd := exec.Command("helm", "pull", parsedURL.String(), "--untar", "-d", tmpDir) // #nosec G204
	err = cmd.Run()
	if err != nil {
		return "", microerror.Maskf(commandError, "failed to execute: %s, %s", cmd.String(), err.Error())
	}

	return tmpDir, nil
}
