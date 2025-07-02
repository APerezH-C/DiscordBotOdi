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
	checkInterval       = 15 * time.Minute
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

func handlePuntosCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Member == nil || i.Member.User == nil {
		return
	}

	if err := userPoints.Load(); err != nil {
		log.Printf("Error cargando puntos: %v", err)
	}

	puntos := userPoints.Get(i.Member.User.ID)

	respondInteraction(s, i, fmt.Sprintf("<@%s> tienes %.2f bostes.", i.Member.User.ID, puntos), false)

}

func handleRankingCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {

	// Cargar puntos
	if err := userPoints.Load(); err != nil {
		log.Printf("Error loading points: %v", err)
		respondInteraction(s, i, "⚠️ Error al cargar el ranking. Intenta nuevamente.", false)
		return
	}

	userPoints.mu.Lock()
	defer userPoints.mu.Unlock()

	// Estructura para el ranking (igual que en tu versión)
	var ranking []struct {
		ID    string
		Name  string
		Score float64
	}

	// Construir ranking
	for userID, points := range userPoints.Points {
		member, err := s.GuildMember(i.GuildID, userID)
		name := userID // Default
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

	// Ordenar (igual que tu versión)
	sort.Slice(ranking, func(i, j int) bool {
		return ranking[i].Score > ranking[j].Score
	})

	// Construir mensaje (igual que tu formato original)
	var msg strings.Builder
	msg.WriteString("**🏆 RANKING GLOBAL**\n")
	msg.WriteString("```\n")
	msg.WriteString("\n")
	msg.WriteString("  Pos.    Usuario                        Puntos      \n")
	msg.WriteString("\n")

	count := 0
	trophies := []string{"🥇", "🥈", "🥉"}
	for i, user := range ranking {
		if i >= 15 {
			break
		}

		displayName := user.Name
		if len(displayName) > 25 {
			displayName = displayName[:22] + "..."
		}

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

	// Responder usando la interacción
	respondInteraction(s, i, msg.String(), false)

}

func voiceChannelChecker(s *discordgo.Session) {
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for range ticker.C {
		pointsToAdd := 75.0 // Valor por defecto

		specialUserMutex.Lock()
		// Prioridad: Lucia > Gabriel > Normal
		if specialUserActive {
			pointsToAdd = 500.0
		} else if specialUserActive1 {
			pointsToAdd = 100.0
		}
		specialUserMutex.Unlock()

		for _, guild := range s.State.Guilds {
			for _, vs := range guild.VoiceStates {
				if !vs.Mute && !vs.SelfMute && !vs.Deaf && !vs.SelfDeaf {
					if added := userPoints.Add(vs.UserID, pointsToAdd); added {
						logName := "normales"
						if pointsToAdd == 500.0 {
							logName = "PREMIUM (Lucia)"
						} else if pointsToAdd == 100.0 {
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
				log.Printf("Usuario especial (Lucia) %s detectado en llamada - Activando modo premium (500 puntos)\n", specialUserID)
				s.ChannelMessageSend(channelID, fmt.Sprintf("<@&%s>⚠️ Lucia en llamada ⚠️ +500 bostes", notificationRoleID))
			} else if specialUserActive1 {
				log.Printf("Usuario especial (Gabriel) %s detectado en llamada - Activando modo premium (60 puntos)\n", specialUserID1)
				s.ChannelMessageSend(channelID, fmt.Sprintf("<@&%s>⚠️ Gabriel en llamada ⚠️ +100 bostes", notificationRoleID))
			} else {
				log.Printf("Ningún usuario especial detectado - Volviendo a modo normal (75 puntos)\n")
			}
		}
		specialUserMutex.Unlock()
	}
}
