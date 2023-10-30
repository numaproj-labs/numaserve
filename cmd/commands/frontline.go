package commands

import (
	"context"

	"github.com/numaproj-labs/numaserve/pkg/frontline"
	"github.com/spf13/cobra"
)

func NewFrontlineCommand() *cobra.Command {
	var (
		insecure   bool
		port       int
		sync       bool
		configpath string
		backendURL string
	)

	command := &cobra.Command{
		Use:   "frontline",
		Short: "Start a Front line server",
		Run: func(cmd *cobra.Command, args []string) {
			if !cmd.Flags().Changed("port") && insecure {
				port = 8080
			}
			server := frontline.NewFrontLineService(port, backendURL)
			server.Run(context.Background())
		},
	}
	command.Flags().BoolVar(&insecure, "insecure", false, "Whether to disable TLS, defaults to false.")
	command.Flags().IntVarP(&port, "port", "p", 8443, "Port to listen on, defaults to 8443 or 8080 if insecure is set")
	command.Flags().BoolVar(&sync, "sync", false, "Start Frontline server with Synchronous.")
	command.Flags().StringVar(&backendURL, "backendurl", "", "Backend Service URL")
	command.Flags().StringVar(&configpath, "configpath", "/tmp/config/config.json", "Config file path.")
	return command
}
