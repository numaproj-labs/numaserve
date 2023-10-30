package commands

import (
	"context"
	"github.com/numaproj-labs/numaserve/pkg/sink"
	sinksdk "github.com/numaproj/numaflow-go/pkg/sinker"
	"github.com/numaproj/numaflow/pkg/shared/logging"
	"github.com/spf13/cobra"
	"log"
)

func NewSinkCommand() *cobra.Command {
	logger := logging.NewLogger().Named("http-sink")
	hs := sink.HttpSink{Logger: logger}
	command := &cobra.Command{
		Use:   "sink",
		Short: "Start a UDF sink",
		Run: func(cmd *cobra.Command, args []string) {

			//creating http client
			hs.CreateHTTPClient()
			hs.Logger.Info("HTTP Sink starting successfully with args %v", hs)
			err := sinksdk.NewServer(&hs).Start(context.Background())
			if err != nil {
				log.Panic("Failed to start sink function server: ", err)
			}
		},
	}

	command.Flags().StringVar(&hs.URL, "url", "", "URL")
	command.Flags().StringVar(&hs.Method, "method", "POST", "HTTP Method")
	command.Flags().IntVar(&hs.Retries, "retries", 3, "Request Retries")
	command.Flags().IntVar(&hs.Timeout, "timeout", 30, "Request Timeout in seconds")
	command.Flags().BoolVar(&hs.SkipInsecure, "insecure", false, "Skip TLS verify")
	command.Flags().BoolVar(&hs.DropIfError, "dropIfError", false, "Messages will drop after retry")
	// Parse the flag

	return command
}
