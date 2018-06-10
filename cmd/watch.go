package cmd

import (
	"github.com/spf13/cobra"
	"fmt"
)


var watchCmd = &cobra.Command{
	Use:     "watch",
	Short:   "Watch master for changes and sync to slave. NOT IMPLEMENTED YET!",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Watch")
	},
}

