package cli

import (
	"github.com/happyhackingspace/vulnerable-target/pkg/templates"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available templates with descriptions",
	Run: func(cmd *cobra.Command, args []string) {
		templates.List()
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
