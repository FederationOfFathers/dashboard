package mixer

import (
	"fmt"
	"time"
)

// since there is no go library for mixer, we should move api calls to a separate package

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

func (m Mixer) GetChannelUrl() string {
	return fmt.Sprintf("https://mixer.com/%s", m.BeamUsername)
}
