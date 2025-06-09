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
	"time"
)

func handleStatsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {

	userID := i.Member.User.ID
	username := i.Member.User.Username
	avatarURL := i.Member.User.AvatarURL("")

	er := userStats.load()
	if er != nil {
		log.Printf("Error cargando estadísticas: %v", er)
	}

	userStats.mu.Lock()
	stats, exists := userStats.Stats[userID]
	userStats.mu.Unlock()

	if !exists {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "📊 No tienes estadísticas registradas aún.",
			},
		})
		return
	}

	profit := stats.TotalGanado - stats.TotalApostado
	winRate := 0.0
	if stats.ApuestasTotales > 0 {
		winRate = (stats.TotalGanado / stats.TotalApostado) * 100
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 Estadísticas de %s", username),
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Apuestas Totales",
				Value: fmt.Sprintf("%d", stats.ApuestasTotales),
			},
			{
				Name:   "Total Apostado",
				Value:  fmt.Sprintf("%.2f", stats.TotalApostado),
				Inline: true,
			},
			{
				Name:   "Total Ganado",
				Value:  fmt.Sprintf("%.2f", stats.TotalGanado),
				Inline: true,
			},
			{
				Name:  "Profit Neto",
				Value: fmt.Sprintf("%.2f", profit),
			},
			{
				Name:  "Porcentaje de Retorno",
				Value: fmt.Sprintf("%.2f%%", winRate),
			},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: avatarURL,
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (us *UserStats) load() error {
	us.mu.Lock()
	defer us.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("dice_stats")
	var result struct {
		Stats map[string]UserStat `bson:"stats"`
	}

	err := collection.FindOne(ctx, bson.M{}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			us.Stats = make(map[string]UserStat)
			return nil
		}
		return fmt.Errorf("error al cargar estadísticas: %w", err)
	}

	us.Stats = result.Stats
	return nil
}

// Guardar en MongoDB
func (us *UserStats) save() error {
	us.mu.Lock()
	defer us.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("dice_stats")
	_, err := collection.UpdateOne(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{"stats": us.Stats}},
		options.Update().SetUpsert(true),
	)

	if err != nil {
		return fmt.Errorf("error al guardar estadísticas: %w", err)
	}
	return nil
}

func (us *UserStats) get(userID string) (UserStat, bool) {
	us.mu.Lock()
	defer us.mu.Unlock()

	stat, exists := us.Stats[userID]
	return stat, exists
}
