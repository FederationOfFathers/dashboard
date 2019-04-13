package bot

import (
	"fmt"
	"go.uber.org/zap"
)

func (d DiscordAPI) addRoleToUser(userId string, roleId string) bool {
	err := d.discord.GuildMemberRoleAdd(d.Config.GuildId, userId, roleId)
	if err != nil {
		Logger.Error("could not add role",
			zap.String("userId", userId),
			zap.String("roleId", roleId),
			zap.Error(err))
		return false
	}

	Logger.Info("added role to user",
		zap.String("userId", userId),
		zap.String("roleId", roleId),
	)
	return true

}

func (d DiscordAPI) removeRoleFromUser(userId string, roleId string) bool {
	err := d.discord.GuildMemberRoleRemove(d.Config.GuildId, userId, roleId)
	if err != nil {
		Logger.Error("could not remove role", zap.Error(err))
		return false
	}

	Logger.Info("removed role from user",
		zap.String("userId", userId),
		zap.String("roleId", roleId),
	)
	return true

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
