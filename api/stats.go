package api

import (
	"fmt"
	"io"
	"net/http"
)

func init() {
	Router.PathPrefix("/xhr/stats/").Methods("GET").Handler(jwtHandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			rsp, err := http.Get(fmt.Sprintf("http://127.0.0.1:8874%s", r.URL.Path[10:]))
			if rsp != nil {
				defer rsp.Body.Close()
			}
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			io.Copy(w, rsp.Body)
		},
	))
}
