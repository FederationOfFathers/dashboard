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

	//add handlers
	discordApi.discord.AddHandler(discordApi.roleAssignmentHandler)
	discordApi.discord.AddHandler(discordApi.teamCommandHandler)
	discordApi.discord.AddHandler(discordApi.verifiedEventsHandler)

	go discordApi.mindTempChannels()

	//go discordApi.setChannelAssignMessage()

	discordApi.discord.UpdateStatus(0, "ui.fofgaming.com | !team")

	// data cache
	data.load()
	populateLists()
	go mindLists()

	return discordApi

}

// verifiedEventsHandler checks if the user is verified before running the handler
func (d *DiscordAPI) verifiedEventsHandler(s *discordgo.Session, event *discordgo.MessageCreate) {
	if event.GuildID != d.Config.GuildId {
		return
	}
	fields := strings.Fields(event.Content)
	if len(fields) <= 1 {
		return
	}

	switch fields[0] {
	case channelCommand:
		d.tempChannelCommandHandler(s, event)
	case inviteCommand:
		d.inviteTempChannelHandler(s, event)
	case leaveCommand:
		d.leaveTempChannelHandler(s, event)
	}
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
	guildChannels := &GuildChannels{
		Categories: []ChannelCategory{
			{ID: "", Name: ""},
		},
	}

	channels, err := d.discord.GuildChannels(d.Config.GuildId)
	if err != nil {
		Logger.Error("unable to get guild channels", zap.Error(err))
	}

	var textCh []discordgo.Channel
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

func (d *DiscordAPI) teamCommandHandler(s *discordgo.Session, event *discordgo.MessageCreate) {
	if event.GuildID != d.Config.GuildId {
		return
	}
	switch event.Content {
	case "!team":
		d.sendTeamToolLink(event)
	}
}

func (d DiscordAPI) sendTeamToolLink(m *discordgo.MessageCreate) {
	d.discord.ChannelMessageSend(m.ChannelID, "FoF Team Tool -> https://ui.fofgaming.com")
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

func (d *DiscordAPI) FindGuildRoleByName(name string) (*discordgo.Role, error) {
	roles, err := d.discord.GuildRoles(d.Config.GuildId)
	if err != nil {
		return nil, err
	}

	for _, role := range roles {
		if role.Name == name {
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

	loc, _ := time.LoadLocation("America/New_York") // show times in EST

	openSpots := e.Need - len(members)

	messageEmbed := discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s has created a new event", host),
		Description: e.Title,
		Color:       0x007BFF,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Date",
				Value:  e.When.In(loc).Format("1/2, 15:04 PM MST"),
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

func saveChannelsToDB(gc *GuildChannels) error {
	var err error
	for _, cat := range gc.Categories {
		for _, ch := range cat.Channels {
			dbEventChannel := &db.EventChannel{
				ID:                  ch.ID,
				ChannelCategoryName: cat.Name,
				ChannelCategoryID:   cat.ID,
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

func userIDFromMention(mention string) string {
	return strings.Trim(mention[2:len(mention)-1], "!")
}

func channelIDFromChannelLink(channelLink string) string {
	return strings.Trim(channelLink[2:len(channelLink)-1], "!")
}

func (d *DiscordAPI) textChannelsInCategory(categoryID string) []*Channel {

	channels := d.guildChannels()
	// get channels of member channels category
	for _, category := range channels.Categories {
		if category.ID == memberCategoryID {
			return category.Channels
		}
	}

	return []*Channel{}
}
