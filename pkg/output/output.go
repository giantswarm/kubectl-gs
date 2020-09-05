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
	TypeReport         = "report"
)

func IsOutputDefault(output *string) bool {
	return output == nil || *output == TypeDefault
}

func IsOutputName(output *string) bool {
	return output == nil || *output == TypeName
}

func IsOutputReport(output *string) bool {
	return *output == "report"
}
