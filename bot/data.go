package bot

import (
	"fmt"
	"sync"
	"time"

	"github.com/FederationOfFathers/dashboard/store"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

var DiscordCoreDataUpdated = sync.NewCond(new(sync.Mutex))

// ErrUsernameNotFound represents being unable to find a user by a given name (they can change)
var ErrUsernameNotFound = fmt.Errorf("Unable to find any user by that name")
var ErrChannelNotFound = fmt.Errorf("Unable to find any channel by that name")
var ErrRoleNotFound = fmt.Errorf("Unable to find any role by that name")

// ErrSlackAPIUnresponsive means that on boot up we were unable to fetch any users from the slack api
// so we assume that the api is unresponsive. It needs to be there at least when starting up to get
// initial lists of users, groups, and channels
var ErrDiscordAPIUnresponsive = fmt.Errorf("The Discord api returned no data. Error assumed")

// UpdateTimer sets how often to check for updated users, group, and channel lists in slack
var UpdateTimer = 30 * time.Minute
var UpdateRequest = make(chan struct{})

// var connection *slack.Info
var connected bool
var token string

// var Logger *zap.Logger
var StartupNotice = false

// LogLevel sets the logging verbosity for the package
var LogLevel = zap.InfoLevel

// SlackData is the structure for the state that we are geeping "up to date" during runtime. It is
// ephemeral and goes away to be repopulated on program shutdown
type DiscordData struct {
	sync.RWMutex
	Users             []*discordgo.Member
	Roles             []*discordgo.Role
	ChannelCategories []*discordgo.Channel
	Channels          []*discordgo.Channel
}

func (d *DiscordData) load() {
	store.DB.Discord().Pull("v1-data", &d) // v2-data? discord-data?
}

func (d *DiscordData) save() {
	store.DB.Discord().Put("v1-data", d)
}

// Data is the living representation of the current SlackData
var data = new(DiscordData)

func (d *DiscordData) IsUserIDAdmin(id string) (bool, error) {
	return IsUserIDAdmin(id)
}

func (d *DiscordData) Member(id string) (*discordgo.Member, error) {
	d.Lock()
	defer d.Unlock()
	for _, u := range d.Users {
		if u.User.ID == id {
			return u, nil
		}
	}
	return nil, ErrUsernameNotFound
}

func Member(id string) (*discordgo.Member, error) {
	return data.Member(id)
}
func (d *DiscordData) ChannelByID(id string) (*discordgo.Channel, error) {
	d.RLock()
	defer d.RUnlock()
	for c := range d.Channels {
		if d.Channels[c].ID == id {
			return d.Channels[c], nil
		}
	}
	return nil, ErrChannelNotFound
}

func (d *DiscordData) RoleByName(group string) (*discordgo.Role, error) {
	d.Lock()
	defer d.Unlock()
	for g := range d.Roles {
		if d.Roles[g].Name == group {
			return d.Roles[g], nil
		}
	}
	return nil, ErrRoleNotFound
}

// UserByName is a helper function to find the slack.User for a given username (@{{username}})
func (d *DiscordData) MemberByNick(username string) (*discordgo.Member, error) {
	d.Lock()
	defer d.Unlock()
	for u := range d.Users {
		if d.Users[u].Nick != "" && d.Users[u].Nick == username {
			return d.Users[u], nil
		}
		if d.Users[u].User.Username == username {
			return d.Users[u], nil
		}
	}
	return nil, ErrUsernameNotFound
}

// GetChannels returns a list of all slack.Channels
func (d *DiscordData) GetChannels() []*discordgo.Channel {
	var rval = []*discordgo.Channel{}
	d.RLock()
	for _, v := range d.Channels {
		if v.Type == discordgo.ChannelTypeGuildText {
			rval = append(rval, v)
		}
	}
	d.RUnlock()
	return rval
}

// GetUsers returns a list of all slack.User
func (d *DiscordData) GetMembers() []*discordgo.Member {
	var rval = []*discordgo.Member{}
	d.RLock()
	for _, v := range d.Users {
		rval = append(rval, v)
	}
	d.RUnlock()
	return rval
}

func GetMembers() []*discordgo.Member {
	return data.Users
}

// starts the ting
func mindLists() {
	passiveUpdate := time.Tick(UpdateTimer)
	tick := time.Tick(1 * time.Second)
	last := time.Now().Add(0 - (5 * time.Second))
	want := false
	for {
		select {
		case <-passiveUpdate:
			want = true
		case <-UpdateRequest:
			want = true
		case <-tick:
			if want {
				if time.Now().Sub(last) >= (5 * time.Second) {
					want = false
					populateLists()
				}
			}
		}
	}
}

func updateDiscordMemberList() error {
	u := []*discordgo.Member{}

	// get members
	hasMoreMembers := true
	next := "0"
	for hasMoreMembers {
		g, e := discordApi.discord.GuildMembers(discordApi.Config.GuildId, next, 1000)
		if e != nil {
			Logger.Error("Failed to fetch user list from discord", zap.Error(e))
			return e
		}
		u = append(u, g...)

		//
		if len(g) < 1000 {
			// last page has been processed
			hasMoreMembers = false
		} else {
			// we have more pages to get
			next = g[len(g)-1].User.ID
		}
	}

	data.Lock()
	data.Users = u
	data.Unlock()
	Logger.Info("Updated user list from discord", zap.Int("count", len(u)))
	return nil
}

func updateSlackRolesList() error {
	r, e := discordApi.discord.GuildRoles(discordApi.Config.GuildId)
	if e != nil {
		Logger.Error("Failed to fetch roles list from discord", zap.Error(e))
		return e
	}
	data.Lock()
	data.Roles = r
	data.Unlock()
	Logger.Info("Updated Roles list from discord", zap.Int("count", len(r)))
	return nil
}

func updateDiscordChannelsList() error {
	c, e := discordApi.discord.GuildChannels(discordApi.Config.GuildId)
	if e != nil {
		Logger.Error("Failed to fetch channel list from discord", zap.Error(e))
		return e
	}
	data.Lock()
	categories := []*discordgo.Channel{}
	chans := []*discordgo.Channel{}
	for _, channel := range c {
		if channel.Type == discordgo.ChannelTypeGuildText {
			chans = append(chans, channel)
		} else if channel.Type == discordgo.ChannelTypeGuildCategory {
			categories = append(categories, channel)
		}

	}
	data.Channels = chans
	data.ChannelCategories = categories
	data.Unlock()
	Logger.Info("Updated ChannelCategories list from slack", zap.Int("count", len(categories)))
	Logger.Info("Updated Channel list from slack", zap.Int("count", len(chans)))
	return nil
}

func populateLists() {
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		updateDiscordMemberList()
		wg.Done()
	}()
	go func() {
		updateSlackRolesList()
		wg.Done()
	}()
	go func() {
		updateDiscordChannelsList()
		wg.Done()
	}()
	wg.Wait()
	data.Lock()
	data.save()
	DiscordCoreDataUpdated.L.Lock()
	DiscordCoreDataUpdated.Broadcast()
	DiscordCoreDataUpdated.L.Unlock()
	data.Unlock()
}
