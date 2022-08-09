package encryption

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/cobra"

	"github.com/giantswarm/kubectl-gs/internal/gitops/encryption"
	"github.com/giantswarm/kubectl-gs/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/internal/gitops/structure"
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

func getKeyName(c structure.StructureConfig) string {
	if c.WorkloadCluster != "" {
		return fmt.Sprintf("'%s' Workload Cluster encryption", c.WorkloadCluster)
	} else if c.Organization != "" {
		return fmt.Sprintf("'%s' Organization encryption", c.Organization)
	}

	return fmt.Sprintf("'%s' Flux master", c.ManagementCluster)
}

func (r *runner) run(ctx context.Context, cmd *cobra.Command, args []string) error {
	config := structure.StructureConfig{
		ManagementCluster: r.flag.ManagementCluster,
		Organization:      r.flag.Organization,
		EncryptionTarget:  r.flag.Target,
		WorkloadCluster:   r.flag.WorkloadCluster,
	}

	if r.flag.Generate {
		keyName := getKeyName(config)

		keyPair, err := encryption.GenerateKeyPair(keyName)
		if err != nil {
			return microerror.Mask(err)
		}

		config.EncryptionKeyPair = keyPair
	}

	creatorConfig, err := structure.NewEncryption(config)
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

	return nil
}
