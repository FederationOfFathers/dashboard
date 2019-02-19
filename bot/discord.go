package bot

import (
	"fmt"
	"strings"

	"github.com/FederationOfFathers/dashboard/messaging"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type DiscordAPI struct {
	Config         DiscordCfg
	discord        *discordgo.Session
	assignmentMsgs map[string]map[string]string
}

type DiscordCfg struct {
	ClientId        string         `yaml:"appClientId"`
	Token           string         `yaml:"botToken"`
	StreamChannelId string         `yaml:"streamChannelId"`
	GuildId         string         `yaml:"guildId"`
	RoleCfg         DiscordRoleCfg `yaml:"roleConfig"`
}

var discordApi *DiscordAPI

func NewDiscordAPI(cfg DiscordCfg) *DiscordAPI {
	return &DiscordAPI{
		Config: cfg,
	}
}

// StartDiscord starts Discord API bot
func StartDiscord(cfg DiscordCfg) *DiscordAPI {
	discordApi = NewDiscordAPI(cfg)
	discordApi.Connect()
	if cfg.RoleCfg.ChannelId != "" {
		discordApi.StartRoleHandlers()
	}

	discordApi.discord.UpdateStatus(0, "Delete All Things Slack")

	// data cache
	data.load()
	populateLists()
	go mindLists()

	return discordApi
}

func (d *DiscordAPI) teamCommandHandler(s *discordgo.Session, event *discordgo.MessageCreate) {
	Logger.Debug("handle", zap.Any("event", event))
	switch event.Content {
	case "!team":
		d.sendTeamToolLink(event)
	}
}

func (d DiscordAPI) sendTeamToolLink(m *discordgo.MessageCreate) {
	d.discord.ChannelMessageSend(m.ChannelID, "https://ui.fofgaming.com")
}

// FindIDByUsername searches the server for a user with the specified username. Returns the ID and username
func (d *DiscordAPI) FindIDByUsername(username string) (string, string) {
	return d.FindIDByUsernameStartingAt(username, "0")
}

// FindGuildRole searches the configured guild roles to find the one that matches the given roleID
func (d *DiscordAPI) FindGuildRole(roleID string) (*discordgo.Role, error) {
	roles, err := d.discord.GuildRoles(d.Config.GuildId)
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if role.ID == roleID {
			return role, nil
		}
	}

	return nil, fmt.Errorf("No matching role found")
}

// FindIDByUsernameStartingAt searches the server for a user with the specified username starting at the id/snowflake. Returns the ID and username
func (d *DiscordAPI) FindIDByUsernameStartingAt(username string, snowflake string) (string, string) {
	members, err := d.discord.GuildMembers(d.Config.GuildId, snowflake, 1000)
	if err != nil {
		Logger.Error("unable to get guild members list", zap.Error(err))
	}

	// if no members, we've iterated through all members
	if len(members) <= 0 {
		return "", ""
	}

	// search for the member in the current list
	usernameParts := strings.SplitN(username, "#", 2)
	maxID := snowflake
	for _, member := range members {
		maxID = member.User.ID
		// return matching usrname/discriminator combo
		if strings.ToLower(member.User.Username) == strings.ToLower(usernameParts[0]) && member.User.Discriminator == usernameParts[1] {
			return member.User.ID, member.Nick
		}
	}

	// recursion to keep searching
	return d.FindIDByUsernameStartingAt(username, maxID)
}

// Connect Needs to be called before any other API function work
func (d *DiscordAPI) Connect() {
	dg, err := discordgo.New("Bot " + d.Config.Token)
	if err != nil {
		Logger.Error("Unable to create discord connection", zap.Error(err))
		return
	}

	d.discord = dg
	dg.Open()
}

// Needs to be called to disconnect from discord
func (d *DiscordAPI) Shutdown() {
	Logger.Warn("Discord is shutting down")
	d.discord.Close()
}

// SendDM sends a DM to a user from the bot
func (d *DiscordAPI) SendDM(userID string, message string) {
	if ch, err := d.discord.UserChannelCreate(userID); err != nil {
		Logger.Error("Unable to create DM", zap.String("userID", userID), zap.Error(err))
	} else {
		_, err := d.discord.ChannelMessageSend(ch.ID, message)
		if err != nil {
			Logger.Error("unable to send DM", zap.String("userID", userID), zap.String("message", message), zap.Error(err))
		}
	}
}

func (d DiscordAPI) PostStreamMessage(sm messaging.StreamMessage) error {
	if d.discord == nil {
		return fmt.Errorf("discord API not connected")
	}
	if d.Config.StreamChannelId == "" {
		return fmt.Errorf("stream channel id not configured")
	}
	author := discordgo.MessageEmbedAuthor{
		Name: fmt.Sprintf("%s is live!", sm.Username),
	}
	thumbnail := discordgo.MessageEmbedThumbnail{
		URL: sm.UserLogo,
	}
	footer := discordgo.MessageEmbedFooter{
		Text:    fmt.Sprintf("%s | %s", sm.Platform, sm.Timestamp),
		IconURL: sm.PlatformLogo,
	}
	messageEmbed := discordgo.MessageEmbed{
		Description: sm.URL,
		Color:       sm.PlatformColorInt,
		URL:         sm.URL,
		Author:      &author,
		Thumbnail:   &thumbnail,
		Footer:      &footer,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Game",
				Value:  fmt.Sprintf("%s - %s", sm.Game, sm.Description),
				Inline: false,
			},
		},
	}
	_, err := d.discord.ChannelMessageSendEmbed(d.Config.StreamChannelId, &messageEmbed)
	return err
}
