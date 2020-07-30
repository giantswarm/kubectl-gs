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
	StackTraces   bool
}

type ErrorPrinter struct {
	disableColors bool
	stackTraces   bool
}

func New(c Config) *ErrorPrinter {
	ep := &ErrorPrinter{
		disableColors: c.DisableColors,
		stackTraces:   c.StackTraces,
	}

	return ep
}

func (ep *ErrorPrinter) Format(err error) string {
	message := microerror.Pretty(err, ep.stackTraces)
	if len(message) < 1 {
		return ""
	}

	var builder strings.Builder
	rows := strings.SplitN(message, "\n", 2)
	builder.WriteString(ep.formatTitle(rows[0]))

	if len(rows) > 1 {
		builder.WriteString("\n")
		builder.WriteString(rows[1])
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
