package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

var (
	dbFile        = "points.json"
	checkInterval = 10 * time.Minute
)

// Estructuras de datos
type UserPoints struct {
	Points map[string]float64 `json:"points"`
	mu     sync.Mutex
}

func (up *UserPoints) Load(filename string) error {
	up.mu.Lock()
	defer up.mu.Unlock()

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			up.Points = make(map[string]float64)
			return nil
		}
		return err
	}

	return json.Unmarshal(data, up)
}

func (up *UserPoints) Save(filename string) error {
	up.mu.Lock()
	defer up.mu.Unlock()

	data, err := json.MarshalIndent(up, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

func (up *UserPoints) Add(userID string, points float64) bool {
	up.mu.Lock()
	defer up.mu.Unlock()

	if up.Points == nil {
		up.Points = make(map[string]float64)
	}

	// Verificar que no quedará balance negativo (excepto para restas)
	if points < 0 && up.Points[userID]+points < 0 {
		return false
	}

	up.Points[userID] += points
	return true
}

func (up *UserPoints) Get(userID string) float64 {
	up.mu.Lock()
	defer up.mu.Unlock()

	return up.Points[userID]
}

func (up *UserPoints) Set(userID string, amount float64) { // Nueva función para establecer valores
	up.mu.Lock()
	defer up.mu.Unlock()

	if up.Points == nil {
		up.Points = make(map[string]float64)
	}

	up.Points[userID] = amount
}

func handlePuntosCommands(s *discordgo.Session, m *discordgo.MessageCreate, content string) {
	if content == "!bostes" {
		puntos := userPoints.Get(m.Author.ID)
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%s> tienes %.2f bostes.", m.Author.ID, puntos))
	}

	if content == "!ranking" {
		userPoints.mu.Lock()
		defer userPoints.mu.Unlock()

		msg := "**Ranking de bostes:**\n"
		for userID, pts := range userPoints.Points {
			msg += fmt.Sprintf("<@%s>: %.2f bostes\n", userID, pts)
		}
		s.ChannelMessageSend(m.ChannelID, msg)
	}
}

func voiceChannelChecker(s *discordgo.Session) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		for _, guild := range s.State.Guilds {
			// Obtener estados de voz directamente del estado del guild
			voiceStates := guild.VoiceStates

			for _, vs := range voiceStates {
				// Verificar si el usuario no está silenciado
				if !vs.Mute && !vs.SelfMute && !vs.Deaf && !vs.SelfDeaf {
					userPoints.Add(vs.UserID, 10)
					log.Printf("Bostes añadidos a %s (ahora tiene %.2f)\n", vs.UserID, userPoints.Get(vs.UserID))
				}
			}
		}

		err := userPoints.Save(dbFile)
		if err != nil {
			log.Printf("Error al guardar bostes: %v\n", err)
		}
	}
}
