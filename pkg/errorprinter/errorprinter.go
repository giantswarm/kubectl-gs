package errorprinter

import (
	"strings"
	"unicode"

	"github.com/fatih/color"
)

const (
	prefix = "Error: "
)

type ErrorPrinter struct {
	DisableColors bool
}

func New() *ErrorPrinter {
	ep := &ErrorPrinter{}

	return ep
}

func (ep *ErrorPrinter) Format(pErr error) string {
	if pErr == nil {
		return ""
	}

	errorRows := strings.Split(pErr.Error(), "\n")

	var messageBuilder strings.Builder

	{
		title := ep.formatTitle(errorRows[0])
		messageBuilder.WriteString(title)
		messageBuilder.WriteString("\n")
	}

	{
		// Add subtitle, if it exists.
		for _, row := range errorRows[1:] {
			messageBuilder.WriteString(row)
			messageBuilder.WriteString("\n")
		}
	}

	return messageBuilder.String()
}

func (ep *ErrorPrinter) formatTitle(title string) string {
	if title == "" {
		return ""
	}

	microerrorTypeLastCharIdx := strings.Index(title, ":") + 2
	if microerrorTypeLastCharIdx > 1 && len(title) > microerrorTypeLastCharIdx {
		// Remove the microerror type from the error message, if
		// there's a custom message available after it.
		title = title[microerrorTypeLastCharIdx:]
	} else {
		// Remove the 'error' suffix from errors that end with it.
		// This is usually the case with microerrors that don't have
		// any additional message.
		title = strings.TrimSuffix(title, " error")
	}

	{
		// Capitalize first letter.
		tmpTitle := []rune(title)
		tmpTitle[0] = unicode.ToUpper(tmpTitle[0])

		// Add 'Error: ' prefix.
		tmpTitle = append([]rune(prefix), tmpTitle...)

		title = string(tmpTitle)
	}

	if !ep.DisableColors {
		title = color.RedString(title)
	}

	return title
}
