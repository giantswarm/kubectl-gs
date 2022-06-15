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

func NewMcDir(config McConfig) *Dir {
	mcDir := newDir(config.Name)
	mcDir.AddFile(config.Name+".yaml", unstructured.Unstructured{
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
	mcDir.AddDirectory(newDir(secretsDirectory))
	mcDir.AddDirectory(newDir(sopsPublicKeysDirectory))
	mcDir.AddDirectory(newDir(organizationsDirectory))

	return mcDir
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
		err := os.WriteFile(fmt.Sprintf("%s/%s", path, file.name), file.data, 0644)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	for _, dir := range d.dirs {
		dir.Write(path)
	}

	return nil
}
