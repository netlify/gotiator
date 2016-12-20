package cmd

import (
	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/netlify/speedy/conf"
	"github.com/netlify/speedy/messaging"
)

var rootCmd = &cobra.Command{
	Short: "speedy",
	Long:  "speedy",
	Run:   run,
}

func RootCmd() *cobra.Command {
	rootCmd.PersistentFlags().StringP("config", "c", "", "a config file to use")
	rootCmd.AddCommand(versionCmd)

	return rootCmd
}

func start(cmd *cobra.Command) (*conf.Config, *logrus.Entry) {
	config, err := conf.LoadConfig(cmd)
	if err != nil {
		logrus.WithError(err).Fatalf("Failed to load configuation: %v", err)
	}

	log, err := conf.ConfigureLogging(&config.LogConf)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to configure logging")
	}

	return config, log.WithField("version", Version)
}

func run(cmd *cobra.Command, _ []string) {
	config, log := start(cmd)
	messaging.Configure(config.NatsConf, log)

	// TODO
}
