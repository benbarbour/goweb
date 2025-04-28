package middleware

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

var BasicAuthenticationRealm = "goweb"

type User struct {
	Name        string
	DisplayName string
	Email       string
	Password    string
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

// An AuthenticationCallback should return u if u can be authenticated.
type AuthenticationCallback func(u *User) (*User, error)

// An AuthenticationFn should return a User if one can be determined from r and is authenticated by cb.
type AuthenticationFn func(r *http.Request, cb AuthenticationCallback) (*User, error)

var basicAuthenError = errors.New("user could not be parsed from Authorization headers")

// Authenticate constructs a Middleware that uses the given functions to authenticate a request.
// If authentication fails a StatusUnauthorized is sent. If authentFn is BasicAuthen then a WWW-Authenticate
// is also sent to challenge the user to log in.
func Authenticate(authentFn AuthenticationFn, cb AuthenticationCallback, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := GetLoggerFromCtx(r.Context())

		u, err := authentFn(r, cb)
		if err != nil {
			if errors.Is(err, basicAuthenError) {
				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, BasicAuthenticationRealm))
			} else {
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				logger.Error("middleware.Authenticate error", "err", err)
				return
			}
		}
		if u == nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		logger = logger.With(slog.Group("auth",
			slog.String("user", u.Name),
		))
		r = r.WithContext(context.WithValue(r.Context(), ctxKeyLogger, logger))
		r = r.WithContext(context.WithValue(r.Context(), ctxKeyUser, u))

		next.ServeHTTP(w, r)
	})
}

// BasicAuthen is an AuthenticationFn that parses the User from the
// http.Request's BasicAuth headers. If one can't be parsed then this middleware
// responds with a WWW-Authenticate header prompting the user to authenticate. The
// "Basic realm" can be changed by modifiying the BasicAuthenticationRealm variable.
func BasicAuthen(r *http.Request, cb AuthenticationCallback) (*User, error) {
	uname, pass, ok := r.BasicAuth()
	if !ok {
		return nil, basicAuthenError
	}
	return cb(&User{Name: uname, Password: pass})
}

// TrustedHeaderAuthen is an AuthenticationMiddleware that parses the User from
// the Remote-User (required), Remote-Groups, Remote-Name, and Remote-Email headers.
func TrustedHeaderAuthen(r *http.Request, cb AuthenticationCallback) (*User, error) {
	uname := r.Header.Get("Remote-User")
	if uname == "" {
		return nil, errors.New("user could not be parsed from trusted headers")
	}

	return cb(&User{
		Name:        uname,
		Groups:      r.Header.Values("Remote-Groups"),
		DisplayName: r.Header.Get("Remote-Name"),
		Email:       r.Header.Get("Remote-Email"),
	})
}

func GetUserFromCtx(ctx context.Context) *User {
	v := ctx.Value(ctxKeyUser)
	if u, ok := v.(*User); ok {
		return u.Clone()
	}
	return nil
}
