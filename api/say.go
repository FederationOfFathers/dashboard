package api

import (
	"net/http"

	"github.com/FederationOfFathers/dashboard/slack"
)

func init() {
	Router.Path("/api/v0/say").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		to := r.FormValue("to")
		if to == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		msg := r.FormValue("message")
		if msg == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		bot.SendMessage(to, msg)
	})
}
