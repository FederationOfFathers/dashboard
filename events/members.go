package events

import (
	"time"

	"github.com/nlopes/slack"
)

type EventMember struct {
	SlackID  string    `json:"slack_id"`
	Username string    `json:"username"`
	Gamertag string    `json:"gamertag"`
	Joined   time.Time `json:"joined"`
}

func NewMember(u slack.User) EventMember {
	return EventMember{
		SlackID:  u.ID,
		Username: u.Name,
		Gamertag: u.Profile.FirstName,
		Joined:   time.Now(),
	}
}
