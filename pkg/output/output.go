package output

const (
	TypeDefault        = ""
	TypeJSON           = "json"
	TypeYAML           = "yaml"
	TypeName           = "name"
	TypeGoTemplate     = "go-template"
	TypeGoTemplateFile = "go-template-file"
	TypeTemplate       = "template"
	TypeTemplateFile   = "template-file"
	TypeJsonPath       = "jsonpath"
	TypeJsonPathFile   = "jsonpath-file"
)

func IsOutputDefault(output *string) bool {
	return output == nil || *output == TypeDefault
}

func IsOutputName(output *string) bool {
	return output == nil || *output == TypeName
}
