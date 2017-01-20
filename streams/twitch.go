package streams

import (
	"fmt"
	"time"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/db"
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

func Twitch(oauth string) error {
	c, err := twch.NewClient(oauth, nil)
	twitchClient = c
	return err
}

func MustTwitch(oauth string) {
	if err := Twitch(oauth); err != nil {
		panic(err)
	}
}

func (t twitchStream) startMessage(memberID int) (string, slack.PostMessageParameters, error) {
	var messageParams = slack.NewPostMessageParameters()
	var userID string

	member, err := DB.MemberByID(memberID)
	if err != nil {
		return "", messageParams, err
	}
	userID = member.Name

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
	for _, stream := range Streams {
		if stream.Twitch == "" {
			continue
		}
		twlog.Debug("minding twitch stream", zap.String("twithc id", stream.Twitch))
		updateTwitch(stream)
	}
	twlog.Debug("end minding")
}

func updateTwitch(s *db.Stream) {
	t, _, err := twitchClient.Streams.GetStream(s.Twitch)
	if err != nil {
		twlog.Error("error fetching stream", zap.String("key", s.Twitch), zap.Error(err))
		return
	}
	var stream = &twitchStream{t}
	if stream.ID == nil {
		if s.TwitchStreamID != "" {
			// Was streaming. Stopped..
			s.TwitchStreamID = ""
			s.TwitchStop = time.Now().Unix()
			s.Save()
		}
		return
	}
	streamID := fmt.Sprintf("%d", *stream.ID)
	if s.TwitchStreamID == streamID {
		// Continued streaming
		return
	}
	if s.TwitchStreamID == "" {
		twlog.Debug("started", zap.String("key", s.Twitch))
		s.TwitchStart = time.Now().Unix()
		s.TwitchStreamID = streamID
	} else {
		twlog.Debug("stopped and then started again", zap.String("key", s.Twitch))
		s.TwitchStart = time.Now().Unix()
		s.TwitchStop = s.TwitchStart - 1
		s.TwitchStreamID = streamID
	}
	s.Save()
	if msg, params, err := stream.startMessage(s.MemberID); err == nil {
		if err := bridge.PostMessage("@demitriousk", msg, params); err != nil {
			twlog.Error("error posting start message to slack", zap.String("key", s.Twitch), zap.Error(err))
		}
	} else {
		twlog.Error("error building start message", zap.String("key", s.Twitch), zap.Error(err))
	}
}
