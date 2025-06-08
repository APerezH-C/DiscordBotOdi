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
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	checkInterval       = 10 * time.Minute
	minuteCheckInterval = 1 * time.Minute
	specialUserID       = "638458084653531137" // Reemplaza con el ID real del usuario
	specialUserID1      = "507890132154843146"
	specialUserActive   = false
	specialUserActive1  = false
	specialUserMutex    sync.Mutex
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
		if err := userPoints.Load(); err != nil {
			log.Printf("Error loading points: %v", err)
			s.ChannelMessageSend(m.ChannelID, "⚠️ Error al cargar el ranking. Intenta nuevamente.")
			return
		}

		userPoints.mu.Lock()
		defer userPoints.mu.Unlock()

		// Construir ranking
		var ranking []struct {
			ID    string
			Name  string
			Score float64
		}

		for userID, points := range userPoints.Points {
			member, err := s.GuildMember(m.GuildID, userID)
			name := userID // Default if can't get name
			if err == nil {
				name = member.User.Username
				if member.Nick != "" {
					name = member.Nick
				}
			}

			ranking = append(ranking, struct {
				ID    string
				Name  string
				Score float64
			}{userID, name, points})
		}

		sort.Slice(ranking, func(i, j int) bool {
			return ranking[i].Score > ranking[j].Score
		})

		// Construir mensaje
		var msg strings.Builder
		msg.WriteString("**🏆 RANKING GLOBAL**\n")
		msg.WriteString("```\n")
		msg.WriteString("\n")
		msg.WriteString("  Pos.    Usuario                        Puntos      \n")
		msg.WriteString("\n")
		count := 0
		trophies := []string{"🥇", "🥈", "🥉"}
		for i, user := range ranking {
			if i >= 15 { // Limitar a top 15
				break
			}

			// Formatear nombre para que no rompa la tabla
			displayName := user.Name
			if len(displayName) > 25 {
				displayName = displayName[:22] + "..."
			}

			// Añadir emoji de trofeo a los primeros 3
			trophy := ""
			if i < len(trophies) {
				trophy = trophies[i]
			}

			if count < 3 {
				msg.WriteString(fmt.Sprintf("  %-2d%-2s   %-28s   %9.2f   \n",
					i+1,
					trophy,
					displayName,
					user.Score))
				count++
				if count == 3 {
					msg.WriteString("\n")
				}
			} else {
				msg.WriteString(fmt.Sprintf("  %-2d%-2s    %-28s   %9.2f   \n",
					i+1,
					trophy,
					displayName,
					user.Score))
			}

		}

		msg.WriteString("\n")
		msg.WriteString("```")

		// Enviar mensaje
		_, err := s.ChannelMessageSend(m.ChannelID, msg.String())
		if err != nil {
			log.Printf("Error sending ranking: %v", err)
		}
	}
}

func voiceChannelChecker(s *discordgo.Session) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		pointsToAdd := 5.0 // Valor por defecto

		specialUserMutex.Lock()
		// Prioridad: Lucia > Gabriel > Normal
		if specialUserActive {
			pointsToAdd = 100.0
		} else if specialUserActive1 {
			pointsToAdd = 10.0
		}
		specialUserMutex.Unlock()

		for _, guild := range s.State.Guilds {
			for _, vs := range guild.VoiceStates {
				if !vs.Mute && !vs.SelfMute && !vs.Deaf && !vs.SelfDeaf {
					if added := userPoints.Add(vs.UserID, pointsToAdd); added {
						logName := "normales"
						if pointsToAdd == 100.0 {
							logName = "PREMIUM (Lucia)"
						} else if pointsToAdd == 10.0 {
							logName = "PREMIUM (Gabriel)"
						}
						log.Printf("Bostes %s añadidos a %s (ahora tiene %.2f)\n", logName, vs.UserID, userPoints.Get(vs.UserID))
					}
				}
			}
		}

		if err := userPoints.Save(); err != nil {
			log.Printf("Error al guardar bostes: %v\n", err)
		}
	}
}

func checkSpecialUser(s *discordgo.Session) {
	ticker := time.NewTicker(minuteCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		foundLucia := false
		foundGabriel := false

		// Buscar a los usuarios especiales en todos los canales de voz
		for _, guild := range s.State.Guilds {
			for _, vs := range guild.VoiceStates {
				if vs.UserID == specialUserID && !vs.Mute && !vs.SelfMute && !vs.Deaf && !vs.SelfDeaf {
					foundLucia = true
				} else if vs.UserID == specialUserID1 && !vs.Mute && !vs.SelfMute && !vs.Deaf && !vs.SelfDeaf {
					foundGabriel = true
				}
			}
		}

		specialUserMutex.Lock()
		// Actualizamos los estados independientemente
		if foundLucia != specialUserActive || foundGabriel != specialUserActive1 {
			specialUserActive = foundLucia
			specialUserActive1 = foundGabriel

			if specialUserActive {
				log.Printf("Usuario especial (Lucia) %s detectado en llamada - Activando modo premium (100 puntos)\n", specialUserID)
				s.ChannelMessageSend(channelID, fmt.Sprintf("<@&%s>⚠️ Lucia en llamada ⚠️", notificationRoleID))
			} else if specialUserActive1 {
				log.Printf("Usuario especial (Gabriel) %s detectado en llamada - Activando modo premium (10 puntos)\n", specialUserID1)
				s.ChannelMessageSend(channelID, fmt.Sprintf("<@&%s>⚠️ Gabriel en llamada ⚠️", notificationRoleID))
			} else {
				log.Printf("Ningún usuario especial detectado - Volviendo a modo normal (5 puntos)\n")
			}
		}
		specialUserMutex.Unlock()
	}
}
