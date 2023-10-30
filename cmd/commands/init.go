package commands

import (
	downloader "github.com/numaproj-labs/numaserve/pkg/init"
	"github.com/spf13/cobra"
)

func NewInitCommand() *cobra.Command {

	command := &cobra.Command{
		Use:   "init",
		Short: "Start a Model Downloader",
		Run: func(cmd *cobra.Command, args []string) {
			downloader.DownloadModel()
		},
	}
	return command
}
