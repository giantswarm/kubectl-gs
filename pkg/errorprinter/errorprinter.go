package errorprinter

import (
	"strings"
	"unicode"
)

const (
	prefix = "Error: "
)

type ErrorPrinter struct {
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
	messageBuilder.WriteString(prefix)

	{
		title := errorRows[0]
		if title == "" {
			return ""
		}

		title = ep.formatTitle(title)
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
		title = string(tmpTitle)
	}

	return title
}
