package goweb

import (
	"net/http"
	"strings"
)

func HttpXXX(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func Http405(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	HttpXXX(w, http.StatusMethodNotAllowed)
}
