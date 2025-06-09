package main

import "github.com/bwmarrin/discordgo"

func handleHelpCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "📜 Lista de Comandos de BosteBot",
		Color:       0x00ff00, // Verde
		Description: "Aquí tienes todos los comandos disponibles:",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "🎲 Comandos de Juego",
				Value: "`/bostedice <under/over> <número 1-95> <cantidad>` - Juega a los dados eligiendo porcentaje y posicion\n" +
					"`/bostestats` - Muestra tus estadísticas personales de apuestas\n" +
					"`/apuesta <win|lose> <cantidad>` - Apuesta puntos en un juego\n" +
					"`/revertirapuesta` - Revierte la apuesta\n" +
					"`/cargar <1-9>` - Recarga la ruleta rusa\n" +
					"`/disparar` - Dispara\n" +
					"`/terminar` - Finaliza la ruleta rusa\n",
				Inline: false,
			},
			{
				Name: "🛒 Comandos de Tienda",
				Value: "`/bostecompra <nombre_objeto>` - Compra un objeto de la tienda\n" +
					"`/bostetienda` - Muestra los objetos disponibles\n" +
					"`/bosteinventario` - Muestra tus objetos comprados",
				Inline: false,
			},
			{
				Name:   "📊 Comandos de Puntos",
				Value:  "`/bostes` - Muestra tus puntos actuales",
				Inline: false,
			},
			{
				Name: "🛠️ Otros Comandos",
				Value: "`/quienes <ID>` - Muestra información de un usuario (solo admin)\n" +
					"`/bosteHelp` - Muestra esta ayuda\n" +
					"`/bosteSeed` - Genera una nueva seed y muestra la anterior\n" +
					"`/verify <server_seed> <client_seed> <nonce>` - Verifica el resultado del dice\n" +
					"`/notificaciones on` - Activa las notificaciones\n" +
					"`/notificaciones off` - Desactiva las notificaciones\n" +
					"`/notificaciones` - Muestra el estado de tu suscripcion",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Usa !bosteHelp <comando> para más información sobre un comando específico",
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
