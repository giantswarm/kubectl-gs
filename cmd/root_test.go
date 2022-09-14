package cmd

import (
	"strings"
	"testing"

	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestRootCommand(t *testing.T) {

	t.Run("case 0: Conflicting flag names", func(t *testing.T) {
		loggerCfg := micrologger.Config{}
		logger, err := micrologger.New(loggerCfg)

		if err != nil {
			t.Fatal(err)
		}

		fs := afero.NewOsFs()

		rootCmd, err := New(Config{Logger: logger, FileSystem: fs})

		if err != nil {
			t.Fatal(err)
		}

		globalFlags := getGlobalFlags(rootCmd)

		require.Greater(t, len(globalFlags), 0)

		flags := getFlagsFromCommands(rootCmd, []string{})

		for cmdName, cmdFlags := range flags {
			lookForShadowedFlags(t, cmdName, cmdFlags, globalFlags)
		}
	})
}

func getGlobalFlags(cmd *cobra.Command) []string {
	var globalFlags []string
	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		globalFlags = append(globalFlags, flag.Name)
	})
	return globalFlags
}

func getFlagsFromCommands(cmd *cobra.Command, cmdPath []string) map[string][]string {
	var flags []string
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if flag.Deprecated == "" {
			flags = append(flags, flag.Name)
		}
	})

	flagsByCmd := make(map[string][]string)
	flagsByCmd[strings.Join(append(cmdPath, cmd.Use), " ")] = flags

	for _, subCmd := range cmd.Commands() {
		for subCmdName, subCmdFlags := range getFlagsFromCommands(subCmd, append(cmdPath, cmd.Use)) {
			flagsByCmd[subCmdName] = subCmdFlags
		}
	}

	return flagsByCmd
}

func lookForShadowedFlags(t *testing.T, cmdName string, cmdFlags []string, globalFlags []string) {
	var shadowedFlags []string
	for _, cmdFlag := range cmdFlags {
		for _, globalFlag := range globalFlags {
			if globalFlag == cmdFlag {
				shadowedFlags = append(shadowedFlags, cmdFlag)
			}
		}
	}
	if len(shadowedFlags) > 0 {
		t.Errorf("'%s' has shadowed global flags ['%s']\n", cmdName, strings.Join(shadowedFlags, "', '"))
	}
}
