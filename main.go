//go:generate echo "---[ making sure fileb0x is installed ]"
//go:generate go get -v github.com/UnnoTed/fileb0x
//go:generate echo "---[ updating ../dashboard-ui ]"
//go:generate /bin/bash -c "cd ../dashboard-ui && git pull"
//go:generate echo "---[ importing ../dashboard-ui/application/ ]"
//go:generate fileb0x ./b0x.json
//go:generate echo "---[ building ]"
//go:generate go build -v
//go:generate echo "---[ done ]"
package main

import (
	"fmt"
	"net/http"

	"github.com/FederationOfFathers/dashboard/api"
	"github.com/FederationOfFathers/dashboard/events"
	"github.com/FederationOfFathers/dashboard/slack"
	"github.com/FederationOfFathers/dashboard/ui"
	"github.com/apokalyptik/cfg"
	"github.com/uber-go/zap"
)

var slackAPIKey = "xox...."
var logger = zap.NewJSON()

func init() {
	scfg := cfg.New("cfg-slack")
	scfg.StringVar(&slackAPIKey, "apiKey", slackAPIKey, "Slack API Key (env: SLACK_APIKEY)")
	scfg.StringVar(&bot.CdnPrefix, "cdnPrefix", bot.CdnPrefix, "http url base from which to store saved uploads")
	scfg.StringVar(&bot.CdnPath, "cdnPath", bot.CdnPath, "Filesystem path to store uploads in")

	acfg := cfg.New("cfg-api")
	acfg.StringVar(&api.ListenOn, "listen", api.ListenOn, "API bind address (env: API_LISTEN)")
	acfg.StringVar(&api.AuthSecret, "secret", api.AuthSecret, "Authentication secret for use in generating login tokens")
	acfg.StringVar(&api.JWTSecret, "hmac", api.JWTSecret, "Authentication secret used for JWT tokens")

	ecfg := cfg.New("cfg-events")
	ecfg.StringVar(&events.SaveFile, "savefile", events.SaveFile, "path to the file in which events should be persisted")
	ecfg.DurationVar(&events.SaveInterval, "saveinterval", events.SaveInterval, "how often to check and see if we need to save data")
}

func main() {
	cfg.Parse()
	bot.AuthTokenGenerator = api.GenerateValidAuthTokens

	bot.LoginLink = fmt.Sprintf("http://fofgaming.com%s/", api.ListenOn)

	data, err := bot.SlackConnect(slackAPIKey)
	events.Start(data)
	if err != nil {
		logger.Fatal("Unable to contact the slack API", zap.Error(err))
	}
	api.Router.PathPrefix("/application/").Handler(http.FileServer(ui.HTTP))
	api.Run(data, events.Data)
}
