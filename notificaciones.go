package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
)

const notificationRoleID = "1381300449264537620" // Reemplaza con el ID real del rol

func handleNotificationSubscribe(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	err := s.GuildMemberRoleAdd(i.GuildID, userID, notificationRoleID)
	if err != nil {
		log.Printf("Error añadiendo rol: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ No pude añadirte el rol de notificaciones. ¿Tengo los permisos necesarios?",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ <@%s>, ahora recibirás notificaciones importantes!", userID),
		},
	})
}

func handleNotificationUnsubscribe(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	err := s.GuildMemberRoleRemove(i.GuildID, userID, notificationRoleID)
	if err != nil {
		log.Printf("Error removiendo rol: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ No pude quitarte el rol de notificaciones. ¿Tengo los permisos necesarios?",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("🔕 <@%s>, ya no recibirás notificaciones importantes.", userID),
		},
	})
}

func showNotificationStatus(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	member, err := s.GuildMember(i.GuildID, userID)
	if err != nil {
		log.Printf("Error obteniendo miembro: %v", err)
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ No pude verificar tu estado de notificaciones.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	hasRole := false
	for _, role := range member.Roles {
		if role == notificationRoleID {
			hasRole = true
			break
		}
	}

	msg := ""
	if hasRole {
		msg = fmt.Sprintf("🔔 <@%s>, actualmente estás suscrito a las notificaciones importantes.", userID)
	} else {
		msg = fmt.Sprintf("🔕 <@%s>, no estás suscrito a las notificaciones importantes. Usa `/notificaciones on` para activarlas.", userID)
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})

}
