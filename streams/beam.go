package streams

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/FederationOfFathers/dashboard/clients/mixer"
	"github.com/FederationOfFathers/dashboard/db"
	"github.com/FederationOfFathers/dashboard/messaging"
	"go.uber.org/zap"
)

var bplog *zap.Logger

type Mixer mixer.Mixer

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

func (b *Mixer) Update() error {
	b.Online = false
	b.Game = ""
	b.StartedAt = ""
	var c = mixerChannelResponse{}
	var cURL = fmt.Sprintf("https://mixer.com/api/v1/channels/%s", b.BeamUsername)
	bplog.Debug("fetching channel", zap.String("url", cURL))
	chResponse, err := http.Get(cURL)
	if err != nil {
		return err
	}
	defer chResponse.Body.Close()
	if chResponse.StatusCode == 404 {
		bplog.Info(fmt.Sprintf("channel %s is not valid (404)", cURL))
		return nil
	} else if chResponse.StatusCode != 200 {
		return fmt.Errorf("got HTTP %d '%s' for '%s'", chResponse.StatusCode, chResponse.Status, cURL)
	}

	bodyContent, err := ioutil.ReadAll(chResponse.Body)
	if err != nil {
		bplog.Error("Unable to read body bytes", zap.Error(err))
	}

	if err := json.Unmarshal(bodyContent, &c); err != nil {
		return fmt.Errorf("Unable to decode JSON - %s", err.Error())
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
	var m = mixerManifestResponse{}
	var mURL = fmt.Sprintf("https://mixer.com/api/v1/channels/%d/manifest.light2", b.ChannelID)
	bplog.Debug("fetching manifest", zap.String("url", mURL))
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
	m := Mixer{
		BeamUsername: s.Beam,
	}
	err := m.Update()
	if err != nil {
		bplog.Error("Error updating mixer stream details", zap.Error(err))
		return
	}

	if !m.Online {
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

	var startedAt = m.StartedTime.Unix()
	if startedAt <= s.BeamStart && s.BeamGame == m.Game {
		// Continuation of known stream
		return
	}

	s.BeamStart = startedAt
	s.BeamGame = m.Game
	if s.BeamStop > s.BeamStart {
		s.BeamStop = s.BeamStart - 1
	}
	updateErr := s.Save()
	if updateErr != nil {
		bplog.Error(fmt.Sprintf("Unable to save stream data: %v", updateErr))
		return
	}

	messaging.SendMixerStreamMessage(mixer.Mixer(m))
}
