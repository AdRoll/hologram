package main

import (
	"fmt"
	"github.com/AdRoll/hologram/log"
	"github.com/AdRoll/hologram/protocol"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	rootCmd.AddCommand(meCmd)
}

var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Assumes the default role",
	Run: func(cmd *cobra.Command, args []string) {
		err := me()
		if err != nil {
			log.Errorf("%s", err)
			os.Exit(1)
		}
	},
}

func me() error {
	response, err := request(&protocol.AgentRequest{
		GetUserCredentials: &protocol.GetUserCredentials{},
	})

	if err != nil {
		return err
	}

	if response.GetFailure() != nil {
		return fmt.Errorf("error from server: %s", response.GetFailure().GetErrorMessage())
	}

	if response.GetSuccess() != nil {
		log.Info("Successfully loaded credentials for you")
		return nil
	}

	return fmt.Errorf("unexpected response type: %v", response)
}
