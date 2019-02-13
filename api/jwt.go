package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/securecookie"
)

// JWTSecret is the Secret used when signing JTW tokens
var JWTSecret string
var jwtSecretBytes []byte

var cookie *securecookie.SecureCookie
var cookieName = "secure-cookie"

type contextKey int

const (
	authContext contextKey = iota
)

var errUnauthenticated = fmt.Errorf("Unauthenticated Request")

func handlerFunc(fn func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(fn)
}

func requestAuth(r *http.Request) map[string]string {
	if a := r.Context().Value(authContext); a != nil {
		return a.(map[string]string)
	}
	return map[string]string{}
}

func authorize(userID string, memberID int, w http.ResponseWriter, r *http.Request) {
	var auth = requestAuth(r)
	auth["userid"] = userID                   //slack userid
	auth["memberid"] = strconv.Itoa(memberID) //member id
	if encoded, err := cookie.Encode(cookieName, auth); err == nil {
		http.SetCookie(
			w,
			&http.Cookie{
				Name:     cookieName,
				Value:    encoded,
				Path:     "/",
				Domain:   "fofgaming.com",
				Expires:  time.Now().Add(365 * 24 * time.Hour),
				HttpOnly: false,
			},
		)
	}
}

// TODO if using old userid, replace with memberid based token
func authorized(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	if a := r.Context().Value(authContext); a != nil {
		auth := a.(map[string]string)
		if memberid, memberOk := auth["memberid"]; memberOk && memberid != "" {
			return r, nil
		}
		if _, ok := auth["userid"]; ok {
			return r, nil
		}
		return r, errUnauthenticated
	}
	if c, err := r.Cookie(cookieName); err == nil {
		auth := make(map[string]string)
		if err = cookie.Decode(cookieName, c.Value, &auth); err == nil {
			return r.WithContext(context.WithValue(r.Context(), authContext, auth)), nil
		}
	}
	return r, errUnauthenticated
}

func authenticated(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		r, err = authorized(w, r)
		if err != nil {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("forbidden"))
			return
		}
		next(w, r)
	})
}
