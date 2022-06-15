package structure

import (
	"fmt"
	"os"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

func (d *Dir) AddDirectory(dir *Dir) {
	d.dirs = append(d.dirs, dir)
}

func (d *Dir) AddFile(name string, manifest unstructured.Unstructured) error {
	manifestRaw, err := yaml.Marshal(manifest.Object)
	if err != nil {
		return microerror.Mask(err)
	}

	d.files = append(d.files, &File{
		name: name,
		data: manifestRaw,
	})

	return nil
}

func NewMcDir(config McConfig) (*Dir, error) {
	mcDir := newDir(config.Name)
	err := mcDir.AddFile(config.Name+".yaml", unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fluxKustomizationAPIVersion,
			"kind":       fluxKustomizationKind,
			"metadata": map[string]string{
				"name":      fmt.Sprintf("%s-%s", config.Name, topLevelKustomizationSuffix),
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"interval":           config.RefreshInterval,
				"path":               fmt.Sprintf("./%s/%s", topLevelGitOpsDirectory, config.Name),
				"prune":              "false",
				"serviceAccountName": config.ServiceAccount,
				"sourceRef": map[string]string{
					"kind": config.RepositoryKind,
					"name": config.RepositoryName,
				},
				"timeout": config.RefreshTimeout,
			},
		},
	})
	if err != nil {
		return nil, microerror.Mask(err)
	}
	mcDir.AddDirectory(newDir(secretsDirectory))
	mcDir.AddDirectory(newDir(sopsPublicKeysDirectory))
	mcDir.AddDirectory(newDir(organizationsDirectory))

	return mcDir, nil
}

func (d *Dir) Print(path string) {
	path = fmt.Sprintf("%s/%s", path, d.name)

	fmt.Println(path)
	for _, file := range d.files {
		fmt.Printf("%s/%s\n", path, file.name)
		fmt.Println(string(file.data))
	}

	for _, dir := range d.dirs {
		dir.Print(path)
	}
}

func (d *Dir) Write(path string) error {
	var err error

	path = fmt.Sprintf("%s/%s", path, d.name)

	err = os.Mkdir(path, 0755)
	if err != nil {
		return microerror.Mask(err)
	}

	for _, file := range d.files {
		err := os.WriteFile(fmt.Sprintf("%s/%s", path, file.name), file.data, 0600)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	for _, dir := range d.dirs {
		err = dir.Write(path)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
