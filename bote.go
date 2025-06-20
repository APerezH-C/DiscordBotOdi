package main

import (
	"RuletaRusaOdi/database"
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

type Bote struct {
	Bostebote  float64   `bson:"bostebote"`
	LastWinner string    `bson:"lastWinner"`
	LastAmount float64   `bson:"lastAmountWinner"`
	LastDate   time.Time `bson:"lastDate"`
}

type BoteWrapper struct {
	Bote Bote `bson:"bote"`
}

func AddToBote(amount float64) error {
	collection := database.GetCollection("bote")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Usa $inc dentro de $set para mantener la estructura anidada
	_, err := collection.UpdateOne(
		ctx,
		bson.M{},
		bson.M{
			"$inc": bson.M{"bote.bostebote": amount},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func GetFullBote() (*Bote, error) {
	collection := database.GetCollection("bote")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result BoteWrapper
	err := collection.FindOne(ctx, bson.M{}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Crear documento con estructura anidada
			newBote := Bote{
				Bostebote:  0,
				LastWinner: "",
				LastAmount: 0,
				LastDate:   time.Time{},
			}
			_, err = collection.InsertOne(ctx, bson.M{"bote": newBote})
			return &newBote, err
		}
		return nil, err
	}
	return &result.Bote, nil
}

func UpdateBoteAfterTransfer(s *discordgo.Session, guildID string, userID string, amount float64) error {

	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return fmt.Errorf("error obteniendo miembro: %v", err)
	}

	// Usar el nickname si existe, sino el username
	displayName := member.User.Username
	if member.Nick != "" {
		displayName = member.Nick
	}

	collection := database.GetCollection("bote")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = collection.UpdateOne(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{
			"bote.bostebote":        0,
			"bote.lastWinner":       displayName,
			"bote.lastAmountWinner": amount,
			"bote.lastDate":         time.Now(),
		}},
		options.Update().SetUpsert(true),
	)
	return err
}

func ResetBote() error {
	collection := database.GetCollection("bote")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.UpdateOne(
		ctx,
		bson.M{}, // Filtro vacío para actualizar el primer documento
		bson.M{
			"$set": bson.M{
				"bote.bostebote":        0, // Asegura que se actualice dentro del objeto "bote"
				"bote.lastWinner":       "",
				"bote.lastAmountWinner": 0,
				"bote.lastDate":         time.Time{}, // Fecha cero o null
			},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func handleBoteCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bote, err := GetFullBote()
	if err != nil {
		respondInteraction(s, i, "❌ Error al consultar el bote", true)
		return
	}

	embed := &discordgo.MessageEmbed{
		Title: "💰 Bote Acumulado",
		Color: 0xFFD700, // Color dorado
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Total",
				Value: fmt.Sprintf("```%.2f bostes```", bote.Bostebote),
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    "El bote aumenta con el 1% de apuestas ganadas y 25% de las perdidas",
			IconURL: i.Member.User.AvatarURL(""),
		},
	}

	if bote.LastWinner != "" {
		member, err := s.GuildMember(i.GuildID, bote.LastWinner)
		winnerName := bote.LastWinner
		if err == nil {
			if member.Nick != "" {
				winnerName = member.Nick
			} else {
				winnerName = member.User.Username
			}
		}

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Último ganador",
			Value: fmt.Sprintf("👑 %s\n💵 %.2f bostes\n📅 %s",
				winnerName,
				bote.LastAmount,
				bote.LastDate.Format("02/01/2006 15:04")),
			Inline: true,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleResetBoteCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Verificar permisos de administrador
	userID := i.Member.User.ID

	if userID != "431796013934837761" {
		respondInteraction(s, i, "❌ Solo los administradores pueden usar este comando", true)
		return
	}

	err := ResetBote()
	if err != nil {
		respondInteraction(s, i, "❌ Error al resetear el bote", true)
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔄 Bote Reiniciado",
		Color:       0x00FF00, // Verde
		Description: "El bote ha sido reiniciado a 0 bostes",
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Reset por: %s", i.Member.User.Username),
			IconURL: i.Member.User.AvatarURL(""),
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func handleGiveBoteCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Verificar permisos de administrador
	userID := i.Member.User.ID

	if userID != "431796013934837761" {
		respondInteraction(s, i, "❌ Solo los administradores pueden usar este comando", true)
		return
	}

	// Obtener el bote actual
	currentBote, err := GetFullBote() // Nueva función que veremos abajo
	if err != nil {
		respondInteraction(s, i, "❌ Error al consultar el bote", true)
		return
	}

	if currentBote.Bostebote <= 0 {
		respondInteraction(s, i, "❌ El bote está vacío", true)
		return
	}

	// Obtener usuario objetivo
	targetUser := i.ApplicationCommandData().Options[0].UserValue(s)

	// Transferir el bote y actualizar historial
	userPoints.Add(targetUser.ID, currentBote.Bostebote)

	// Actualizar el bote con los nuevos campos
	err = UpdateBoteAfterTransfer(s, i.GuildID, targetUser.ID, currentBote.Bostebote)
	if err != nil {
		respondInteraction(s, i, "❌ Error al actualizar el bote", true)
		return
	}

	// Guardar cambios
	err = userPoints.Save()
	if err != nil {
		respondInteraction(s, i, "❌ Error al guardar los puntos", true)
		return
	}

	// Crear embed de respuesta
	embed := &discordgo.MessageEmbed{
		Title: "🎉 Bote Entregado",
		Color: 0x00FF00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Usuario",
				Value: fmt.Sprintf("<@%s>", targetUser.ID),
			},
			{
				Name:  "Cantidad",
				Value: fmt.Sprintf("```%.2f bostes```", currentBote.Bostebote),
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Transferido por: %s", i.Member.User.Username),
			IconURL: i.Member.User.AvatarURL(""),
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}
