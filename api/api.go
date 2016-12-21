package api

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	"github.com/netlify/netlify-api-proxy/conf"
	"github.com/rs/cors"
)

type API struct {
	version   string
	jwtSecret string
	handler   http.Handler
	apis      []*apiProxy
}

type apiProxy struct {
	matcher *regexp.Regexp
	handler http.Handler
	token   string
	roles   []string
}

type JWTClaims struct {
	ID           string                 `json:"id"`
	Email        string                 `json:"email"`
	AppMetaData  map[string]interface{} `json:"app_metadata"`
	UserMetaData map[string]interface{} `json:"user_metadata"`
	*jwt.StandardClaims
}

var bearerRegexp = regexp.MustCompile(`^(?i)Bearer (\S+$)`)

func (a *API) Version(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, 200, map[string]string{
		"version":     a.version,
		"name":        "Netlify API Proxy",
		"description": "Netlify API Proxy is a dead simple API gateway",
	})
}

func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		a.Version(w, r)
	} else {
		for _, proxy := range a.apis {
			if proxy.matcher.MatchString(r.URL.Path) {
				if a.authenticateProxy(w, r, proxy) {
					proxy.handler.ServeHTTP(w, r)
				}
				return
			}
		}

		http.Error(w, "Not Found", 404)
	}
}

func NewAPIWithVersion(config *conf.Configuration, version string) *API {
	api := &API{version: version, jwtSecret: config.JWT.Secret}

	for _, apiSettings := range config.APIs {
		proxy := &apiProxy{}
		proxy.matcher = regexp.MustCompile("^/" + apiSettings.Name + "/?")
		proxy.token = os.Getenv("NETLIFY_API_" + strings.ToUpper(apiSettings.Name))
		proxy.roles = apiSettings.Roles

		target, err := url.Parse(apiSettings.URL)
		if err != nil {
			logrus.WithError(err).Fatalf("Error parsing URL for %v: %v", apiSettings.Name, apiSettings.URL)
		}
		targetQuery := target.RawQuery
		director := func(req *http.Request) {
			req.Host = target.Host
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = singleJoiningSlash(target.Path, proxy.matcher.ReplaceAllString(req.URL.Path, "/"))
			if targetQuery == "" || req.URL.RawQuery == "" {
				req.URL.RawQuery = targetQuery + req.URL.RawQuery
			} else {
				req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
			}
			if _, ok := req.Header["User-Agent"]; !ok {
				// explicitly disable User-Agent so it's not set to default value
				req.Header.Set("User-Agent", "")
			}
			if proxy.token != "" {
				req.Header.Set("Authorization", "Bearer "+proxy.token)
			} else {
				req.Header.Del("Authorization")
			}
			logrus.Infof("Proxying to: %v", req.URL)
		}

		proxy.handler = &httputil.ReverseProxy{Director: director}
		api.apis = append(api.apis, proxy)
	}

	corsHandler := cors.New(cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	})

	api.handler = corsHandler.Handler(api)
	return api
}

// ListenAndServe starts the REST API
func (a *API) ListenAndServe(hostAndPort string) error {
	return http.ListenAndServe(hostAndPort, a.handler)
}

func (a *API) authenticateProxy(w http.ResponseWriter, r *http.Request, proxy *apiProxy) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		UnauthorizedError(w, "This endpoint requires a Bearer token")
		return false
	}

	matches := bearerRegexp.FindStringSubmatch(authHeader)
	if len(matches) != 2 {
		UnauthorizedError(w, "This endpoint requires a Bearer token")
		return false
	}

	token, err := jwt.ParseWithClaims(matches[1], &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Header["alg"] != jwt.SigningMethodHS256.Name {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.jwtSecret), nil
	})
	if err != nil {
		UnauthorizedError(w, fmt.Sprintf("Invalid token: %v", err))
		return false
	}

	claims := token.Claims.(*JWTClaims)
	if claims.StandardClaims.ExpiresAt < time.Now().Unix() {
		msg := fmt.Sprintf("Token expired at %v", time.Unix(claims.StandardClaims.ExpiresAt, 0))
		UnauthorizedError(w, msg)
		return false
	}

	roles, ok := claims.AppMetaData["roles"]
	if ok {
		roleStrings, _ := roles.([]interface{})
		for _, data := range roleStrings {
			role, _ := data.(string)
			for _, proxyRole := range proxy.roles {
				if role == proxyRole {
					return true
				}
			}
		}
	}

	UnauthorizedError(w, "Required role not found in JWT")
	return false
}
