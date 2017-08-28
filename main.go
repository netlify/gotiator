package main

import (
	"github.com/netlify/gotiator/cmd"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := cmd.RootCmd().Execute(); err != nil {
		logrus.WithError(err).Fatal("Failed to run root cmd")
	}
}
