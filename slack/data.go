package bot

import (
	"fmt"
	"sync"
	"time"

	"github.com/FederationOfFathers/dashboard/store"
	"github.com/nlopes/slack"
	"github.com/uber-go/zap"
)

// ErrUsernameNotFound represents being unable to find a user by a given name (they can change)
var ErrUsernameNotFound = fmt.Errorf("Unable to find any user by that name")
var ErrChannelNotFound = fmt.Errorf("Unable to find any channel by that name")
var ErrGroupNotFound = fmt.Errorf("Unable to find any group by that name")

// ErrSlackAPIUnresponsive means that on boot up we were unable to fetch any users from the slack api
// so we assume that the api is unresponsive. It needs to be there at least when starting up to get
// initial lists of users, groups, and channels
var ErrSlackAPIUnresponsive = fmt.Errorf("The slack api returned no data. Error assumed")

// UpdateTimer sets how often to check for updated users, group, and channel lists in slack
var UpdateTimer = time.Minute

// SlackData is the structure for the state that we are geeping "up to date" during runtime. It is
// ephemeral and goes away to be repopulated on program shutdown
type SlackData struct {
	sync.RWMutex
	Users    []slack.User
	Groups   []slack.Group
	Channels []slack.Channel
}

func (s *SlackData) load() {
	store.DB.Slack().Pull("v1-data", &s)
}

func (s *SlackData) save() {
	store.DB.Slack().Put("v1-data", s)
}

// Data is the living representation of the current SlackData
var data = new(SlackData)

func (s *SlackData) IsUserIDAdmin(id string) (bool, error) {
	return IsUserIDAdmin(id)
}

func (s *SlackData) IsUsernameAdmin(name string) (bool, error) {
	return IsUsernameAdmin(name)
}

func (s *SlackData) User(id string) (*slack.User, error) {
	s.Lock()
	defer s.Unlock()
	for _, u := range s.Users {
		if u.ID == id {
			return &u, nil
		}
	}
	return nil, ErrUsernameNotFound
}

func (s *SlackData) ChannelByName(channel string) (*slack.Channel, error) {
	s.Lock()
	defer s.Unlock()
	for c := range s.Channels {
		if s.Channels[c].Name == channel {
			return &s.Channels[c], nil
		}
	}
	return nil, ErrChannelNotFound
}

func (s *SlackData) GroupByName(group string) (*slack.Group, error) {
	s.Lock()
	defer s.Unlock()
	for g := range s.Groups {
		if s.Groups[g].Name == group {
			return &s.Groups[g], nil
		}
	}
	return nil, ErrGroupNotFound
}

// UserByName is a helper function to find the slack.User for a given username (@{{username}})
func (s *SlackData) UserByName(username string) (*slack.User, error) {
	s.Lock()
	defer s.Unlock()
	for u := range s.Users {
		if s.Users[u].Name == username {
			return &s.Users[u], nil
		}
	}
	return nil, ErrUsernameNotFound
}

// UserGroups for a given userID get a list of slack.Groups that they are a member of. The bot
// (or user) who we are needs to be in the groups. So it's a cross section of the groups that
// we AND the user are both in.
func (s *SlackData) UserGroups(userID string) []slack.Group {
	var rval = []slack.Group{}
	s.RLock()
	for _, group := range s.Groups {
		for _, member := range group.Members {
			if member == userID {
				rval = append(rval, group)
			}
		}
	}
	s.RUnlock()
	return rval
}

// UserChannels returns a list of all slack.Channel that the user is a member of.
// unlike Groups.. we do not need to be a member of the channel to see this info.
func (s *SlackData) UserChannels(userID string) []slack.Channel {
	var rval = []slack.Channel{}
	s.RLock()
	for _, channel := range s.Channels {
		for _, member := range channel.Members {
			if member == userID {
				rval = append(rval, channel)
			}
		}
	}
	s.RUnlock()
	return rval
}

// GetGroups return a list of all slack.Group that we are in
func (s *SlackData) GetGroups() []slack.Group {
	var rval = []slack.Group{}
	s.RLock()
	for _, v := range s.Groups {
		rval = append(rval, v)
	}
	s.RUnlock()
	return rval
}

// GetChannels returns a list of all slack.Channels
func (s *SlackData) GetChannels() []slack.Channel {
	var rval = []slack.Channel{}
	s.RLock()
	for _, v := range s.Channels {
		rval = append(rval, v)
	}
	s.RUnlock()
	return rval
}

// GetUsers returns a list of all slack.User
func (s *SlackData) GetUsers() []slack.User {
	var rval = []slack.User{}
	s.RLock()
	for _, v := range s.Users {
		rval = append(rval, v)
	}
	s.RUnlock()
	return rval
}

func mindLists() {
	t := time.Tick(UpdateTimer)
	for {
		select {
		case <-t:
			populateLists()
		}
	}
}

func populateLists() {
	if u, e := api.GetUsers(); e == nil {
		data.Lock()
		data.Users = u
		data.Unlock()
		logger.Info("Updated user list from slack", zap.Int("count", len(u)))
	} else {
		logger.Error("Failed to fetch user list from slack", zap.Error(e))
	}

	if g, e := api.GetGroups(false); e == nil {
		data.Lock()
		groups := []slack.Group{}
		data.Unlock()
		for _, gr := range g {
			if gr.IsArchived {
				// Archived groups aren't real groups
				api.LeaveGroup(gr.ID)
				continue
			}
			if len(gr.Members) < 2 {
				// If I am the only member of a group then leave it
				api.LeaveGroup(gr.ID)
				continue
			}
			if len(gr.Name) > 5 && gr.Name[:5] == "mpdm-" {
				// Multi Party Direct MEssages don't count
				continue
			}
			logger.Debug("am in group", zap.String("group_id", gr.ID), zap.String("group_name", gr.Name))
			groups = append(groups, gr)
		}
		data.Groups = groups
		logger.Info("Updated Group list from slack", zap.Int("count", len(g)))
	} else {
		logger.Error("Failed to fetch group list from slack", zap.Error(e))
	}

	if c, e := api.GetChannels(false); e == nil {
		data.Lock()
		chans := []slack.Channel{}
		for _, channel := range c {
			if channel.IsArchived == true {
				logger.Debug(
					"Filtering channel from channel list",
					zap.String("channel_id", channel.ID),
					zap.String("channel_name", channel.Name),
					zap.String("reason", "archived"))
				continue
			}
			chans = append(chans, channel)
		}
		data.Channels = chans
		data.Unlock()
		logger.Info("Updated Channel list from slack", zap.Int("count", len(c)))
		if connected {
			for _, channel := range chans {
				if channel.IsMember {
					logger.Debug(
						"already in channel",
						zap.String("channel_id", channel.ID),
						zap.String("channel_name", channel.Name))
					continue
				}
				if _, err := rtm.JoinChannel(channel.ID); err != nil {
					logger.Error("failed to join channel",
						zap.String("channel_id", channel.ID),
						zap.String("channel_name", channel.Name),
						zap.Error(err))
				} else {
					logger.Info(
						"joined channel",
						zap.String("channel_id", channel.ID),
						zap.String("channel_name", channel.Name))
				}
			}
		}
	} else {
		logger.Error("Failed to fetch channel list from slack", zap.Error(e))
	}
	data.Lock()
	data.save()
	data.Unlock()
}
