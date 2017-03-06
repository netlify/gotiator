package cmd

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/netlify/gotiator/api"
	"github.com/netlify/gotiator/conf"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:  "serve",
	Long: "Start API server",
	Run: func(cmd *cobra.Command, args []string) {
		execWithConfig(cmd, serve)
	},
}

func serve(config *conf.Configuration) {
	api := api.NewAPIWithVersion(config, Version)

	l := fmt.Sprintf("%v:%v", config.API.Host, config.API.Port)
	logrus.Infof("Netlify Auth API started on: %s", l)
	api.ListenAndServe(l)
}
