package streams

import (
	"fmt"
	"time"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/store"
	"github.com/levi/twch"
	"github.com/nlopes/slack"
	"github.com/uber-go/zap"
)

var twlog = zap.New(zap.NewJSONEncoder()).With(zap.String("module", "streams"), zap.String("service", "twitch"))
var TwitchOAuthKey string
var twitchClient *twch.Client

type twitchStream struct {
	*twch.Stream
}

func (t twitchStream) startMessage(userID string) (string, slack.PostMessageParameters, error) {
	var messageParams = slack.NewPostMessageParameters()

	user, err := bridge.Data.Slack.User(userID)
	if err != nil {
		return "", messageParams, err
	}

	var playing = "something"
	if t.Channel.Game != nil {
		playing = *t.Channel.Game
	} else if t.Channel.Title != nil {
		playing = *t.Channel.Title
	}

	var preview = ""
	if t.Preview.Large != nil {
		preview = *t.Preview.Large
	} else if t.Preview.Medium != nil {
		preview = *t.Preview.Medium
	} else if t.Preview.Small != nil {
		preview = *t.Preview.Small
	}

	messageParams.AsUser = true
	messageParams.Parse = "full"
	messageParams.LinkNames = 1
	messageParams.UnfurlMedia = true
	messageParams.UnfurlLinks = true
	messageParams.EscapeText = false
	messageParams.Attachments = append(messageParams.Attachments, slack.Attachment{
		Title:     fmt.Sprintf("Watch %s play %s", user.Profile.RealNameNormalized, playing),
		TitleLink: *t.Channel.URL,
		ThumbURL:  preview,
	})
	message := fmt.Sprintf(
		"*@%s* has begun streaming *%s* at %s",
		user.Name,
		playing,
		*t.Channel.URL,
	)
	return message, messageParams, err
}

func mindTwitch() {
	twlog.Debug("begin minding")
	for key, stream := range Streams {
		if stream.Kind != "twitch" {
			continue
		}
		twlog.Debug("minding", zap.String("key", key))
		stream.update()
	}
	twlog.Debug("end minding")
}

func (s *Stream) updateTwitch() {
	now := time.Now()

	t, _, err := twitchClient.Streams.GetStream(s.ServiceID)
	if err != nil {
		twlog.Error("error fetching stream", zap.String("key", s.Key()), zap.Error(err))
		return
	}
	var stream = &twitchStream{t}

	if stream.ID == nil {
		if s.Twitch != nil {
			twlog.Debug("stopped streaming", zap.String("key", s.Key()))
			s.Twitch = nil
			s.Stop = &now
			store.DB.Streams().Put(s.Key(), s)
		}
		return
	}

	if s.Twitch == nil {
		s.Twitch = &twch.Stream{}
	}

	if s.Twitch.ID != nil {
		if *s.Twitch.ID == *stream.ID {
			twlog.Debug("still streaming", zap.String("key", s.Key()))
			return
		}
		s.Twitch = t
		then := now.Add(0 - time.Second)
		s.Stop = &then
		s.Start = &now
		twlog.Debug("stopped and then started again", zap.String("key", s.Key()))
		store.DB.Streams().Put(s.Key(), s)
		return
	}
	s.Start = &now
	s.Twitch = t
	store.DB.Streams().Put(s.Key(), s)
	twlog.Debug("started", zap.String("key", s.Key()))
	if msg, params, err := stream.startMessage(s.UserID); err == nil {
		if err := bridge.PostMessage("@demitriousk", msg, params); err != nil {
			twlog.Error("error posting start message to slack", zap.String("key", s.Key()), zap.Error(err))
		}
	} else {
		twlog.Error("error building start message", zap.String("key", s.Key()), zap.Error(err))
	}

}
