package messaging

import (
	"reflect"
	"time"

	"github.com/FederationOfFathers/dashboard/clients/mixer"
	"github.com/FederationOfFathers/dashboard/db"
	"github.com/knspriggs/go-twitch"
	"go.uber.org/zap"
)

var msgApis []MsgAPI
var Logger *zap.Logger

type StreamMessage struct {
	Platform         string
	PlatformLogo     string
	PlatformColor    string
	PlatformColorInt int
	Username         string
	UserLogo         string
	URL              string
	Game             string
	Description      string
	Timestamp        string
}

// Add Messaging APIs that will be used to send messages
func AddMsgAPI(msgApi MsgAPI) {
	Logger.Info("Adding new message API", zap.String("type", reflect.TypeOf(msgApi).String()))
	msgApis = append(msgApis, msgApi)
}

type MsgAPI interface {
	PostStreamMessage(sm StreamMessage) error
	PostNewEventMessage(e *db.Event) error
	//PostMessageToChannel(channel string, message string)
}

func SendTwitchStreamMessage(t twitch.StreamType) {
	var playing = t.Channel.Game
	if playing == "" {
		playing = "something"
	}
	sm := StreamMessage{
		Platform:         "Twitch",
		PlatformLogo:     "https://slack-imgs.com/?c=1&o1=wi16.he16.si.ip&url=https%3A%2F%2Fwww.twitch.tv%2Ffavicon.ico",
		PlatformColor:    "#6441A4",
		PlatformColorInt: 6570404,
		Username:         t.Channel.DisplayName,
		UserLogo:         t.Channel.Logo,
		URL:              t.Channel.URL,
		Game:             playing,
		Description:      t.Channel.Status,
		Timestamp:        time.Now().Format("01/02/2006 15:04 MST"),
	}
	postStreamMessageToAllApis(sm)
}
func SendMixerStreamMessage(m mixer.Mixer) {
	sm := StreamMessage{
		Platform:         "Mixer",
		PlatformLogo:     "https://mixer.com/_latest/assets/favicons/favicon-16x16.png",
		PlatformColor:    "#1FBAED",
		PlatformColorInt: 2079469,
		Username:         m.BeamUsername,
		UserLogo:         m.AvatarUrl,
		URL:              m.GetChannelUrl(),
		Game:             m.Game,
		Description:      m.Title,
		Timestamp:        time.Now().Format("01/02/2006 15:04 MST"),
	}
	postStreamMessageToAllApis(sm)
}

func SendNewEventMessage(e *db.Event) {
	for _, msgApi := range msgApis {
		err := msgApi.PostNewEventMessage(e)
		if err != nil {
			Logger.Error("unable to send event notice", zap.Error(err), zap.Any("event", e))
		}
	}
}

func postStreamMessageToAllApis(sm StreamMessage) {
	for _, msgApi := range msgApis {
		err := msgApi.PostStreamMessage(sm)
		if err != nil {
			Logger.Error("unable to send stream update", zap.Error(err))
		}
	}
}
