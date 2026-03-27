package usagetemplate

import (
	"github.com/spf13/cobra"
)

// Apply sets a custom usage template on cmd that shows command aliases
// alongside names in the "Available Commands" section.
//
// Cobra's default template only shows .Name for subcommands. This template
// uses .NameAndAliases instead, and computes padding based on the longest
// NameAndAliases string so that columns remain aligned.
func Apply(cmd *cobra.Command) {
	cobra.AddTemplateFunc("nameAndAliasesPadding", func(cmd *cobra.Command) int {
		padding := 11 // cobra's minNamePadding
		for _, sub := range cmd.Commands() {
			n := len(sub.NameAndAliases())
			if n > padding {
				padding = n
			}
		}
		return padding
	})

	cmd.SetUsageTemplate(usageTemplate)
}

// rpad is already provided by cobra as a template function,
// so we only need to supply the template itself.

// usageTemplate is cobra's default usage template with two changes:
// 1. .Name replaced by .NameAndAliases in the Available Commands section
// 2. .NamePadding replaced by (nameAndAliasesPadding .Parent) for correct alignment
var usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .NameAndAliases (nameAndAliasesPadding .Parent) }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .NameAndAliases (nameAndAliasesPadding .Parent) }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .NameAndAliases (nameAndAliasesPadding .Parent) }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
