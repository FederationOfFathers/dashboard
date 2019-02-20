package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func init() {
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
			touchFile := ""
			if payload.Ref == "refs/heads/dev" {
				touchFile = "queue-rebuild-dev"
			} else if payload.Ref == "refs/heads/master" {
				touchFile = "queue-rebuild-prod"
			}
			fp, _ := os.OpenFile(touchFile, os.O_RDONLY|os.O_CREATE, 0666)
			if fp != nil {
				fp.Close()
			}
		},
	)
}
