package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"math/rand"
	"time"
)

var (
	gameActive = false
	barrel     []bool
	usedShots  = map[int]bool{}
)

func handleRuletaCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Obtener los datos de la interacción
	data := i.ApplicationCommandData()

	switch data.Name {
	case "cargar":
		// Verificar si ya hay un juego activo
		if gameActive {
			respondInteraction(s, i, "Ya hay una partida activa. Usa `/terminar` para acabarla.", true)
			return
		}

		// Obtener el número de balas del comando
		var balas int
		if len(data.Options) > 0 {
			if opt, ok := data.Options[0].Value.(float64); ok {
				balas = int(opt)
			}
		}

		// Validar el número de balas
		if balas < 1 || balas > 9 {
			respondInteraction(s, i, "El número de balas debe estar entre 1 y 9.", true)
			return
		}

		// Iniciar el juego
		iniciarJuego(balas)
		gameActive = true
		respondInteraction(s, i, fmt.Sprintf("🔫 Juego iniciado con %d bala(s) entre 9 ranuras. ¡Prepárense!", balas), false)

	case "disparar":
		// Verificar si hay un juego activo
		if !gameActive {
			respondInteraction(s, i, "No hay una partida activa. Usa `/cargar <n>` para comenzar una.", true)
			return
		}

		// Verificar si se han usado todas las ranuras
		if len(usedShots) >= 9 {
			respondInteraction(s, i, "🔚 Todas las ranuras se han usado. Fin del juego.", false)
			gameActive = false
			return
		}

		// Realizar el disparo
		disparo := disparar()
		if disparo {
			respondInteraction(s, i, fmt.Sprintf("💥 %s ha muerto...", i.Member.User.Mention()), false)
			gameActive = false
			muteUser(s, i.GuildID, i.Member.User.ID, i.ChannelID)
		} else {
			respondInteraction(s, i, fmt.Sprintf("😮 %s ha sobrevivido.", i.Member.User.Mention()), false)
		}

	case "terminar":
		// Verificar si hay un juego activo
		if !gameActive {
			respondInteraction(s, i, "No hay un juego en curso.", true)
			return
		}
		gameActive = false
		respondInteraction(s, i, "🛑 El juego ha sido terminado.", false)
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
