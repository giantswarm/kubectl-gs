package template

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"text/template"

	"github.com/giantswarm/kubectl-gs/cmd/template/cluster/provider"
	nodepool "github.com/giantswarm/kubectl-gs/cmd/template/nodepool/provider"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

type runner struct {
	flag   *flag
	logger micrologger.Logger
	stdout io.Writer
	stderr io.Writer
}

type kgsConfigItem struct {
	ID       string
	Defaults string
}

type kgsConfig struct {
	Cluster            kgsConfigItem
	Machinedeployments []kgsConfigItem
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
	if r.flag.FromFile == "" {
		err := cmd.Help()
		if err != nil {
			return microerror.Mask(err)
		}
		return nil
	}

	fs := afero.NewOsFs()

	data, err := afero.ReadFile(fs, r.flag.FromFile)
	if err != nil {
		return microerror.Mask(err)
	}
	var c kgsConfig
	err = yaml.Unmarshal(data, &c)
	if err != nil {
		return microerror.Mask(err)
	}
	outputFolder := filepath.Dir(r.flag.FromFile)

	err = templateCluster(ctx, fs, c.Cluster.ID, c.Cluster.Defaults, filepath.Join(outputFolder, fmt.Sprintf("%s.yaml", c.Cluster.ID)))
	if err != nil {
		return microerror.Mask(err)
	}

	for _, md := range c.Machinedeployments {
		err = templateNodepool(ctx, fs, c.Cluster.ID, md.ID, md.Defaults, filepath.Join(outputFolder, fmt.Sprintf("%s.yaml", md.ID)))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	err = templateKustomization(fs, c, filepath.Join(outputFolder, "kustomization.yaml"))
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func templateCluster(ctx context.Context, fs afero.Fs, clusterID string, defaultsPath string, outputPath string) error {
	data, err := afero.ReadFile(fs, defaultsPath)
	if err != nil {
		return microerror.Mask(err)
	}
	var clusterConfig provider.ClusterCRsConfig
	err = yaml.Unmarshal(data, &clusterConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	f, err := fs.Create(outputPath)
	if err != nil {
		return microerror.Mask(err)
	}

	clusterConfig.Name = clusterID
	err = provider.WriteOpenStackTemplate(ctx, f, clusterConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func templateNodepool(ctx context.Context, fs afero.Fs, clusterID string, nodepoolID string, defaultsPath string, outputPath string) error {
	data, err := afero.ReadFile(fs, defaultsPath)
	if err != nil {
		return microerror.Mask(err)
	}
	var nodepoolConfig nodepool.NodePoolCRsConfig
	err = yaml.Unmarshal(data, &nodepoolConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	f, err := fs.Create(outputPath)
	if err != nil {
		return microerror.Mask(err)
	}

	nodepoolConfig.NodePoolID = nodepoolID
	nodepoolConfig.ClusterName = clusterID
	err = nodepool.WriteCAPOTemplate(ctx, f, nodepoolConfig)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func templateKustomization(fs afero.Fs, config kgsConfig, outputPath string) error {
	ids := []string{
		config.Cluster.ID,
	}
	for _, md := range config.Machinedeployments {
		ids = append(ids, md.ID)
	}

	t := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
{{range . }}
- {{.}}
{{end}}`

	f, err := fs.Create(outputPath)
	if err != nil {
		return microerror.Mask(err)
	}

	te := template.Must(template.New("resource").Parse(t))
	var buf bytes.Buffer
	// Template from our inputs.
	err = te.Execute(&buf, ids)
	if err != nil {
		return microerror.Mask(err)
	}
	// JSON to YAML.
	mutated, err := yaml.JSONToYAML(buf.Bytes())
	if err != nil {
		return microerror.Mask(err)
	}
	// Write the yaml to our file.
	_, err = f.Write(mutated)
	if err != nil {
		return microerror.Mask(err)
	}
	return nil
}
