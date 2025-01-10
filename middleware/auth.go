package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

type User struct {
	Name        string
	DisplayName string
	Email       string
	Groups      []string
}

func (u *User) Clone() *User {
	u2 := &User{
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Email:       u.Email,
		Groups:      make([]string, len(u.Groups)),
	}
	copy(u2.Groups, u.Groups)
	return u2
}

type BasicAuthorizationFunc func(r *http.Request, user, password string) bool

// BasicAuthentication saves a User, parsed from BasicAuth headers or URL, to the http.Request's context.
// If no BasicAuth information is present then it sends a WWW-Authenticate header to prompt the browser
// to ask for credentials.
// The User can be retrieved with GetUserFromCtx() in later middleware.
func BasicAuthentication(fn BasicAuthorizationFunc, realm string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uname, pass, ok := r.BasicAuth()

		logger := GetLoggerFromCtx(r.Context())
		if logger != nil {
			logger = logger.With(slog.Group("auth",
				slog.String("user", uname),
				slog.String("type", "basic"),
			))
			r = r.WithContext(context.WithValue(r.Context(), ctxKeyLogger, logger))
		}

		if !ok || !fn(r, uname, pass) {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		logger.Info("login")

		r = r.WithContext(context.WithValue(r.Context(), ctxKeyUser, &User{
			Name: uname,
		}))

		next.ServeHTTP(w, r)
	})
}

type TrustedHeaderAuthorizationFn func(r *http.Request, user string, groups []string) bool

// TrustedHeaderAuthentication parses a the Remote-User, Remote-Groups, Remote-Name, and Remote-Email
// headers into a User and save it in the http.Request's context.
// The User can be retrieved with GetUserFromCtx() in later middleware.
func TrustedHeaderAuthentication(fn TrustedHeaderAuthorizationFn, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uname := r.Header.Get("Remote-User")
		groups := r.Header.Values("Remote-Groups")

		logger := GetLoggerFromCtx(r.Context())
		if logger != nil {
			logger = logger.With(slog.Group("auth",
				slog.String("user", uname),
				slog.String("type", "trustedHeader"),
			))
			r = r.WithContext(context.WithValue(r.Context(), ctxKeyLogger, logger))
		}

		if !fn(r, uname, groups) {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		logger.Info("login")

		r = r.WithContext(context.WithValue(r.Context(), ctxKeyUser, &User{
			Name:        uname,
			Groups:      groups,
			DisplayName: r.Header.Get("Remote-Name"),
			Email:       r.Header.Get("Remote-Email"),
		}))

		next.ServeHTTP(w, r)
	})
}

func GetUserFromCtx(ctx context.Context) *User {
	v := ctx.Value(ctxKeyUser)
	if u, ok := v.(*User); ok {
		return u.Clone()
	}
	return nil
}
