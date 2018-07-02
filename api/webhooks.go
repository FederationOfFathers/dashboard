package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/FederationOfFathers/dashboard/bot"
)

func init() {
	Router.Path("/api/v0/slack/login").Methods("POST").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		user := r.Form.Get("user_id")
		if home := os.Getenv("SERVICE_DIR"); home == "" {
			bot.SendLogin(user)
		} else {
			bot.SendDevLogin(user)
		}
	})
	Router.Path("/api/v0/gh/ui-rebuild").Methods("POST").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var payload struct {
				Ref string `json:"ref"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				Logger.Warn(err.Error())
			}
			r.Body.Close()
			Logger.Warn(fmt.Sprintf("%#v", payload))
			fp, _ := os.OpenFile("queue-rebuild", os.O_RDONLY|os.O_CREATE, 0666)
			if fp != nil {
				fp.Close()
			}
		},
	)
}
