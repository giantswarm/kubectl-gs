package provider

import (
	"strings"
)

const (
	naValue = "n/a"
)

func formatCondition(condition string) string {
	return strings.ToUpper(condition)
}
