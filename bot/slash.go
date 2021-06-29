package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

// slashCommandHandlers handles and routes all incoming interaction based commands based on the interaction type
func (d *DiscordAPI) slashCommandHandlers(s *discordgo.Session, i *discordgo.InteractionCreate) {

	Logger.With(zap.Any("interaction", i.Interaction)).Debug("slash command interaction")

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		d.slashApplicationCommanInteractionHandler(s, i)
	case discordgo.InteractionMessageComponent:
		d.slashMessageComponentHandler(s, i)
	}

}

// slashMessageComponentHandler handles and routes interactions with components, such as button clicks
func (d *DiscordAPI) slashMessageComponentHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {

	customID := i.Interaction.MessageComponentData().CustomID

	switch customID[:strings.Index(customID, ":")] {
	case "stream":
		d.slashStreamComponentHandler(s, i)
	}
}

// slashApplicationCommanInteractionHandler handles and routes initial slash command messages
func (d *DiscordAPI) slashApplicationCommanInteractionHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {

	switch i.Interaction.ApplicationCommandData().Name {
	case "stream":
		d.slashStreamHandler(s, i)
	}
}
