package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
)

const notificationRoleID = "1381300449264537620" // Reemplaza con el ID real del rol

func handleNotificationSubscribe(s *discordgo.Session, m *discordgo.MessageCreate) {
	err := s.GuildMemberRoleAdd(m.GuildID, m.Author.ID, notificationRoleID)
	if err != nil {
		log.Printf("Error añadiendo rol: %v", err)
		s.ChannelMessageSend(m.ChannelID, "❌ No pude añadirte el rol de notificaciones. ¿Tengo los permisos necesarios?")
		return
	}
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ <@%s>, ahora recibirás notificaciones importantes!", m.Author.ID))
}

func handleNotificationUnsubscribe(s *discordgo.Session, m *discordgo.MessageCreate) {
	err := s.GuildMemberRoleRemove(m.GuildID, m.Author.ID, notificationRoleID)
	if err != nil {
		log.Printf("Error removiendo rol: %v", err)
		s.ChannelMessageSend(m.ChannelID, "❌ No pude quitarte el rol de notificaciones. ¿Tengo los permisos necesarios?")
		return
	}
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("🔕 <@%s>, ya no recibirás notificaciones importantes.", m.Author.ID))
}

func showNotificationStatus(s *discordgo.Session, m *discordgo.MessageCreate) {
	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		log.Printf("Error obteniendo miembro: %v", err)
		s.ChannelMessageSend(m.ChannelID, "❌ No pude verificar tu estado de notificaciones.")
		return
	}

	hasRole := false
	for _, role := range member.Roles {
		if role == notificationRoleID {
			hasRole = true
			break
		}
	}

	if hasRole {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("🔔 <@%s>, actualmente estás suscrito a las notificaciones importantes.", m.Author.ID))
	} else {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("🔕 <@%s>, no estás suscrito a las notificaciones importantes. Usa `!notificaciones on` para activarlas.", m.Author.ID))
	}
}
