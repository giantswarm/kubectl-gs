package common

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/kubectl-gs/v5/internal/gitops/filesystem/creator"
)

// AppendFromTemplate add files from the given template to the
// given directory by appending to the file objects list.
func AppendFromTemplate(dst *[]*creator.FsObject, path string, templates func() []Template, config interface{}) error {
	fileObjects, err := addFilesFromTemplate(path, templates, config)
	if err != nil {
		return microerror.Mask(err)
	}

	*dst = append(*dst, fileObjects...)

	return nil
}

// addFilesFromTemplate add files from the given template to the
// given directory.
func addFilesFromTemplate(path string, templates func() []Template, config interface{}) ([]*creator.FsObject, error) {
	var err error

	fsObjects := make([]*creator.FsObject, 0)
	for _, t := range templates() {
		// First, we template the name of the file
		nameTemplate := template.Must(template.New("name").Parse(t.Name))
		var name bytes.Buffer
		err = nameTemplate.Execute(&name, config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		contentTemplate := template.Must(template.New("files").Funcs(sprig.TxtFuncMap()).Parse(t.Data))

		// Next, we template the file content
		var content bytes.Buffer
		err = contentTemplate.Execute(&content, config)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		// Instead of conditioning in the `template` package and not returning an
		// empty file, we return empty files and condition here, effectively
		// removing them from the structure set.
		if len(content.Bytes()) <= 1 {
			continue
		}

		file := name.String()
		if path != "" {
			file = fmt.Sprintf("%s/%s", path, file)
		}

		fsObjects = append(
			fsObjects,
			creator.NewFsObject(
				file,
				content.Bytes(),
				t.Permission,
			),
		)
	}

	return fsObjects, nil
}
