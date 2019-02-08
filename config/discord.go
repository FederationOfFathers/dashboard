package config

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

type DiscordCfg struct {
	ClientId        string         `yaml:"appClientId"`
	Secret          string         `yaml:"appSecret"`
	Token           string         `yaml:"botToken"`
	StreamChannelId string         `yaml:"streamChannelId"`
	GuildId         string         `yaml:"guildId"`
	RoleCfg         DiscordRoleCfg `yaml:"roleConfig"`
}
