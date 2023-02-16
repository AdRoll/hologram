package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

// Version will be linked at compile time
var Version = "Unknown - Not built using standard process"

var rootCmd = &cobra.Command{
	Use:   "hologram",
	Short: "Easy, painless AWS credentials on developer laptops.",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
}

func main() {
	(*rootCmd).SetHelpCommand(&cobra.Command{Hidden: true})
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
