package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

type EmojiRole struct {
	EmojiId string `yaml:"emojiId"`
	RoleId  string `yaml:"roleId"`
}

type DiscordEmojiRoleGroup struct {
	MessageTitle string      `yaml:"messageTitle"`
	MessageBody  string      `yaml:"messageBody"`
	Roles        []EmojiRole `yaml:"roles"`
}

type DiscordRoleCfg struct {
	ChannelId       string                  `yaml:"channelId"`
	EmojiRoleGroups []DiscordEmojiRoleGroup `yaml:"emojiRoles"`
}

func (d *DiscordAPI) StartRoleHandlers() {
	d.listRoles()
	d.clearRoleChannel()
	d.createRoleMessages()
	d.discord.AddHandler(d.roleAssignmentHandler)
	Logger.Info("Role assignment handler started")
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

			userHadRole := false
			// if the member has the mapped role, remove it
			for _, id := range member.Roles {
				if id == roleId {
					d.removeRoleFromUser(event.UserID, roleId)
					userHadRole = true // true, even if an attempt was made and failed
				}
			}

			// if the member does not have the mapped role, add it
			if !userHadRole {
				d.addRoleToUser(event.UserID, roleId)
			}

			// DM the user the confirmation
			role, err := d.FindGuildRole(roleId)
			if err != nil {
				Logger.Error("Unable to find role", zap.Error(err))
			}
			if userHadRole {
				d.SendDM(event.UserID, fmt.Sprintf("The %s role has been removed", role.Name))
			} else {
				d.SendDM(event.UserID, fmt.Sprintf("You now have the %s role!", role.Name))
			}

		}

		// remove the users reaction. If the add/remove failed, they can click it again to re-trigger
		err := d.discord.MessageReactionRemove(
			event.ChannelID,
			event.MessageID,
			emojiId,
			event.UserID,
		)

		if err != nil {
			Logger.Error("could not remove reaction from message",
				zap.String("user", event.UserID),
				zap.String("emoji", emojiId),
				zap.Error(err))
		}
	}

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