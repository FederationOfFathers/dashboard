//go:generate echo "---[ making sure fileb0x is installed ]"
//go:generate go get -v github.com/UnnoTed/fileb0x
//go:generate echo "---[ updating ../dashboard-ui ]"
//go:generate /bin/bash -c "cd ../dashboard-ui && git pull && cd dashboard && npm install && au build"
//go:generate echo "---[ importing ../dashboard-ui/application/ ]"
//go:generate fileb0x ./b0x.json
//go:generate echo "---[ building ]"
//go:generate go build -v
//go:generate echo "---[ done ]"
package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/FederationOfFathers/dashboard/api"
	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/events"
	"github.com/FederationOfFathers/dashboard/store"
	"github.com/FederationOfFathers/dashboard/streams"
	"github.com/FederationOfFathers/dashboard/ui"
	"github.com/apokalyptik/cfg"
	"github.com/uber-go/zap"
)

var slackAPIKey = "xox...."
var logger = zap.New(zap.NewJSONEncoder())
var devPort = 0
var noUI = false
var DB *db.DB
var mysqlURI string

func init() {
	scfg := cfg.New("cfg-slack")
	scfg.StringVar(&slackAPIKey, "apiKey", slackAPIKey, "Slack API Key (env: SLACK_APIKEY)")
	scfg.StringVar(&bot.CdnPrefix, "cdnPrefix", bot.CdnPrefix, "http url base from which to store saved uploads")
	scfg.StringVar(&bot.CdnPath, "cdnPath", bot.CdnPath, "Filesystem path to store uploads in")

	acfg := cfg.New("cfg-api")
	acfg.StringVar(&api.ListenOn, "listen", api.ListenOn, "API bind address (env: API_LISTEN)")
	acfg.StringVar(&api.AuthSecret, "secret", api.AuthSecret, "Authentication secret for use in generating login tokens")
	acfg.StringVar(&api.JWTSecret, "hmac", api.JWTSecret, "Authentication secret used for JWT tokens")
	acfg.IntVar(&devPort, "ui-dev", devPort, "proxy /application/ to localhost:devport/")

	ecfg := cfg.New("cfg-events")
	ecfg.StringVar(&events.SaveFile, "savefile", events.SaveFile, "path to the file in which events should be persisted")
	ecfg.DurationVar(&events.SaveInterval, "saveinterval", events.SaveInterval, "how often to check and see if we need to save data")

	ucfg := cfg.New("cfg-ui")
	ucfg.BoolVar(&noUI, "disable-serving", noUI, "Disable Serving of the UI")

	dcfg := cfg.New("cfg-db")
	dcfg.StringVar(&mysqlURI, "mysql", mysqlURI, "MySQL Connection URI")
	dcfg.StringVar(&store.DBPath, "path", store.DBPath, "Path to the database file")
}

func main() {
	cfg.Parse()

	store.Mind()

	DB = db.New("mysql", mysqlURI)
	streams.DB = DB
	api.DB = DB

	bot.AuthTokenGenerator = api.GenerateValidAuthTokens
	bot.LoginLink = fmt.Sprintf("http://fofgaming.com%s/", api.ListenOn)

	err := bot.SlackConnect(slackAPIKey)
	if err != nil {
		logger.Fatal("Unable to contact the slack API", zap.Error(err))
	}
	streams.Init("#-fof-streaming")
	events.Start()
	if !noUI {
		if devPort == 0 {
			api.Router.PathPrefix("/").Handler(http.FileServer(ui.HTTP))
		} else {
			rpURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d/", devPort))
			if err != nil {
				panic(err)
			}
			rp := httputil.NewSingleHostReverseProxy(rpURL)
			api.Router.PathPrefix("/").Handler(rp)
		}
	}
	api.Run()
}
