package main

import "github.com/bwmarrin/discordgo"

func handleHelpCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	embed := &discordgo.MessageEmbed{
		Title:       "📜 Lista de Comandos de BosteBot",
		Color:       0x00ff00, // Verde
		Description: "Aquí tienes todos los comandos disponibles:",
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "🎲 Comandos de Juego",
				Value: "`!bosteDice <under/over> <número 1-95> <cantidad>` - Juega a los dados eligiendo porcentaje y posicion\n" +
					"`!apuesta <cantidad>` - Apuesta puntos en un juego\n" +
					"`!revertirApuesta` - Revierte la apuesta\n" +
					"`!cargar <1-9>` - Recarga la ruleta rusa",
				Inline: false,
			},
			{
				Name: "🛒 Comandos de Tienda",
				Value: "`!bosteCompra <nombre_objeto>` - Compra un objeto de la tienda\n" +
					"`!bosteTienda` - Muestra los objetos disponibles\n" +
					"`!bosteInventario` - Muestra tus objetos comprados",
				Inline: false,
			},
			{
				Name:   "📊 Comandos de Puntos",
				Value:  "`!bostes` - Muestra tus puntos actuales",
				Inline: false,
			},
			{
				Name: "🛠️ Otros Comandos",
				Value: "`!quienes <ID>` - Muestra información de un usuario (solo admin)\n" +
					"`!bosteHelp` - Muestra esta ayuda" +
					"`!bosteSeed` - Genera una nueva seed y muestra la anterior" +
					"`!verify <server_seed> <client_seed> <nonce>` - Verifica el resultado del dice",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Usa !bosteHelp <comando> para más información sobre un comando específico",
		},
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}
