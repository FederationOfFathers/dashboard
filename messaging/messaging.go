package messaging

import (
	"reflect"
	"time"

	"github.com/FederationOfFathers/dashboard/clients/mixer"
	"github.com/FederationOfFathers/dashboard/db"
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
	PostJoinEventMessage(e *db.Event, member string) error
	//PostMessageToChannel(channel string, message string)
}

func SendTwitchStreamMessage(sm StreamMessage) {
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
			Logger.Error("unable to send event notice", zap.Any("event", e), zap.Error(err))
		}
	}
}

func SendJoinEventMessage(e *db.Event, member *db.Member) {
	for _, msgApi := range msgApis {
		err := msgApi.PostJoinEventMessage(e, member.Name)
		if err != nil {
			Logger.Error("unable to send event join message", zap.Any("event", e), zap.String("member", member.Discord), zap.Error(err))
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
