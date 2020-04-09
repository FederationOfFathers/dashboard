package bot

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const channelCommand = "!channel"
const inviteCommand = "!invite"
const leaveCommand = "!leave"

const memberCategoryID = "440865911910563861" //TODO config?
const memberChannelRoleFmt = "mc_%s"
const channelAssignName = "channel-assign"
const joinMemberChannelEmoji = "✅"
const leaveMemberChannelEmoji = "🛑"
const channelMaxIdleTime = time.Hour * 120 // TTL for a channel without new messages

// !channel channel_name
func (d *DiscordAPI) tempChannelCommandHandler(s *discordgo.Session, event *discordgo.MessageCreate) {

	// check that we're listening to the right guild
	if event.GuildID != d.Config.GuildId {
		return
	}

	// make sure we have the right format for the command
	// TODO we could provide an error message to the user :shrug:
	fields := strings.Fields(event.Content)
	if len(fields) <= 2 && fields[0] != channelCommand {
		return
	}

	// the channel name is the first field. anything after the space is dropped
	newChannelName := strings.ToLower(fields[1])

	//check if channel already exists
	if ok, chID := d.checkTextChannelPresenceByName(memberCategoryID, newChannelName); ok {
		message := fmt.Sprintf("The channel <#%s> already exists. Check <#%s> for the list of channels.", chID, d.memberChannelAssignID)
		if _, err := d.discord.ChannelMessageSend(event.ChannelID, message); err != nil {
			Logger.Error("unable to send intro message", zap.String("channel", event.ChannelID), zap.Error(err))
		}
		return
	}

	// create channel
	ch, err := d.newMemberChannel(newChannelName)
	if err != nil {
		Logger.Error("unable to create member channel", zap.String("channel_name", newChannelName), zap.Error(err))
		return
	}

	//create role
	mcRole, err := d.newMemberChannelRole(ch.Name, ch.ID)
	if err != nil {
		Logger.Error("unable to create role for temp channel", zap.String("channel", newChannelName), zap.String("id", ch.ID), zap.Error(err))
		return
	}

	// update channel permissions
	if err := d.updateMemberChannelPermissions(ch.Name, ch.ID, mcRole.ID); err != nil {
		Logger.Error("unable to apply channel permissions",
			zap.String("channelName", ch.Name),
			zap.String("channelID", ch.ID),
			zap.String("roleID", mcRole.ID),
			zap.Error(err))
	}

	// add role to user
	user := event.Author.ID
	if err := d.discord.GuildMemberRoleAdd(d.Config.GuildId, user, mcRole.ID); err != nil {
		Logger.Error("invite - unable to add role", zap.String("user", user), zap.String("role", mcRole.ID), zap.Error(err))
	}

	// send intro message
	if _, err := d.discord.ChannelMessageSend(ch.ID, fmt.Sprintf("This channel was created by <@%s>.\nType `!leave` in this channel to be removed.", event.Author.ID)); err != nil {
		Logger.Error("unable to send intro message", zap.String("channel", ch.ID), zap.Error(err))
	}

	d.setChannelAssignMessage()

}

// newMemberChannel creates a new member channel with default permissions denying everyone and allowing the bot
func (d DiscordAPI) newMemberChannel(channelName string) (*discordgo.Channel, error) {
	channelData := discordgo.GuildChannelCreateData{
		Name:     channelName,
		ParentID: memberCategoryID,
		Type:     discordgo.ChannelTypeGuildText,
		PermissionOverwrites: []*discordgo.PermissionOverwrite{
			{
				ID:   d.Config.GuildId, // deny everyone to view
				Type: "role",
				Deny: 0x00000400,
			},
			{
				ID:    d.Config.ClientId, // bot permissions
				Type:  "member",
				Allow: 0x00000010 + 0x00000040 + 0x00000800 + 0x00000400 + 0x00004000 + 0x00008000 + 0x00010000 + 0x00040000,
			},
		},
	}

	return d.discord.GuildChannelCreateComplex(d.Config.GuildId, channelData)
}

// checkTextChannelPresenceByName checks if a channel exists in the given category and returns true and the id of the channel if it exists
func (d DiscordAPI) checkTextChannelPresenceByName(categoryID, channelName string) (bool, string) {
	memberChannels := d.textChannelsInCategory(categoryID)
	for _, mc := range memberChannels {
		if mc.Name == channelName {
			return true, mc.ID
		}
	}

	return false, ""
}

func (d DiscordAPI) updateMemberChannelPermissions(channelName, channelID, channelRoleID string) error {

	// add overwrite perm with role. see https://discordapp.com/developers/docs/topics/permissions
	po := []*discordgo.PermissionOverwrite{
		{
			ID:    channelRoleID, // allow the new role to send text
			Type:  "role",
			Allow: 0x00000040 + 0x00000800 + 0x00000400 + 0x00004000 + 0x00008000 + 0x00010000 + 0x00040000,
		},
		{
			ID:   d.Config.GuildId, // deny everyone to view
			Type: "role",
			Deny: 0x00000400,
		},
		{
			ID:    d.Config.ClientId, // bot permissions
			Type:  "member",
			Allow: 0x00000040 + 0x00000800 + 0x00000400 + 0x00004000 + 0x00008000 + 0x00010000 + 0x00040000 + 0x00000010,
		},
	}

	// apply permissions
	_, err := d.discord.ChannelEditComplex(channelID, &discordgo.ChannelEdit{PermissionOverwrites: po, Position: 9})
	if err != nil {
		return fmt.Errorf("unable to set permissions for channel role %s: %s", channelName, err.Error())
	}

	return nil

}

// newMemberChannelRole creates a new channel role and applies the appropriate permissions to the channel
func (d DiscordAPI) newMemberChannelRole(channelName, channelID string) (*discordgo.Role, error) {
	//create role
	mcRole, err := d.discord.GuildRoleCreate(d.Config.GuildId)
	if err != nil {
		return nil, fmt.Errorf("unable to create member channel role" + err.Error())
	}

	// set role name (discordgo doesn't let you do it when you create it, yet)
	mcRole, err = d.discord.GuildRoleEdit(d.Config.GuildId, mcRole.ID, fmt.Sprintf(memberChannelRoleFmt, channelName), 0xFFFFFF, mcRole.Hoist, mcRole.Permissions, mcRole.Mentionable) //TODO change color?
	if err != nil {
		return nil, fmt.Errorf("unable to edit role %s: %s", mcRole.ID, err.Error())
	} else {
		Logger.Info("created new role", zap.String("id", mcRole.ID), zap.String("name", mcRole.Name))
	}

	return mcRole, nil
}

func (d DiscordAPI) addMemberChannelAssigner(channelID string) error {
	msg, err := d.discord.ChannelMessageSend(d.memberChannelAssignID, fmt.Sprintf("<#%s>", channelID))
	if err != nil {
		return fmt.Errorf("cannot create assignment message %s", err.Error())
	}

	err = d.discord.MessageReactionAdd(d.memberChannelAssignID, msg.ID, joinMemberChannelEmoji)
	err = d.discord.MessageReactionAdd(d.memberChannelAssignID, msg.ID, leaveMemberChannelEmoji)
	if err != nil {
		Logger.Error("unable to add reactions for channel", zap.String("channel", channelID), zap.Error(err))
	}

	return nil
}

func (d DiscordAPI) removeMemberChannelAssigner(channelID string) error {
	lastMessageID := ""
	for {
		messages, err := d.discord.ChannelMessages(d.memberChannelAssignID, 1, "", lastMessageID, "")
		if err != nil {
			return err
		}

		for _, message := range messages {
			lastMessageID = message.ID
			fmt.Println("message: " + lastMessageID)
			if channelID == channelIDFromChannelLink(message.Content) {
				err := d.discord.ChannelMessageDelete(d.memberChannelAssignID, message.ID)
				if err != nil {
					return fmt.Errorf("unable to delete mc assign message: %s", err.Error())
				}
			}
		}
		if len(messages) < 100 {
			break
		}
	}
	return nil
}

func (d *DiscordAPI) inviteTempChannelHandler(s *discordgo.Session, event *discordgo.MessageCreate) {
	if event.GuildID != d.Config.GuildId {
		return
	}
	fields := strings.Fields(event.Content)
	if len(fields) <= 2 && fields[0] != inviteCommand {
		return
	}

	// get channel
	ch, err := d.discord.Channel(event.ChannelID)
	if err != nil {
		Logger.Error("invite - unable to find channel", zap.String("channel", event.ChannelID), zap.Error(err))
		return
	}

	// get role
	role, err := d.FindGuildRoleByName(fmt.Sprintf(memberChannelRoleFmt, ch.Name))
	if err != nil {
		Logger.Error("invite - unable to find channel role", zap.String("channel_name", ch.Name), zap.Error(err))
		return
	}

	// parse user and add role
	user := userIDFromMention(fields[1])
	if err := d.discord.GuildMemberRoleAdd(d.Config.GuildId, user, role.ID); err != nil {
		Logger.Error("invite - unable to add role", zap.String("user", user), zap.String("role", role.ID), zap.Error(err))
	}
}

func (d *DiscordAPI) leaveTempChannelHandler(s *discordgo.Session, event *discordgo.MessageCreate) {
	if event.GuildID != d.Config.GuildId {
		return
	}
	fields := strings.Fields(event.Content)
	if len(fields) <= 1 && fields[0] != leaveCommand {
		return
	}

	// get channel
	ch, err := d.discord.Channel(event.ChannelID)
	if err != nil {
		Logger.Error("leave - unable to find channel", zap.String("channel", event.ChannelID), zap.Error(err))
		return
	}

	// get role
	role, err := d.FindGuildRoleByName(fmt.Sprintf(memberChannelRoleFmt, ch.Name))
	if err != nil {
		Logger.Error("leave - unable to find channel role", zap.String("channel_name", ch.Name), zap.Error(err))
		return
	}

	// parse user and add role
	user := event.Author.ID
	if err := d.discord.GuildMemberRoleRemove(d.Config.GuildId, user, role.ID); err != nil {
		Logger.Error("leave - unable to remove role", zap.String("user", user), zap.String("role", role.ID), zap.Error(err))
	}

	_ = d.discord.ChannelMessageDelete(event.ChannelID, event.Message.ID)

}

func (d *DiscordAPI) mindTempChannels() {
	hourTick := time.Tick(1 * time.Hour) // 1 hour mind, because this could have been started at any time
	d.setChannelAssignMessage()
	for {
		select {
		case <-hourTick:
			d.purgeOldTempChannels()
		}
	}
}

func (d *DiscordAPI) purgeOldTempChannels() {

	memberChannels := d.textChannelsInCategory(memberCategoryID)
	sort.Slice(memberChannels, func(i, j int) bool {
		return memberChannels[i].Name < memberChannels[j].Name
	})

	channelsRemoved := 0

	for _, channel := range memberChannels {
		// get channel data
		ch, err := d.discord.Channel(channel.ID)
		if err != nil {
			Logger.Error("could not get channel", zap.String("channel_id", channel.ID), zap.Error(err))
			continue
		}

		// skip non text channels
		if ch.Type != discordgo.ChannelTypeGuildText {
			continue
		}

		// skip the channel assign channel
		if ch.Name == channelAssignName {
			continue
		}

		// get last message in channel
		lastMessage, err := d.discord.ChannelMessage(ch.ID, ch.LastMessageID)
		if err != nil {
			// no last message means a new channel
			continue
		}

		// get time of last message
		lastMessageTime, err := lastMessage.Timestamp.Parse()
		if err != nil {
			Logger.Error("unable to parse timestamp. check for API changes", zap.String("message", string(lastMessage.Timestamp)), zap.Error(err))
			continue
		}

		// if more than 2 days, delete
		if time.Since(lastMessageTime) > channelMaxIdleTime {

			// find role id by name and delete
			role, err := d.FindGuildRoleByName(fmt.Sprintf(memberChannelRoleFmt, channel.Name))
			if err != nil {
				Logger.Error("unable to find role", zap.Error(err), zap.String("channel", channel.Name))
			} else {
				if err := d.discord.GuildRoleDelete(d.Config.GuildId, role.ID); err != nil {
					Logger.Error("unable to delete role", zap.Error(err), zap.String("role", role.ID))
				}
			}

			// channel delete
			_, err = d.discord.ChannelDelete(ch.ID)
			if err != nil {
				Logger.Error("unable to delete member channel", zap.String("channelName", ch.Name), zap.String("channelID", ch.ID), zap.Error(err))
				continue
			} else {
				Logger.Info("deleted channel", zap.String("channel", ch.ID), zap.String("name", ch.Name))
				channelsRemoved++
			}

		}
	}

	// reset channel assign message on deletion
	if channelsRemoved > 0 {
		d.setChannelAssignMessage()
	}
}

func (d *DiscordAPI) setChannelAssignMessage() {
	// find channel assign channel
	memberChannels := d.textChannelsInCategory(memberCategoryID)
	var assignChannel *Channel

	for i, ch := range memberChannels {
		if ch.Name == channelAssignName {
			assignChannel = ch

			// remove the channel assign channel
			memberChannels[i] = memberChannels[len(memberChannels)-1]
			memberChannels = memberChannels[:len(memberChannels)-1]
			break
		}
	}

	// sort member channels
	sort.Slice(memberChannels, func(i, j int) bool { return memberChannels[i].Name < memberChannels[j].Name })

	fmt.Printf("memberChannels: %v\n", memberChannels)

	if assignChannel == nil || assignChannel.ID == "" {
		Logger.Warn("unable to locate channel-assign channel", zap.String("channelID", assignChannel.ID))
		return
	}

	// remove all channel messages //todo need to redo
	d.clearChannelMessages(assignChannel.ID)

	introMessage := "Welcome to member channels.\n" + "" +
		"* To join a channel click the ✅ below a channel.\n" +
		"* Click the 🛑 to leave the channel.\n" +
		"* To create a new channel, type `!channel channel_name` with no spaces in the channel name. This can be done in any channel with FoF Bot, like <#%s>\n" +
		"* Channels are auto deleted after ~5 days of inactivity\n" +
		"---------------------------------------------------------------------------"

	d.discord.ChannelMessageSend(
		d.memberChannelAssignID,
		fmt.Sprintf(introMessage, d.Config.GuildId),
	)

	// create new messages
	for _, xch := range memberChannels {
		if err := d.addMemberChannelAssigner(xch.ID); err != nil {
			Logger.Error("unable to add assigner message", zap.String("channel", xch.Name), zap.Error(err))
		}
	}

}

func (d *DiscordAPI) clearChannelMessages(channelID string) {

	messages, err := d.discord.ChannelMessages(channelID, 50, "", "", "")
	if err != nil {
		Logger.Error("Unable to access channel messages", zap.String("channelId", channelID), zap.Error(err))
		return
	}
	for _, message := range messages {
		if message.Author.ID == d.Config.ClientId {
			err := d.discord.ChannelMessageDelete(channelID, message.ID)
			if err == nil {
				Logger.Info("Removed bot message", zap.String("messageId", message.ID))
			} else {
				Logger.Error("Unable to remove message", zap.String("messageId", message.ID), zap.Error(err))
			}

		}
	}
}

func (d DiscordAPI) handleMemberChannelRole(s *discordgo.Session, event *discordgo.MessageReactionAdd) {

	msg, err := d.discord.ChannelMessage(event.ChannelID, event.MessageID)
	if err != nil {
		Logger.Error("Unable to get message", zap.Error(err))
		return
	}

	userID := event.UserID

	channelID := channelIDFromChannelLink(msg.Content)
	ch, err := d.discord.Channel(channelID)
	if err != nil {
		Logger.Error("unable to get mc channel info", zap.Error(err), zap.String("channel", msg.Content))
		return
	}

	// get role
	role, err := d.FindGuildRoleByName(fmt.Sprintf(memberChannelRoleFmt, ch.Name))
	if err != nil {
		Logger.Error("unable to find channel role", zap.String("channel_name", ch.Name), zap.Error(err))
		return
	}

	switch event.Emoji.Name {
	case joinMemberChannelEmoji:
		// add role
		if err := d.discord.GuildMemberRoleAdd(d.Config.GuildId, userID, role.ID); err != nil {
			Logger.Error("unable to add role", zap.String("user", userID), zap.String("role", role.Name), zap.String("roleID", role.ID), zap.Error(err))
			return
		}
	case leaveMemberChannelEmoji:
		// remove role
		if err := d.discord.GuildMemberRoleRemove(d.Config.GuildId, userID, role.ID); err != nil {
			Logger.Error("unable to remove role", zap.String("user", userID), zap.String("role", role.Name), zap.String("roleID", role.ID), zap.Error(err))
			return
		}
	}

	d.removeReaction(event.ChannelID, event.MessageID, event.Emoji.Name, event.UserID)

}

func (d DiscordAPI) removeReaction(channelID, messageID, emojiID, userID string) {
	// remove the users reaction. If the add/remove failed, they can click it again to re-trigger
	err := d.discord.MessageReactionRemove(
		channelID,
		messageID,
		emojiID,
		userID,
	)

	if err != nil {
		Logger.Error("could not remove reaction from message",
			zap.String("user", userID),
			zap.String("emoji", emojiID),
			zap.Error(err))
	}
}

func (d DiscordAPI) handleMemberChannelRoleRemoval(s *discordgo.Session, event *discordgo.MessageReactionRemove) {
	msg, err := d.discord.ChannelMessage(event.ChannelID, event.MessageID)
	if err != nil {
		Logger.Error("Unable to get message", zap.Error(err))
		return
	}

	userID := event.UserID

	channelID := channelIDFromChannelLink(msg.Content)
	ch, err := d.discord.Channel(channelID)
	if err != nil {
		Logger.Error("mc role remove: unable to add role", zap.Error(err), zap.String("channel", msg.Content))
		return
	}

	// get role
	role, err := d.FindGuildRoleByName(fmt.Sprintf(memberChannelRoleFmt, ch.Name))
	if err != nil {
		Logger.Error("mc role remove: unable to find channel role", zap.String("channel_name", ch.Name), zap.Error(err))
		return
	}

	// remove role
	if err := d.discord.GuildMemberRoleRemove(d.Config.GuildId, userID, role.ID); err != nil {
		Logger.Error("mc role remove: unable to add role", zap.String("user", userID), zap.String("role", role.Name), zap.String("roleID", role.ID), zap.Error(err))
		return
	}
}
