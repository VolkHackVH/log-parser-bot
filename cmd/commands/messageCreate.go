package commands

import "github.com/bwmarrin/discordgo"

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	if m.Content == "!status" {
		s.ChannelMessageSend(m.ChannelID, "OK!")
	}
}
