package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"
)

type Proxy struct {
	jwtSecret string
	apis      []*ApiProxy
}

type ApiProxy struct {
	name    string
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

// Error is an error with a message
type Error struct {
	Message string `json:"msg"`
	Code    int    `json:"code"`
}

var bearerRegexp = regexp.MustCompile(`^(?i)Bearer (\S+$)`)

func New(secret string, apis []*ApiProxy) *Proxy {
	return &Proxy{
		jwtSecret: secret,
		apis:      apis,
	}
}

func NewApiProxy(name, urlString, token string, roles []string) (*ApiProxy, error) {
	matcher := regexp.MustCompile("^/" + name + "/?")

	target, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	director := buildDirector(target, matcher, token)
	handler := &httputil.ReverseProxy{Director: director}

	return &ApiProxy{
		name:    name,
		matcher: matcher,
		handler: handler,
		token:   token,
		roles:   roles,
	}, nil
}

func (p *Proxy) HandleRequest(w http.ResponseWriter, r *http.Request) *Error {
	for _, api := range p.apis {
		if api.matcher.MatchString(r.URL.Path) {
			if err := p.authenticateProxy(w, r, api); err != nil {
				return err
			}

			api.handler.ServeHTTP(w, r)
			return nil
		}
	}

	return &Error{"Not Found", 404}
}

func (p *Proxy) authenticateProxy(w http.ResponseWriter, r *http.Request, proxy *ApiProxy) *Error {
	if r.Method == http.MethodOptions {
		return nil
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return &Error{"This endpoint requires a Bearer token", 401}
	}

	matches := bearerRegexp.FindStringSubmatch(authHeader)
	if len(matches) != 2 {
		return &Error{"This endpoint requires a Bearer token", 401}
	}

	token, err := jwt.ParseWithClaims(matches[1], &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Header["alg"] != jwt.SigningMethodHS256.Name {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(p.jwtSecret), nil
	})
	if err != nil {
		return &Error{fmt.Sprintf("Invalid token: %v", err), 401}
	}

	claims := token.Claims.(*JWTClaims)
	if claims.StandardClaims.ExpiresAt < time.Now().Unix() {
		msg := fmt.Sprintf("Token expired at %v", time.Unix(claims.StandardClaims.ExpiresAt, 0))
		return &Error{msg, 401}
	}

	roles, ok := claims.AppMetaData["roles"]
	if ok {
		roleStrings, _ := roles.([]interface{})
		for _, data := range roleStrings {
			role, _ := data.(string)
			for _, proxyRole := range proxy.roles {
				if role == proxyRole {
					return nil
				}
			}
		}
	}

	return &Error{"Required role not found in JWT", 401}
}

func buildDirector(target *url.URL, matcher *regexp.Regexp, token string) func(req *http.Request) {
	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.Host = target.Host
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = singleJoiningSlash(target.Path, matcher.ReplaceAllString(req.URL.Path, "/"))
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		if _, ok := req.Header["User-Agent"]; !ok {
			// explicitly disable User-Agent so it's not set to default value
			req.Header.Set("User-Agent", "")
		}
		if req.Method != http.MethodOptions {
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			} else {
				req.Header.Del("Authorization")
			}
		}
		// Make sure we don't end up with double cors headers
		logrus.Infof("Proxying to: %v", req.URL)
	}
	return director
}

// From https://golang.org/src/net/http/httputil/reverseproxy.go?s=2298:2359#L72
func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}
