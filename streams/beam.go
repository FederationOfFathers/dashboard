package streams

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/db"
	"github.com/nlopes/slack"
	"go.uber.org/zap"
)

var bplog *zap.Logger

type mixerChannelResponse struct {
	Name          string `json:"name"`
	Token         string `json:"token"`
	ChannelOnline bool   `json:"online"`
	ChannelID     int64
	User          struct {
		AvatarUrl string `json:"avatarUrl"`
	} `json:"user"`
	Type struct {
		Name string `json:"name"`
	} `json:"type"`
}

type mixerManifestResponse struct {
	StartedAt string `json:"startedAt"`
}

type Mixer struct {
	BeamUsername string
	ChannelID    int64
	Online       bool
	Title        string
	Game         string
	StartedAt    string
	StartedTime  time.Time
	AvatarUrl    string
}

func (b *Mixer) Update() error {
	b.Online = false
	b.Game = ""
	b.StartedAt = ""
	var c = new(mixerChannelResponse)
	var cURL = fmt.Sprintf("https://mixer.com/api/v1/channels/%s", b.BeamUsername)
	bplog.Info("fetching channel", zap.String("url", cURL))
	chResponse, err := http.Get(cURL)
	if err != nil {
		return err
	}
	defer chResponse.Body.Close()
	if chResponse.StatusCode != 200 {
		return fmt.Errorf("got HTTP %d '%s' for '%s'", chResponse.StatusCode, chResponse.Status, cURL)
	}
	if err := json.NewDecoder(chResponse.Body).Decode(&c); err != nil {
		return err
	}
	if !c.ChannelOnline {
		return nil
	}
	b.Online = c.ChannelOnline
	b.ChannelID = c.ChannelID
	b.Game = c.Type.Name
	b.Title = c.Name
	b.BeamUsername = c.Token
	b.AvatarUrl = c.User.AvatarUrl
	var m = new(mixerManifestResponse)
	var mURL = fmt.Sprintf("https://mixer.com/api/v1/channels/%d/manifest.light2", b.ChannelID)
	bplog.Info("fetching manifest", zap.String("url", mURL))
	rsp, err := http.Get(mURL)
	if err != nil {
		return err
	}
	defer rsp.Body.Close()
	if rsp.StatusCode == 404 {
		// Stream went offline
		return nil
	}
	if rsp.StatusCode != 200 {
		return fmt.Errorf("got HTTP %d '%s' for '%s'", chResponse.StatusCode, chResponse.Status, mURL)
	}
	if err := json.NewDecoder(rsp.Body).Decode(&m); err != nil {
		return err
	}
	b.StartedAt = m.StartedAt
	b.StartedTime, err = time.Parse(time.RFC3339Nano, m.StartedAt)
	if err != nil {
		return err
	}
	return nil
}

func (b *Mixer) startMessage(memberID int) (string, slack.PostMessageParameters, error) {
	var messageParams = slack.NewPostMessageParameters()

	member, err := DB.MemberByID(memberID)
	if err != nil {
		return "", messageParams, err
	}

	user, err := bridge.Data.Slack.User(member.Slack)
	if err != nil {
		return "", messageParams, err
	}

	var playing = b.Game
	if playing == "" {
		playing = "something"
	}

	var chURL = fmt.Sprintf("https://mixer.com/%s", b.BeamUsername)
	messageParams.AsUser = true
	messageParams.Parse = "full"
	messageParams.LinkNames = 1
	messageParams.UnfurlMedia = true
	messageParams.UnfurlLinks = false
	messageParams.EscapeText = false
	messageParams.Attachments = append(messageParams.Attachments, slack.Attachment{
		Fallback:   fmt.Sprintf("Watch %s play %s at %s", user.Profile.RealNameNormalized, playing, chURL),
		Color:      "#1FBAED",
		AuthorIcon: "https://mixer.com/_latest/assets/favicons/favicon-16x16.png",
		AuthorName: "Mixer",
		Title:      fmt.Sprintf("%s playing %s", b.BeamUsername, b.Game),
		TitleLink:  chURL,
		ThumbURL:   b.AvatarUrl,
		Text:       b.Title,
	})
	message := fmt.Sprintf(
		"*@%s* is streaming *%s* at %s",
		user.Name,
		playing,
		chURL,
	)
	return message, messageParams, err
}

func mindMixer() {
	bplog = Logger.With(zap.String("service", "mixer"))
	bplog.Debug("begin minding")
	for _, stream := range Streams {
		if stream.Beam == "" {
			bplog.Debug("not a mixer.com stream", zap.Int("id", stream.ID), zap.Int("member_id", stream.MemberID))
			continue
		}
		bplog.Debug("minding mixer.com stream", zap.String("mixer id", stream.Beam))
		updateMixer(stream)
	}
	bplog.Debug("end minding")
}

func updateMixer(s *db.Stream) {
	mixer := &Mixer{
		BeamUsername: s.Beam,
	}
	err := mixer.Update()
	if err != nil {
		bplog.Error("Error updating mixer stream details", zap.Error(err))
		return
	}

	if !mixer.Online {
		var save bool
		if s.BeamStop < s.BeamStart {
			s.BeamStop = time.Now().Unix()
			save = true
		}
		if s.BeamStop < s.BeamStart {
			s.BeamStop = s.BeamStart + 1
			save = true
		}
		if save {
			stopError := s.Save()
			if stopError != nil {
				bplog.Error(fmt.Sprintf("Unable to save stop data: %v", stopError))
			}
		}
		return
	}

	var startedAt = mixer.StartedTime.Unix()
	if startedAt <= s.BeamStart && s.BeamGame == mixer.Game {
		// Continuation of known stream
		return
	}

	s.BeamStart = startedAt
	s.BeamGame = mixer.Game
	if s.BeamStop > s.BeamStart {
		s.BeamStop = s.BeamStart - 1
	}
	updateErr := s.Save()
	if updateErr != nil {
		bplog.Error(fmt.Sprintf("Unable to save stream data: %v", updateErr))
		return
	}

	if msg, params, err := mixer.startMessage(s.MemberID); err == nil {
		if err := bridge.PostMessage(channel, msg, params); err != nil {
			bplog.Error("error posting start message to slack", zap.String("key", s.Beam), zap.Error(err))
		}
	} else {
		bplog.Error("error building start message", zap.String("key", s.Beam), zap.Error(err), zap.Int("member_id", s.MemberID))
	}
}
