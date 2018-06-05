package streams

import (
	"fmt"
	"time"

	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/messaging"
	"github.com/knspriggs/go-twitch"
	"go.uber.org/zap"
)

var twlog *zap.Logger
var TwitchOAuthKey string

var twitchClient *twitch.Session

func Twitch(clientID string) error {
	var err error
	twitchClient, err = twitch.NewSession(twitch.NewSessionInput{ClientID: clientID})
	if err != nil {
		return err
	}
	return twitchClient.CheckClientID()
}

func MustTwitch(oauth string) {
	if err := Twitch(oauth); err != nil {
		panic(err)
	}
}

type twitchStream twitch.StreamType

func mindTwitch() {
	twlog = Logger.With(zap.String("service", "twitch"))
	twlog.Debug("begin minding")
	for _, stream := range Streams {
		if stream.Twitch == "" {
			twlog.Debug("not a twitch stream", zap.Int("id", stream.ID), zap.Int("member_id", stream.MemberID))
			continue
		}
		twlog.Debug("minding twitch stream", zap.String("twithc id", stream.Twitch))
		updateTwitch(stream)
	}
	twlog.Debug("end minding")
}

func updateTwitch(s *db.Stream) {
	var client = twitchClient
	var foundStream = false

	res, err := client.GetStream(&twitch.GetStreamsInputType{Channel: s.Twitch})
	if err != nil {
		if err.Error() != "json: cannot unmarshal number into Go value of type string" {
			twlog.Error("error fetching stream", zap.String("key", s.Twitch), zap.Error(err))
		}
		return
	}

	switch len(res.Streams) {
	case 1:
		foundStream = true
	case 0:
		twlog.Debug("No active streams", zap.String("key", s.Twitch))
	default:
		twlog.Error("Too many active streams", zap.String("key", s.Twitch))
	}

	if !foundStream {
		var save bool
		if s.TwitchStreamID != "" {
			s.TwitchStreamID = ""
			save = true
		}
		if s.TwitchStop < s.TwitchStart {
			s.TwitchStop = time.Now().Unix()
			save = true
		}
		if s.TwitchStop < s.TwitchStart {
			s.TwitchStop = s.TwitchStart + 1
			save = true
		}
		if save {
			s.Save()
		}
		return
	}

	stream := twitchStream(res.Streams[0])

	if stream.ID == 0 {
		twlog.Error("Invalid stream ID", zap.String("key", s.Twitch))
		return
	}

	var isRecent bool = time.Now().Unix()-s.TwitchStart <= 1800
	streamID := fmt.Sprintf("%d", stream.ID)
	postStreamMessage := true
	if streamID == s.TwitchStreamID && s.TwitchGame == stream.Game {
		twlog.Debug("still streaming...", zap.String("key", s.Twitch))
		return
	} else if isRecent && s.TwitchGame == stream.Game {
		twlog.Debug("new ID, but still streaming...", zap.String("key", s.Twitch))
		postStreamMessage = false
	}

	s.TwitchStreamID = streamID
	s.TwitchStart = time.Now().Unix()
	if s.TwitchStop > s.TwitchStart {
		s.TwitchStop = s.TwitchStart - 1
	}
	s.TwitchGame = stream.Channel.Game
	s.Save()

	if postStreamMessage {
		messaging.SendTwitchStreamMessage(twitch.StreamType(stream))
	}
}
