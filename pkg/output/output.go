package output

const (
	OutputDefault        = ""
	OutputJSON           = "json"
	OutputYAML           = "yaml"
	OutputName           = "name"
	OutputGoTemplate     = "go-template"
	OutputGoTemplateFile = "go-template-file"
	OutputTemplate       = "template"
	OutputTemplateFile   = "template-file"
	OutputJsonPath       = "jsonpath"
	OutputJsonPathFile   = "jsonpath-file"
)

func IsOutputDefault(output *string) bool {
	return output == nil || *output == OutputDefault
}
