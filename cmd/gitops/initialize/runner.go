package initialize

import (
	"context"
	"fmt"
	"io"
	//"os"
	"strconv"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	//"github.com/ProtonMail/gopenpgp/v2/helper"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	//"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/internal/gitops/structure"
	//commonkey "github.com/giantswarm/kubectl-gs/internal/key"
)

const (
	keyName = "Max Mustermann"
	email   = "max.mustermann@example.com"
	rsaBits = 2048
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer
}

func (r *runner) Run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	err := r.flag.Validate()
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.run(ctx, cmd, args)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	var err error

	creatorConfig, err := structure.Initialize()
	if err != nil {
		return microerror.Mask(err)
	}

	creatorConfig.Stdout = r.stdout

	dryRunFlag := cmd.InheritedFlags().Lookup("dry-run")
	if dryRunFlag != nil {
		creatorConfig.DryRun, _ = strconv.ParseBool(dryRunFlag.Value.String())
	}

	localPathFlag := cmd.InheritedFlags().Lookup("local-path")
	if localPathFlag != nil {
		creatorConfig.Path = localPathFlag.Value.String()
	}

	creator := creator.NewCreator(*creatorConfig)

	err = creator.Create()
	if err != nil {
		return microerror.Mask(err)
	}

	ecKey, err := crypto.GenerateKey(name, email, "x25519", 0)
	fmt.Println(ecKey.GetArmoredPublicKey())

	return nil
}
