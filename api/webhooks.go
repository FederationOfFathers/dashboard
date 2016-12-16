package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/FederationOfFathers/dashboard/bot"
)

func init() {
	Router.Path("/api/v0/gh/ui-rebuild").Methods("POST").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			var payload struct {
				Ref string `json:"ref"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				logger.Warn(err.Error())
			}
			r.Body.Close()
			logger.Warn(fmt.Sprintf("%#v", payload))
			go func() {
				if payload.Ref == "refs/heads/dev" {
					bot.SendMessage("#-fof-dashboard", "rebuilding dev ui")
					now := time.Now()
					_, err := exec.Command(
						fmt.Sprintf(
							"%s/services/dashboard-dev/rebuild-ui",
							os.Getenv("HOME"),
						),
					).CombinedOutput()
					if err != nil {
						bot.SendMessage(
							"#-fof-dashboard",
							fmt.Sprintf("error rebuilding dev ui: ```%s```", err.Error()),
						)
					} else {
						bot.SendMessage(
							"#-fof-dashboard",
							fmt.Sprintf("dev ui rebuilt in %s", time.Now().Sub(now).String()),
						)
						time.Sleep(1)
					}
					err = exec.Command("/usr/bin/sv", "restart", fmt.Sprintf("%s/services/dashboard-dev/", os.Getenv("HOME"))).Run()
					if err != nil {
						logger.Warn(err.Error())
					}
				} else if payload.Ref == "refs/heads/master" {
					bot.SendMessage("#-fof-dashboard", "rebuilding production ui")
					now := time.Now()
					_, err := exec.Command(
						fmt.Sprintf(
							"%s/services/dashboard/update",
							os.Getenv("HOME"),
						),
					).CombinedOutput()
					if err != nil {
						bot.SendMessage(
							"#-fof-dashboard",
							fmt.Sprintf("error rebuilding production ui: ```%s```", err.Error()),
						)
					} else {
						bot.SendMessage(
							"#-fof-dashboard",
							fmt.Sprintf("production ui rebuilt in %s", time.Now().Sub(now).String()),
						)
					}
					err = exec.Command("/usr/bin/sv", "restart", fmt.Sprintf("%s/services/dashboard/", os.Getenv("HOME"))).Run()
					if err != nil {
						logger.Warn(err.Error())
					}
				}
			}()
		},
	)
}
