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
	err := mcDir.AddFile(
		fileName(config.Name),
		kustomizationManifest(
			kustomizationName(config.Name, ""),
			kustomizationPath(config.Name, "", ""),
			config.RepositoryName,
			config.ServiceAccount,
			config.RefreshInterval,
			config.RefreshTimeout),
	)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	mcDir.AddDirectory(newDir(directorySecrets))
	mcDir.AddDirectory(newDir(directorySOPSPublicKeys))
	mcDir.AddDirectory(newDir(directoryOrganizations))

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
		file.Write(path)
	}

	for _, dir := range d.dirs {
		err = dir.Write(path)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (f *File) Write(path string) error {
	err := os.WriteFile(fmt.Sprintf("%s/%s", path, f.name), f.data, 0600)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
