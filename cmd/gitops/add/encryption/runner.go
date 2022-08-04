package encryption

import (
	"context"
	"fmt"
	"io"
	//"strconv"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"
	//"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/creator"
	//"github.com/giantswarm/kubectl-gs/internal/gitops/structure"
)

const (
	keyName = "Max Mustermann"
	email   = ""
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
	/*config := structure.McConfig{
		Name:           r.flag.Name,
		RepositoryName: r.flag.RepositoryName,
	}

	creatorConfig, err := structure.NewManagementCluster(config)
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
	}*/

	ecKey, err := crypto.GenerateKey(keyName, email, "x25519", 0)
	if err != nil {
		return err
	}
	fmt.Println(ecKey.GetArmoredPublicKey())
	fmt.Println(ecKey.GetFingerprint())

	return nil
}
