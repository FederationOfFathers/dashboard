package api

import (
	"fmt"
	"io"
	"net/http"
)

func statsPassthrough(uri string, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	rsp, err := http.Get(fmt.Sprintf("http://127.0.0.1:8874%s", uri))
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
	Router.PathPrefix("/api/v0/xhr/stats/").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			statsPassthrough(fmt.Sprintf("%s?%s", r.URL.Path[17:], r.URL.Query().Encode()), w, r)
		},
	))
	Router.PathPrefix("/xhr/stats/").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			statsPassthrough(fmt.Sprintf("%s?%s", r.URL.Path[10:], r.URL.Query().Encode()), w, r)
		},
	))
}
