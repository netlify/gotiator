package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/netlify/netlify-api-proxy/api"
	"github.com/netlify/netlify-api-proxy/conf"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:  "serve",
	Long: "Start API server",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfig(cmd, serve)
	},
}

func execWithConfig(cmd *cobra.Command, fn func(config *conf.Config)) {
	config, err := conf.LoadConfig(cmd)
	if err != nil {
		logrus.Fatalf("Failed to load configration: %+v", err)
	}

	fn(config)
}

func serve(config *conf.Config) {
	api := api.NewAPIWithVersion(config, Version)

	if config.API.Port == 0 && os.Getenv("PORT") != "" {
		port, err := strconv.Atoi(os.Getenv("PORT"))
		if err != nil {
			return nil, errors.Wrap(err, "formatting PORT into int")
		}

		config.API.Port = port
	}

	if config.API.Port == 0 && config.API.Host == "" {
		config.API.Port = 8080
	}

	l := fmt.Sprintf("%v:%v", config.API.Host, config.API.Port)
	logrus.Infof("Netlify Auth API started on: %s", l)
	api.ListenAndServe(l)
}
