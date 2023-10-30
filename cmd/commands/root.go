package commands

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "numaserve",
	Short: "numaserve CLI",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func init() {
	rootCmd.AddCommand(NewFrontlineCommand())
	rootCmd.AddCommand(NewSinkCommand())
	rootCmd.AddCommand(NewControllerCommand())
	rootCmd.AddCommand(NewInitCommand())
}
