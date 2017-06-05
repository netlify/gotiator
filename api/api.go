package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/netlify/gotiator/conf"
	"github.com/netlify/gotiator/proxy"
)

type API struct {
	version string
	proxy   *proxy.Proxy
}

func (a *API) Version(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, 200, map[string]string{
		"version":     a.version,
		"name":        "Gotiator",
		"description": "Gotiator is a dead simple API gateway",
	})
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		a.Version(w, r)
	} else {
		if err := a.proxy.HandleRequest(w, r); err != nil {
			sendJSON(w, err.Code, err)
		}
	}
}

func NewAPIWithVersion(config *conf.Configuration, version string) *API {
	var apis []*proxy.ApiProxy

	for _, apiSettings := range config.APIs {
		token := os.Getenv("NETLIFY_API_" + strings.ToUpper(apiSettings.Name))
		proxy, err := proxy.NewApiProxy(apiSettings.Name, apiSettings.URL, token, apiSettings.Roles)
		if err != nil {
			logrus.WithError(err).Fatalf("Error parsing URL for %v: %v", apiSettings.Name, apiSettings.URL)
		}
		apis = append(apis, proxy)
	}

	proxy := proxy.New(config.JWT.Secret, apis)
	api := &API{version: version, proxy: proxy}

	return api
}

// ListenAndServe starts the REST API
func (a *API) ListenAndServe(hostAndPort string) error {
	return http.ListenAndServe(hostAndPort, a)
}

func sendJSON(w http.ResponseWriter, status int, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.Encode(obj)
}
