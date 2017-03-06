package main

import (
	"github.com/Sirupsen/logrus"
	"github.com/netlify/gotiator/cmd"
)

func main() {
	if err := cmd.RootCmd().Execute(); err != nil {
		logrus.WithError(err).Fatal("Failed to run root cmd")
	}
}
