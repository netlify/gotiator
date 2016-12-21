package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/netlify/netlify-api-proxy/conf"
)

var rootCmd = &cobra.Command{
	Short: "netlify-api-proxy",
	Long:  "netlify-api-proxy",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfig(cmd, serve)
	},
}

func RootCmd() *cobra.Command {
	rootCmd.PersistentFlags().StringP("config", "c", "", "a config file to use")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(serveCmd)

	return rootCmd
}

func execWithConfig(cmd *cobra.Command, fn func(config *conf.Configuration)) {
	configFile, err := cmd.Flags().GetString("config")
	if err != nil {
		logrus.Fatalf("%+v", err)
	}

	config, err := conf.Load(configFile)
	if err != nil {
		logrus.Fatalf("Failed to load configration: %+v", err)
	}

	fn(config)
}
