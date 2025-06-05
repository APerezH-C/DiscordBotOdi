package main

import (
	"github.com/bwmarrin/discordgo"
	"strconv"
	"strings"
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

func handleWhoIsCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "Uso: `!quienes <ID de usuario>`")
		return
	}

	userID := args[0]
	user, err := s.User(userID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Usuario no encontrado")
		return
	}

	// Obtener fecha de creación
	creationTime, err := getDiscordCreationTime(user.ID)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "❌ Error al obtener fecha de creación")
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

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func getId(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Verificar si el mensaje es del bot
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Verificar si el comando es !quienes
	if strings.HasPrefix(m.Content, "!quienes") {
		// Tu ID de usuario
		const yourUserID = "431796013934837761"

		// Verificar si el autor del mensaje eres tú
		if m.Author.ID != yourUserID {
			s.ChannelMessageSend(m.ChannelID, "❌ No tienes permiso para usar este comando")
			return
		}

		args := strings.Fields(m.Content)[1:]
		handleWhoIsCommand(s, m, args)
	}
}
