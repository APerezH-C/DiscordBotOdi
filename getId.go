package main

import (
	"github.com/bwmarrin/discordgo"
	"strconv"
	"time"
)

func getDiscordCreationTime(userID string) (time.Time, error) {
	// Convertir el ID a entero
	id, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	// Discord Epoch (01-01-2015)
	const discordEpoch = 1420070400000
	// Obtener timestamp (los primeros 42 bits del ID)
	timestamp := (id >> 22) + discordEpoch

	// Convertir a tiempo (dividir por 1000 porque está en milisegundos)
	return time.Unix(timestamp/1000, 0), nil
}

func handleWhoIsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	const yourUserID = "431796013934837761"

	// Verificar permisos
	if i.Member.User.ID != yourUserID {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ No tienes permiso para usar este comando.",
				Flags:   1 << 6,
			},
		})
		return
	}

	userID := i.ApplicationCommandData().Options[0].StringValue()
	user, err := s.User(userID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Usuario no encontrado.",
				Flags:   1 << 6,
			},
		})
		return
	}

	// Obtener fecha de creación
	creationTime, err := getDiscordCreationTime(user.ID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Error al obtener fecha de creación.",
				Flags:   1 << 6,
			},
		})
		return
	}

	// Crear embed
	embed := &discordgo.MessageEmbed{
		Title: "Información de Usuario",
		Color: 0x5865F2,
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: user.AvatarURL("256"),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Username",
				Value: user.String(),
			},
			{
				Name:  "ID",
				Value: user.ID,
			},
			{
				Name:  "Cuenta creada",
				Value: creationTime.Format("02/01/2006 15:04 MST"),
			},
			{
				Name:  "Antigüedad",
				Value: time.Since(creationTime).Round(time.Hour * 24).String(),
			},
			{
				Name:  "Es Bot",
				Value: strconv.FormatBool(user.Bot),
			},
		},
	}

	// Enviar embed como respuesta
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
