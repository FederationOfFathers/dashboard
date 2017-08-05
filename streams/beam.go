package streams

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/FederationOfFathers/dashboard/bridge"
	"github.com/FederationOfFathers/dashboard/db"
	"github.com/nlopes/slack"
	"github.com/uber-go/zap"
)

var bplog = zap.New(zap.NewJSONEncoder()).With(zap.String("module", "streams"), zap.String("service", "beam"))

type beamChannelResponse struct {
	Name  string `json:"name"`
	Token string `json:"token"`
	User  struct {
		AvatarUrl string `json:"avatarUrl"`
		Channel   struct {
			ID     int64 `json:"id"`
			Online bool  `json:"online"`
		} `json:"channel"`
	} `json:"user"`
	Type struct {
		Name string `json:"name"`
	} `json:"type"`
}

type beamManifestResponse struct {
	StartedAt string `json:"startedAt"`
}

type Beam struct {
	BeamUsername string
	ChannelID    int64
	Online       bool
	Title        string
	Game         string
	StartedAt    string
	StartedTime  time.Time
	AvatarUrl    string
}

func (b *Beam) Update() error {
	b.Online = false
	b.Game = ""
	b.StartedAt = ""
	var c = new(beamChannelResponse)
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
	if !c.User.Channel.Online {
		return nil
	}
	b.Online = c.User.Channel.Online
	b.ChannelID = c.User.Channel.ID
	b.Game = c.Type.Name
	b.Title = c.Name
	b.BeamUsername = c.Token
	b.AvatarUrl = c.User.AvatarUrl
	var m = new(beamManifestResponse)
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

func (b *Beam) startMessage(memberID int) (string, slack.PostMessageParameters, error) {
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

func mindBeam() {
	bplog.Debug("begin minding")
	for _, stream := range Streams {
		if stream.Beam == "" {
			bplog.Debug("not a mixer.com stream", zap.Int("id", stream.ID), zap.Int("member_id", stream.MemberID))
			continue
		}
		bplog.Debug("minding mixer.com stream", zap.String("beam id", stream.Beam))
		updateBeam(stream)
	}
	bplog.Debug("end minding")
}

func updateBeam(s *db.Stream) {
	beam := &Beam{
		BeamUsername: s.Beam,
	}
	err := beam.Update()
	if err != nil {
		bplog.Error("Error updating beam stream details", zap.Error(err))
		return
	}

	if !beam.Online {
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

	var startedAt = beam.StartedTime.Unix()
	if startedAt <= s.BeamStart && s.BeamGame == beam.Game {
		// Continuation of known stream
		return
	}

	s.BeamStart = startedAt
	s.BeamGame = beam.Game
	if s.BeamStop > s.BeamStart {
		s.BeamStop = s.BeamStart - 1
	}
	updateErr := s.Save()
	if updateErr != nil {
		bplog.Error(fmt.Sprintf("Unable to save stream data: %v", updateErr))
		return
	}

	if msg, params, err := beam.startMessage(s.MemberID); err == nil {
		if err := bridge.PostMessage(channel, msg, params); err != nil {
			bplog.Error("error posting start message to slack", zap.String("key", s.Beam), zap.Error(err))
		}
	} else {
		bplog.Error("error building start message", zap.String("key", s.Beam), zap.Error(err), zap.Int("member_id", s.MemberID))
	}
}
