package helmbinary

import (
	"context"
	"io/ioutil"
	"os/exec"

	"github.com/giantswarm/kubectl-gs/pkg/data/client"
	"github.com/giantswarm/microerror"
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
	tmpDir, err = ioutil.TempDir("", "kubectl-gs-validate-apps-")
	if err != nil {
		return "", microerror.Mask(err)
	}

	cmd := exec.Command("helm", "pull", options.URL, "--untar", "-d", tmpDir)
	_, err = cmd.CombinedOutput()
	if err != nil {
		return "", microerror.Maskf(commandError, "failed to execute: %s, %s", cmd.String(), err.Error())
	}

	return tmpDir, nil
}
