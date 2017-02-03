//go:generate fileb0x ./b0x.json
package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/FederationOfFathers/dashboard/api"
	"github.com/FederationOfFathers/dashboard/bot"
	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/events"
	"github.com/FederationOfFathers/dashboard/store"
	"github.com/FederationOfFathers/dashboard/streams"
	"github.com/FederationOfFathers/dashboard/ui"
	"github.com/apokalyptik/cfg"
	"github.com/uber-go/zap"
)

var twitchClientID = ""
var slackAPIKey = "xox...."
var slackMessagingKey = ""
var logger = zap.New(zap.NewJSONEncoder())
var devPort = 0
var noUI = false
var DB *db.DB
var mysqlURI string
var streamChannel = "-fof-dashboard"
var mindStreams bool

func init() {
	scfg := cfg.New("cfg-slack")
	scfg.StringVar(&slackAPIKey, "apiKey", slackAPIKey, "Slack API Key (env: SLACK_APIKEY)")
	scfg.StringVar(&slackMessagingKey, "messagingKey", slackMessagingKey, "Slack Messaging API Key (env: SLACK_MESSAGINGAPIKEY)")
	scfg.StringVar(&bot.CdnPrefix, "cdnPrefix", bot.CdnPrefix, "http url base from which to store saved uploads")
	scfg.StringVar(&bot.CdnPath, "cdnPath", bot.CdnPath, "Filesystem path to store uploads in")
	scfg.BoolVar(&bot.StartupNotice, "startupNotice", bot.StartupNotice, "send a start-up notice to slack")
	scfg.StringVar(&streamChannel, "streamChannel", streamChannel, "where to send streaming notices")
	scfg.BoolVar(&mindStreams, "mindStreams", mindStreams, "should we mind streaming?")

	acfg := cfg.New("cfg-api")
	acfg.StringVar(&api.ListenOn, "listen", api.ListenOn, "API bind address (env: API_LISTEN)")
	acfg.StringVar(&api.AuthSecret, "secret", api.AuthSecret, "Authentication secret for use in generating login tokens")
	acfg.StringVar(&api.JWTSecret, "hmac", api.JWTSecret, "Authentication secret used for JWT tokens")
	acfg.IntVar(&devPort, "ui-dev", devPort, "proxy /application/ to localhost:devport/")

	ecfg := cfg.New("cfg-events")
	ecfg.StringVar(&events.SaveFile, "savefile", events.SaveFile, "path to the file in which events should be persisted")
	ecfg.DurationVar(&events.SaveInterval, "saveinterval", events.SaveInterval, "how often to check and see if we need to save data")
	ecfg.StringVar(&events.OldEventLinkHMAC, "hmackey", events.OldEventLinkHMAC, "hmac key for generating team tool login links")

	ucfg := cfg.New("cfg-ui")
	ucfg.BoolVar(&noUI, "disable-serving", noUI, "Disable Serving of the UI")

	dcfg := cfg.New("cfg-db")
	dcfg.StringVar(&mysqlURI, "mysql", mysqlURI, "MySQL Connection URI")
	dcfg.StringVar(&store.DBPath, "path", store.DBPath, "Path to the database file")

	tcfg := cfg.New("cfg-twitch")
	tcfg.StringVar(&twitchClientID, "clientID", "", "Twitch OAuth key")
}

func main() {
	cfg.Parse()

	store.Mind()

	DB = db.New("mysql", mysqlURI)
	streams.DB = DB
	api.DB = DB

	bot.AuthTokenGenerator = api.GenerateValidAuthTokens
	if home := os.Getenv("SERVICE_DIR"); home == "" {
		bot.LoginLink = fmt.Sprintf("http://dashboard.fofgaming.com/")
	} else {
		bot.LoginLink = fmt.Sprintf("http://fofgaming.com%s/", api.ListenOn)
	}

	if slackMessagingKey != "" {
		bot.MessagingKey = slackMessagingKey
	} else {
		bot.MessagingKey = slackAPIKey
	}
	err := bot.SlackConnect(slackAPIKey)
	if err != nil {
		logger.Fatal("Unable to contact the slack API", zap.Error(err))
	}

	bridge.SlackCoreDataUpdated = bot.SlackCoreDataUpdated
	bridge.OldEventToolLink = events.OldEventToolLink

	if mindStreams {
		logger.Info("Minding streams", zap.String("channel", streamChannel), zap.String("twitch_client_id", twitchClientID))
		streams.Init(streamChannel)
		streams.MustTwitch(twitchClientID)
		streams.Mind()
	} else {
		streams.MindList()
		logger.Info("Not minding streams")
	}

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
