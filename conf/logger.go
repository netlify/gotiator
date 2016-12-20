package conf

import (
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
)

type LoggingConfig struct {
	Level string `mapstructure:"log_level" json:"log_level"`
	File  string `mapstructure:"log_file" json:"log_file"`
}

// ConfigureLogging will take the logging configuration and also adds
// a few default parameters
func ConfigureLogging(config *LoggingConfig) (*logrus.Entry, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	// always use the full timestamp
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		DisableTimestamp: false,
	})

	// use a file if you want
	if config.File != "" {
		f, errOpen := os.OpenFile(config.File, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)
		if errOpen != nil {
			return nil, errOpen
		}
		logrus.SetOutput(f)
		logrus.Infof("Set output file to %s", config.File)
	}

	if config.Level != "" {
		level, err := logrus.ParseLevel(strings.ToUpper(config.Level))
		if err != nil {
			return nil, err
		}
		logrus.SetLevel(level)
		logrus.Debug("Set log level to: " + logrus.GetLevel().String())
	}

	return logrus.StandardLogger().WithField("hostname", hostname), nil
}
