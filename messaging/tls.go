package messaging

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
)

type tlsConfig struct {
	CAFiles  []string `mapstructure:"ca_files"`
	KeyFile  string   `mapstructure:"key_file"`
	CertFile string   `mapstructure:"cert_file"`
}

// TLSConfig will load the TLS certificate
func (cfg tlsConfig) TLSConfig() (*tls.Config, error) {
	pool := x509.NewCertPool()
	for _, caFile := range cfg.CAFiles {
		caData, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, err
		}

		if !pool.AppendCertsFromPEM(caData) {
			return nil, fmt.Errorf("Failed to add CA cert at %s", caFile)
		}
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		RootCAs:      pool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return tlsConfig, nil
}
