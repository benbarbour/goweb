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

type BasicAuthorizationFunc func(user, password string) bool

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

		if !ok || !fn(uname, pass) {
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

type TrustedHeaderAuthorizationFn func(user string, groups []string) bool

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

		if !fn(uname, groups) {
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
