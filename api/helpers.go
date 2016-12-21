package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Error is an error with a message
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

func sendJSON(w http.ResponseWriter, status int, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.Encode(obj)
}

// UnauthorizedError is simple Error Wrapper
func UnauthorizedError(w http.ResponseWriter, message string) {
	sendJSON(w, 401, &Error{Code: 401, Message: message})
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
