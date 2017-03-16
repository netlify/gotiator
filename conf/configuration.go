package conf

import (
	"bufio"
	"os"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// Config is the main configuration for Gotiator
type Configuration struct {
	LogConf LoggingConfig `mapstructure:"log_conf"`
	JWT     struct {
		Secret string `mapstructure:"secret"`
	} `mapstructure:"jwt"`
	APIs []APISettings `mapstructure:"apis"`
	API  struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"api"`
}

/* APISettings holds the settings for the APIs to proxy
   The Name determines both the path prefix (ie. /github) and the
   environment variable for the access token (NETLIFY_API_GITHUB)
   The URL is the API URL (ie. https://api.github.com/repos/netlify/netlify-www)

	 Only requests signed with a JWT with a matching JWT secret and a claim:
	 {"app_metadata": {"roles": ["api-role"]}}

	 Will be accepted (where the api-role matches one of the roles defined in the API settings) */
type APISettings struct {
	Name  string   `mapstructure:"name"`
	URL   string   `mapstructure:"url"`
	Roles []string `mapstructure:"roles"`

	Token string `mapstructure:"-"`
}

// Load will construct the config from the file `config.json`
func Load(configFile string) (*Configuration, error) {
	viper.SetConfigType("json")

	if configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("./")               // ./config.[json | toml]
		viper.AddConfigPath("$HOME/.gotiator/") // ~/.netlify-commerce/config.[json | toml]
	}

	viper.SetEnvPrefix("gotiator")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "reading configuration from files")
	}

	config := new(Configuration)
	if err := viper.Unmarshal(config); err != nil {
		return nil, errors.Wrap(err, "unmarshaling configuration")
	}

	config, err := populateConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "populate config")
	}

	if err := configureLogging(config); err != nil {
		return nil, errors.Wrap(err, "configure logging")
	}

	return validateConfig(config)
}

func configureLogging(config *Configuration) error {
	// always use the full timestamp
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		DisableTimestamp: false,
	})

	// use a file if you want
	if config.LogConf.File != "" {
		f, errOpen := os.OpenFile(config.LogConf.File, os.O_RDWR|os.O_APPEND, 0660)
		if errOpen != nil {
			return errOpen
		}
		logrus.SetOutput(bufio.NewWriter(f))
		logrus.Infof("Set output file to %s", config.LogConf.File)
	}

	if config.LogConf.Level != "" {
		level, err := logrus.ParseLevel(strings.ToUpper(config.LogConf.Level))
		if err != nil {
			return err
		}
		logrus.SetLevel(level)
		logrus.Debug("Set log level to: " + logrus.GetLevel().String())
	}

	return nil
}

func validateConfig(config *Configuration) (*Configuration, error) {
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

	return config, nil
}
