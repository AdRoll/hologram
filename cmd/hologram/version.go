package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().BoolP("version", "v", false, "print the application version")
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		version, err := cmd.Flags().GetBool("version")
		if err != nil {
			return err
		}
		if version {
			fmt.Println(Version)
			os.Exit(0)
		}
		return nil
	}
}

// Legacy support
var versionCmd = &cobra.Command{
	Use:   "version",
	Hidden: true,
	Short: "print the application version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
		os.Exit(0)
	},
}
