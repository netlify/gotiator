package messaging

import (
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/nats-io/nats"
	"github.com/rybit/nats_logrus_hook"
	"github.com/rybit/nats_metrics"
)

type NatsConfig struct {
	tlsConfig `mapstructure:",squash"`

	Servers        []string `mapstructure:"servers"`
	LogSubject     string   `mapstructure:"log_subject"`
	MetricsSubject string   `mapstructure:"metrics_subject"`
}

// ServerString will build the proper string for nats connect
func (config *NatsConfig) ServerString() string {
	return strings.Join(config.Servers, ",")
}

func Configure(config *NatsConfig, log *logrus.Entry) *nats.Conn {
	var nc *nats.Conn
	var err error
	if config != nil {
		nc, err = ConnectToNats(config, ErrorHandler(log))
		if err == nil {
			if config.LogSubject != "" {
				hook := nhook.NewNatsHook(nc, config.LogSubject)
				logrus.AddHook(hook)
				log.Info("Added nats hook")
			} else {
				log.Info("Skipping nats log hook because it doesn't exist")
			}

			if config.MetricsSubject != "" {
				if err := metrics.Init(nc, config.MetricsSubject); err != nil {
					log.WithError(err).Warn("Failed to configure metrics library")
				}
			} else {
				log.Info("Skipping nats metrics lib because it doesn't exist")
				metrics.Init(nil, "")
			}

		} else {
			log.WithFields(logrus.Fields{
				"cert_file": config.CertFile,
				"key_file":  config.KeyFile,
				"ca_files":  strings.Join(config.CAFiles, ","),
			}).WithError(err).Errorf("Failed to connect to nats, metrics and logs won't be sent")
		}
	} else {
		log.Info("Skipping connecting to nats because there is no nats config")
	}

	return nc
}

// ConnectToNats will do a TLS connection to the nats servers specified
func ConnectToNats(config *NatsConfig, errHandler nats.ErrHandler) (*nats.Conn, error) {
	tlsConfig, err := config.TLSConfig()
	if err != nil {
		return nil, err
	}

	if errHandler != nil {
		return nats.Connect(config.ServerString(), nats.Secure(tlsConfig), nats.ErrorHandler(errHandler))
	}

	return nats.Connect(config.ServerString(), nats.Secure(tlsConfig))
}

func ErrorHandler(log *logrus.Entry) nats.ErrHandler {
	errLogger := log.WithField("component", "error-logger")
	return func(conn *nats.Conn, sub *nats.Subscription, err error) {
		errLogger.WithError(err).WithFields(logrus.Fields{
			"subject":     sub.Subject,
			"group":       sub.Queue,
			"conn_status": conn.Status(),
		}).Error("Error while consuming from " + sub.Subject)
	}
}
