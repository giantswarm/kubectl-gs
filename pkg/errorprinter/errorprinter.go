package errorprinter

import (
	"strings"

	"github.com/fatih/color"
	"github.com/giantswarm/microerror"
)

const (
	prefix = "Error: "
)

type Config struct {
	DisableColors bool
	StackTrace    bool
}

type ErrorPrinter struct {
	disableColors bool
	stackTrace    bool
}

func New(c Config) *ErrorPrinter {
	ep := &ErrorPrinter{
		disableColors: c.DisableColors,
		stackTrace:    c.StackTrace,
	}

	return ep
}

func (ep *ErrorPrinter) Format(err error) string {
	message := microerror.Pretty(err, ep.stackTrace)
	if len(message) < 1 {
		return ""
	}

	var builder strings.Builder
	rows := strings.SplitN(message, "\n", 2)
	builder.WriteString(ep.formatTitle(rows[0]))

	if len(rows) > 1 && len(rows[1]) > 0 {
		builder.WriteString("\n")
		builder.WriteString(ep.formatBody(rows[1]))
	}

	return builder.String()
}

func (ep *ErrorPrinter) formatTitle(title string) string {
	title = strings.TrimSuffix(title, "\n")
	title = prefix + title

	if !ep.disableColors {
		title = color.RedString(title)
	}

	return title
}

func (ep *ErrorPrinter) formatBody(body string) string {
	body = strings.TrimPrefix(body, "\n")

	return body
}
