package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/FederationOfFathers/dashboard/db"
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

type GuildChannels struct {
	Categories []ChannelCategory
}

type ChannelCategory struct {
	ID       string
	Name     string
	Channels []*Channel
}

type Channel struct {
	ID   string
	Name string
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

// MindGuild starts routines to monitor Discord things like channels
func (d *DiscordAPI) MindGuild() {
	// get channels and save them to the db
	go d.mindChannelList()

}

func (d *DiscordAPI) mindChannelList() {
	ticker := time.Tick(1 * time.Minute)

	for {
		select {
		case <-ticker:
			channels := d.guildChannels()
			if err := saveChannelsToDB(channels); err == nil {
				// purge old channels if no errors on save
				DB.PurgeOldEventChannels(-1 * time.Minute)
			}
		}
	}

}

func (d *DiscordAPI) guildChannels() *GuildChannels {
	guildChannels := &GuildChannels{}
	channels, err := d.discord.GuildChannels(d.Config.GuildId)
	if err != nil {
		Logger.Error("unable to get guild channels", zap.Error(err))
	}

	var textCh = []discordgo.Channel{}
	// get the categories
	for _, ch := range channels {
		switch ch.Type {
		case discordgo.ChannelTypeGuildCategory: // create categories
			category := &ChannelCategory{
				ID:   ch.ID,
				Name: ch.Name,
			}
			guildChannels.Categories = append(guildChannels.Categories, *category)
		case discordgo.ChannelTypeGuildText: // store text channels for iteration
			textCh = append(textCh, *ch)
		}

	}

	// sort the text channels
	for _, ch := range textCh {
		parentID := ch.ParentID
		if parentID == "" { // skip chanenls without category
			continue
		}
		for i, cat := range guildChannels.Categories { // find a the parent category and add it
			if cat.ID == parentID {
				tCh := &Channel{
					ID:   ch.ID,
					Name: ch.Name,
				}
				guildChannels.Categories[i].Channels = append(guildChannels.Categories[i].Channels, tCh)
			}
		}
	}

	return guildChannels
}

func (d DiscordAPI) roleAssignmentHandler(s *discordgo.Session, event *discordgo.MessageReactionAdd) {

	// skip if the event was from the bot/app
	if event.UserID == d.Config.ClientId {
		return
	}

	// only handle if the message is one we have configured
	if roles, ok := d.assignmentMsgs[event.MessageID]; ok {

		// Unicode emojis use the unicode character (name) as the id. Others use the name and integer as the id.
		emojiId := event.Emoji.Name
		if event.Emoji.ID != "" {
			emojiId = fmt.Sprintf(":%s:%s", event.Emoji.Name, event.Emoji.ID)
		}

		if roleId, ok := roles[emojiId]; ok {
			// get the user from the server/guild
			member, err := d.discord.GuildMember(d.Config.GuildId, event.UserID)
			if err != nil {
				Logger.Error("Unable to get member", zap.Error(err))
				return
			}
		}
	}

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

func (d *DiscordAPI) PostNewEventMessage(e *db.Event) error {
	if d.discord == nil {
		return fmt.Errorf("discord API not connected")
	}
	var host string
	var members []string
	for _, eMember := range e.Members {
		m, err := DB.MemberByID(eMember.MemberID)
		if err != nil {
			Logger.Error("unable to get member", zap.Int("id", eMember.MemberID), zap.Error(err))
		}
		if eMember.Type == db.EventMemberTypeHost {
			host = m.Name
		}
		members = append(members, m.Name)
	}

	openSpots := e.Need - len(members)

	messageEmbed := discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s has created a new event", host),
		Description: e.Title,
		Color:       0x007BFF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Date",
				Value:  e.When.Format("1/2, 15:04 PM"),
				Inline: true,
			},
			{
				Name:   "Players Needed",
				Value:  strconv.Itoa(e.Need),
				Inline: true,
			},
			{
				Name:   "Open Spots",
				Value:  strconv.Itoa(openSpots),
				Inline: true,
			},
			{
				Name:   fmt.Sprintf("Going (%d)", len(members)),
				Value:  strings.Join(members, " "),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Go to https://ui.fofgaming.com to join",
		},
	}

	_, err := d.discord.ChannelMessageSendEmbed(e.EventChannel.ID, &messageEmbed)
	if err != nil {
		Logger.Error("unable to send discord message", zap.Error(err), zap.Any("message", messageEmbed))
	}

	return err

}

// removes all messages entered by this bot in the channel. Uses the ClientID of the bot/app
func (d *DiscordAPI) clearRoleChannel() {
	channelId := d.Config.RoleCfg.ChannelId

	messages, err := d.discord.ChannelMessages(channelId, 50, "", "", "")
	if err != nil {
		Logger.Error("Unable to access channel messages", zap.String("channelId", channelId), zap.Error(err))
		return
	}
	for _, message := range messages {
		if message.Author.ID == d.Config.ClientId {
			err := d.discord.ChannelMessageDelete(channelId, message.ID)
			if err == nil {
				Logger.Info("Removed bot message", zap.String("messageId", message.ID))
			} else {
				Logger.Error("Unable to remove message", zap.String("messageId", message.ID), zap.Error(err))
			}

		}
	}
}

// creates messages in the channel and adds the emojis
func (d *DiscordAPI) createRoleMessages() {
	d.assignmentMsgs = make(map[string]map[string]string)
	for _, group := range d.Config.RoleCfg.EmojiRoleGroups {
		messageEmbed := discordgo.MessageEmbed{
			Title:       group.MessageTitle,
			Description: group.MessageBody,
			Color:       15581239,
		}
		// create the message
		message, err := d.discord.ChannelMessageSendEmbed(d.Config.RoleCfg.ChannelId, &messageEmbed)
		if err != nil {
			Logger.Error("Unable to create message", zap.Error(err))
			continue
		}
		Logger.Info("Added role message", zap.String("message", group.MessageTitle), zap.String("messageId", message.ID))

		// add the emojis
		emojiRoles := make(map[string]string)
		for _, role := range group.Roles {
			emojiRoles[role.EmojiId] = role.RoleId
			err := d.discord.MessageReactionAdd(d.Config.RoleCfg.ChannelId, message.ID, role.EmojiId)
			if err != nil {
				Logger.Error("Unable to add emoji to message",
					zap.String("emojiId", role.EmojiId),
					zap.String("messageId", message.ID),
					zap.Error(err),
				)
			}
		}

		d.assignmentMsgs[message.ID] = emojiRoles
	}
}

func (d DiscordAPI) addRoleToUser(userId string, roleId string) {
	err := d.discord.GuildMemberRoleAdd(d.Config.GuildId, userId, roleId)
	if err != nil {
		Logger.Error("could not add role",
			zap.String("userId", userId),
			zap.String("roleId", roleId),
			zap.Error(err))
	} else {
		Logger.Info("added role to user",
			zap.String("userId", userId),
			zap.String("roleId", roleId),
		)
	}
}

func (d DiscordAPI) removeRoleFromUser(userId string, roleId string) {
	err := d.discord.GuildMemberRoleRemove(d.Config.GuildId, userId, roleId)
	if err != nil {
		Logger.Error("could not remove role", zap.Error(err))
	} else {
		Logger.Info("removed role from user",
			zap.String("userId", userId),
			zap.String("roleId", roleId),
		)
	}
}
func (d *DiscordAPI) listRoles() {
	roles, err := d.discord.GuildRoles(d.Config.GuildId)

	if err != nil {
		Logger.Error("Could not get roles", zap.Error(err))
		return
	}

	for i, role := range roles {
		Logger.Info(fmt.Sprintf("Role %d", i), zap.Any("role", role))
	}
}

func saveChannelsToDB(gc *GuildChannels) error {
	var err error
	for _, cat := range gc.Categories {
		for _, ch := range cat.Channels {
			dbEventChannel := &db.EventChannel{
				ID:                  ch.ID,
				ChannelCategoryName: cat.Name,
				ChannelName:         ch.Name,
				UpdatedAt:           time.Now(),
			}

			if err1 := DB.SaveEventChannel(dbEventChannel); err1 != nil {
				err = err1
				Logger.Error("unable to save event channel data", zap.Error(err1))
			}
		}
	}

	return err
}
