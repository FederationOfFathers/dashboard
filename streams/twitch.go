package streams

import (
	"fmt"
	"time"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/db"
	twitch "github.com/knspriggs/go-twitch"
	"github.com/nlopes/slack"
	"github.com/uber-go/zap"
)

var twlog = zap.New(zap.NewJSONEncoder()).With(zap.String("module", "streams"), zap.String("service", "twitch"))
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

func (t twitchStream) startMessage(memberID int) (string, slack.PostMessageParameters, error) {
	var messageParams = slack.NewPostMessageParameters()

	member, err := DB.MemberByID(memberID)
	if err != nil {
		return "", messageParams, err
	}

	user, err := bridge.Data.Slack.User(member.Slack)
	if err != nil {
		return "", messageParams, err
	}

	var playing = t.Channel.Game
	if playing == "" {
		playing = "something"
	}

	messageParams.AsUser = true
	messageParams.Parse = "full"
	messageParams.LinkNames = 1
	messageParams.UnfurlMedia = true
	messageParams.UnfurlLinks = false
	messageParams.EscapeText = false
	messageParams.Attachments = append(messageParams.Attachments, slack.Attachment{
		Fallback:   fmt.Sprintf("Watch %s play %s at %s", user.Profile.RealNameNormalized, playing, t.Channel.URL),
		Color:      "#6441A4",
		AuthorIcon: "https://slack-imgs.com/?c=1&o1=wi16.he16.si.ip&url=https%3A%2F%2Fwww.twitch.tv%2Ffavicon.ico",
		AuthorName: "Twitch",
		Title:      fmt.Sprintf("%s playing %s", t.Channel.DisplayName, t.Channel.Game),
		TitleLink:  t.Channel.URL,
		ThumbURL:   t.Channel.Logo,
		Text:     t.Channel.Status,
	})
	message := fmt.Sprintf(
		"*@%s* is streaming *%s* at %s",
		user.Name,
		playing,
		t.Channel.URL,
	)
	return message, messageParams, err
}

func mindTwitch() {
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
	twlog.Debug("", zap.Object("stream", stream))

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
	if !postStreamMessage {
		return
	}
	if msg, params, err := stream.startMessage(s.MemberID); err == nil {
		if err := bridge.PostMessage(channel, msg, params); err != nil {
			twlog.Error("error posting start message to slack", zap.String("key", s.Twitch), zap.Error(err))
		}
	} else {
		twlog.Error("error building start message", zap.String("key", s.Twitch), zap.Error(err), zap.Int("member_id", s.MemberID))
	}
}
