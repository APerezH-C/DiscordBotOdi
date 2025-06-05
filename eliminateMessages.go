package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strings"
)

func handleClearBotMessages(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID != "431796013934837761" {
		s.ChannelMessageSend(m.ChannelID, "No tienes permiso.")
		return
	}

	// Obtener últimos 100 mensajes del canal
	messages, err := s.ChannelMessages(m.ChannelID, 100, "", "", "")
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Error al obtener mensajes.")
		return
	}

	var toDelete []string

	// Filtrar solo los mensajes del bot o que comienzan con comandos del bot
	for _, msg := range messages {
		if msg.Author.ID == s.State.User.ID || strings.HasPrefix(msg.Content, "!bosteDice") || strings.HasPrefix(msg.Content, "!verify") {
			toDelete = append(toDelete, msg.ID)
			if len(toDelete) >= 50 {
				break
			}
		}
	}

	// Eliminar mensajes en lotes de 100 máximo (Discord limita bulk delete)
	if len(toDelete) > 0 {
		err = s.ChannelMessagesBulkDelete(m.ChannelID, toDelete)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Error al eliminar mensajes.")
			return
		}
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ Se eliminaron %d mensajes del bot.", len(toDelete)))
	} else {
		s.ChannelMessageSend(m.ChannelID, "No se encontraron mensajes del bot para eliminar.")
	}
}
