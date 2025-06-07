package main

import (
	"RuletaRusaOdi/database"
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"sync"
	"time"
)

var (
	checkInterval = 10 * time.Minute
)

// Estructuras de datos
type UserPoints struct {
	Points map[string]float64 `bson:"points"`
	mu     sync.Mutex
}

func (up *UserPoints) Load() error {
	up.mu.Lock()
	defer up.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("points")
	var result struct {
		Points map[string]float64 `bson:"points"`
	}

	err := collection.FindOne(ctx, bson.M{}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Documento no existe, se mantiene el mapa vacío inicial
			return nil
		}
		return fmt.Errorf("error al cargar puntos: %w", err)
	}

	if result.Points != nil {
		up.Points = result.Points
	}
	return nil
}

func (up *UserPoints) Save() error {
	up.mu.Lock()
	defer up.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("points")
	_, err := collection.UpdateOne(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{"points": up.Points}},
		options.Update().SetUpsert(true),
	)

	if err != nil {
		return fmt.Errorf("error al guardar puntos: %w", err)
	}
	return nil
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
	if m.Author.Bot { // Ignorar mensajes de otros bots
		return
	}

	switch content {
	case "!bostes":
		er := userPoints.Load()
		if er != nil {
			log.Fatal(er)
		}
		puntos := userPoints.Get(m.Author.ID)
		_, err := s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<@%s> tienes %.2f bostes.", m.Author.ID, puntos))
		if err != nil {
			log.Printf("Error enviando mensaje: %v", err)
		}

	case "!ranking":
		er := userPoints.Load()
		if er != nil {
			log.Fatal(er)
		}
		userPoints.mu.Lock()
		defer userPoints.mu.Unlock()

		msg := "**Ranking de bostes:**\n"
		for userID, pts := range userPoints.Points {
			msg += fmt.Sprintf("<@%s>: %.2f bostes\n", userID, pts)
		}

		_, err := s.ChannelMessageSend(m.ChannelID, msg)
		if err != nil {
			log.Printf("Error enviando ranking: %v", err)
		}
	}
}

func voiceChannelChecker(s *discordgo.Session) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		for _, guild := range s.State.Guilds {
			for _, vs := range guild.VoiceStates {
				if !vs.Mute && !vs.SelfMute && !vs.Deaf && !vs.SelfDeaf {
					if added := userPoints.Add(vs.UserID, 5); added {
						log.Printf("Bostes añadidos a %s (ahora tiene %.2f)\n", vs.UserID, userPoints.Get(vs.UserID))
					}
				}
			}
		}

		if err := userPoints.Save(); err != nil {
			log.Printf("Error al guardar bostes: %v\n", err)
		}
	}
}
