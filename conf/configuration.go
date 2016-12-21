package conf

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config is the main configuration for Netlify API Proxy
type Config struct {
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

// LoadConfig loads the config from a file if specified, otherwise from the environment
func LoadConfig(cmd *cobra.Command) (*Config, error) {
	viper.SetConfigType("json")

	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		return nil, err
	}

	viper.SetEnvPrefix("netlify")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if configFile, _ := cmd.Flags().GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("./")
	}

	if err := viper.ReadInConfig(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	config := new(Config)

	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	log.Printf("Config: %v", config)

	return config, nil
}
