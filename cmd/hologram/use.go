package main

import (
	"fmt"
	"github.com/AdRoll/hologram/log"
	"github.com/AdRoll/hologram/protocol"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	rootCmd.AddCommand(useCmd)
}

var useCmd = &cobra.Command{
	Use:   "use",
	Short: "<role> - Assumes the specified role",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		if len(args) > 0 {
			err = use(args[0])
		} else {
			err = fmt.Errorf("usage: hologram use <role>")
		}
		if err != nil {
			log.Errorf("%s", err)
			os.Exit(1)
		}
	},
}

func use(role string) error {
	response, err := request(&protocol.AgentRequest{
		AssumeRole: &protocol.AssumeRole{
			Role: &role,
		},
	})
	if err != nil {
		return err
	}

	if response.GetFailure() != nil {
		return fmt.Errorf(response.GetFailure().GetErrorMessage())
	}

	if response.GetSuccess() != nil {
		output := fmt.Sprintf("Successfully got credentials for role '%s'", role)
		log.Info(output)
		return nil
	}

	return fmt.Errorf("unexpected response type: %v", response)
}