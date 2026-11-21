package org

import (
	"context"
	"io"
	"path/filepath"
	"strconv"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/filesystem/creator"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/key"
	"github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/common"
	structure "github.com/giantswarm/kubectl-gs/v6/internal/gitops/structure/organization"
)

type runner struct {
	flag   *flag
	fs     afero.Fs
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
	config := common.StructureConfig{
		ManagementCluster: r.flag.ManagementCluster,
		Organization:      r.flag.Name,
	}

	localPath := inheritedFlag(cmd, "local-path")

	var err error
	config.RepositoryName, err = r.repositoryName(localPath)
	if err != nil {
		return microerror.Mask(err)
	}

	creatorConfig, err := structure.NewOrganization(config)
	if err != nil {
		return microerror.Mask(err)
	}

	creatorConfig.Stdout = r.stdout

	dryRunFlag := cmd.InheritedFlags().Lookup("dry-run")
	if dryRunFlag != nil {
		creatorConfig.DryRun, _ = strconv.ParseBool(dryRunFlag.Value.String())
	}

	creatorConfig.Fs = r.fs
	creatorConfig.Path = localPath

	creator := creator.NewCreator(*creatorConfig)
	err = creator.Create()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// repositoryName returns the GitRepository the organisation's namespace
// Kustomization reads: --repository-name, or else the one the management
// cluster's Kustomization reads.
func (r *runner) repositoryName(localPath string) (string, error) {
	if r.flag.RepositoryName != "" {
		return r.flag.RepositoryName, nil
	}

	mcDir := key.BaseDirPath(r.flag.ManagementCluster, "", "")
	path := filepath.Join(localPath, mcDir, key.FluxKustomizationFileName(r.flag.ManagementCluster))

	var mc struct {
		Spec struct {
			SourceRef struct {
				Name string `json:"name"`
			} `json:"sourceRef"`
		} `json:"spec"`
	}
	data, err := afero.ReadFile(r.fs, path)
	if err == nil {
		err = yaml.Unmarshal(data, &mc)
	}
	if err != nil || mc.Spec.SourceRef.Name == "" {
		return "", microerror.Maskf(invalidFlagsError, "--%s is not set, and the management cluster's GitRepository can't be read from %s", flagRepositoryName, path)
	}

	return mc.Spec.SourceRef.Name, nil
}

// inheritedFlag returns the value of a flag of the gitops command, or an
// empty string.
func inheritedFlag(cmd *cobra.Command, name string) string {
	f := cmd.InheritedFlags().Lookup(name)
	if f == nil {
		return ""
	}

	return f.Value.String()
}
