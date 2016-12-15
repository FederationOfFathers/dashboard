package api

import (
	"fmt"
	"io"
	"net/http"
)

func usersPassthrough(uri string, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rsp, err := http.Get(fmt.Sprintf("http://127.0.0.1:8875%s", uri))
	if rsp != nil {
		defer rsp.Body.Close()
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	io.Copy(w, rsp.Body)
}

func init() {
	Router.PathPrefix("/api/v0/xhr/users/").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			usersPassthrough(r.URL.Path[17:], w, r)
		},
	))
	Router.PathPrefix("/xhr/users/").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			usersPassthrough(r.URL.Path[10:], w, r)
		},
	))
}
