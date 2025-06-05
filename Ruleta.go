package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"math/rand"
	"strings"
	"time"
)

var (
	gameActive = false
	barrel     []bool
	usedShots  = map[int]bool{}
)

func ruleta(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	content := strings.ToLower(m.Content)

	// Comandos del juego de ruleta rusa
	handleRuletaCommands(s, m, content)

	// Comandos del sistema de boste
	handlePuntosCommands(s, m, content)
}

func handleRuletaCommands(s *discordgo.Session, m *discordgo.MessageCreate, content string) {
	if strings.HasPrefix(content, "!cargar") {
		if gameActive {
			s.ChannelMessageSend(m.ChannelID, "Ya hay una partida activa. Usa `!terminar` para acabarla.")
			return
		}

		var balas int
		fmt.Sscanf(content, "!cargar %d", &balas)

		if balas < 1 || balas > 9 {
			s.ChannelMessageSend(m.ChannelID, "El número de balas debe estar entre 1 y 9.")
			return
		}

		iniciarJuego(balas)
		gameActive = true
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("🔫 Juego iniciado con %d bala(s) entre 9 ranuras de bala. ¡Prepárense!", balas))
		return
	}

	if content == "!disparar" {
		if !gameActive {
			s.ChannelMessageSend(m.ChannelID, "No hay una partida activa. Usa `!cargar <n>` para comenzar una.")
			return
		}

		if len(usedShots) >= 9 {
			s.ChannelMessageSend(m.ChannelID, "🔚 Todas las ranuras se han usado. Fin del juego.")
			gameActive = false
			return
		}

		disparo := disparar()
		if disparo {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("💥 %s ha muerto...", m.Author.Mention()))
			gameActive = false
			muteUser(s, m.GuildID, m.Author.ID, m.ChannelID)
		} else {
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("😮 %s ha sobrevivido.", m.Author.Mention()))
		}
		return
	}

	if content == "!terminar" {
		if !gameActive {
			s.ChannelMessageSend(m.ChannelID, "No hay un juego en curso.")
			return
		}
		gameActive = false
		s.ChannelMessageSend(m.ChannelID, "🛑 El juego ha sido terminado.")
	}
}

func disparar() bool {
	for {
		pos := rand.Intn(9)
		if !usedShots[pos] {
			usedShots[pos] = true
			return barrel[pos]
		}
	}
}

func iniciarJuego(balas int) {
	barrel = make([]bool, 9)
	usedShots = map[int]bool{}
	rand.Seed(time.Now().UnixNano())

	for balas > 0 {
		pos := rand.Intn(9)
		if !barrel[pos] {
			barrel[pos] = true
			balas--
		}
	}
}

func muteUser(s *discordgo.Session, guildID, userID, channelID string) {
	err := s.GuildMemberMute(guildID, userID, true)
	if err != nil {
		log.Println("Error silenciando usuario:", err)
		s.ChannelMessageSend(channelID, fmt.Sprintf("⚠️ No pude silenciar a <@%s> (¿tengo permisos?)", userID))
		return
	}

	_, err = s.ChannelMessageSend(channelID, fmt.Sprintf(
		"🔇 <@%s> ha sido silenciado por 1 minuto por perder la ruleta rusa!",
		userID,
	))
	if err != nil {
		log.Println("Error enviando mensaje de mute:", err)
	}

	go func() {
		time.Sleep(1 * time.Minute)
		err := s.GuildMemberMute(guildID, userID, false)
		if err != nil {
			log.Println("Error removiendo silencio:", err)
		} else {
			_, err = s.ChannelMessageSend(channelID, fmt.Sprintf(
				"🔊 <@%s> puede volver a hablar! El silencio ha terminado.",
				userID,
			))
			if err != nil {
				log.Println("Error enviando mensaje de unmute:", err)
			}
		}
	}()
}
