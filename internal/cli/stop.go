package cli

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/happyhackingspace/vulnerable-target/pkg/options"
	"github.com/happyhackingspace/vulnerable-target/pkg/provider/registry"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop vulnerable enviroment by template id and provider",
	Run: func(cmd *cobra.Command, _ []string) {
		options := options.GetOptions()
		provider := registry.GetProvider(options.ProviderName)
		if len(options.TemplateID) == 0 || len(options.ProviderName) == 0 {
			err := cmd.Help()
			if err != nil {
				log.Fatal().Msgf("%v", err)
			}
			return
		}
		if provider == nil {
			log.Fatal().Msgf("provider %s not found", options.ProviderName)
		}
		if err := provider.Stop(); err != nil {
			log.Fatal().Msgf("%v", err)
		}
		log.Info().Msgf("%s template stopped on %s", options.TemplateID, options.ProviderName)
	},
}

func init() {
	options := options.GetOptions()

	rootCmd.AddCommand(stopCmd)

	stopCmd.Flags().StringVarP(&options.ProviderName, "provider", "p", "",
		fmt.Sprintf("Specify the provider for building a vulnerable environment (%s)",
			strings.Join(slices.Collect(maps.Keys(registry.Providers)), ", ")))

	stopCmd.Flags().StringVar(&options.TemplateID, "id", "",
		"Specify a template ID for targeted vulnerable environment")
}
