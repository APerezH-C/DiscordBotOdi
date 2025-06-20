/*
"objeto1": {
	"Nombre": "objeto11",
	"Precio": 15,
	"Cantidad": 0,
	"Descripcion": "Algo"
},
"objeto2": {
	"Nombre": "objeto22",
	"Precio": 10,
	"Cantidad": 3,
	"Descripcion": "Algo"
}
*/

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
	shop Shop
)

type ShopItem struct {
	Nombre      string `bson:"Nombre"`
	Precio      int    `bson:"Precio"`
	Cantidad    int    `bson:"Cantidad"`
	Descripcion string `bson:"Descripcion"`
	Precio1     string `bson:"Precio1"`
}

type Shop struct {
	Items map[string]ShopItem `bson:"items"`
	mu    sync.Mutex
}

func handleBuyCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Cargar tienda (igual que tu versión original)
	er := shop.Load()
	if er != nil {
		log.Printf("Error cargando tienda: %v", er)
	}

	// Extraer opciones del comando
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		respondInteraction(s, i, "Uso: /bostecompra <nombre-del-objeto>", true)
		return
	}

	itemKey := options[0].StringValue()
	userID := i.Member.User.ID

	shop.mu.Lock()
	item, exists := shop.Items[itemKey]
	shop.mu.Unlock()

	if !exists {
		respondInteraction(s, i, "Ese objeto no existe en la tienda.", true)
		return
	}

	if item.Cantidad <= 0 {
		respondInteraction(s, i, "Este objeto está agotado.", true)
		return
	}

	// Verificar saldo (misma lógica)
	userBalance := userPoints.Get(userID)
	if userBalance < float64(item.Precio) {
		respondInteraction(s, i,
			fmt.Sprintf("Saldo insuficiente. Necesitas %d bostes y tienes %.2f",
				item.Precio, userBalance), true)
		return
	}

	// Procesar compra
	success := userPoints.Add(userID, -float64(item.Precio))
	if !success {
		respondInteraction(s, i, "Error al procesar la compra", true)
		return
	}

	// Actualizar tienda e inventario
	shop.mu.Lock()
	item.Cantidad--
	shop.Items[itemKey] = item
	shop.mu.Unlock()

	inventory.AddItem(userID, item.Nombre)

	// Guardar cambios (igual que tu versión)
	if err := userPoints.Save(); err != nil {
		log.Printf("Error guardando bostes: %v", err)
		respondInteraction(s, i, "Error al guardar los puntos. Contacta con un admin.", true)
		return
	}

	if err := shop.Save(); err != nil {
		log.Printf("Error guardando tienda: %v", err)
	}

	if err := inventory.Save(); err != nil {
		log.Printf("Error guardando inventario: %v", err)
	}

	// Notificar compra exitosa
	respondInteraction(s, i,
		fmt.Sprintf("✅ Compra exitosa! Has adquirido **%s** por %d bostes. Tu nuevo saldo: %.2f",
			item.Nombre, item.Precio, userPoints.Get(userID)), true)

	// Notificación especial (igual que tu versión)
	nickname := i.Member.Nick
	if nickname == "" {
		nickname = i.Member.User.Username
	}

	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Content: fmt.Sprintf("<@&%s>⚠️ %s compró **%s** ⚠️", notificationRoleID, nickname, item.Nombre),
	})
	if err != nil {
		return
	}
}

func handleShopCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Cargar tienda (igual que tu versión original)
	er := shop.Load()
	if er != nil {
		log.Printf("Error cargando tienda: %v", er)
	}

	shop.mu.Lock()
	defer shop.mu.Unlock()

	// Verificar si la tienda está vacía (igual que tu versión)
	if len(shop.Items) == 0 {
		respondInteraction(s, i, "La tienda está vacía.", true)
		return
	}

	// Crear embed (idéntico a tu versión)
	embed := &discordgo.MessageEmbed{
		Title:       "🏪 Tienda",
		Description: "Aquí tienes los artículos disponibles:",
		Color:       0xf3cfb2, // Mismo color que usabas
		Fields:      []*discordgo.MessageEmbedField{},
	}

	// Añadir items (misma lógica)
	for key, item := range shop.Items {
		field := &discordgo.MessageEmbedField{
			Name: fmt.Sprintf("__%s__ (%s)", item.Nombre, key),
			Value: fmt.Sprintf("💰 Precio: %s bostes\n📦 Cantidad disponible: %d\n📝 Descripción: %s",
				item.Precio1, item.Cantidad, item.Descripcion),
			Inline: false,
		}
		embed.Fields = append(embed.Fields, field)
	}

	// Responder con el embed usando tu método respondInteraction
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
	if err != nil {
		log.Printf("Error enviando embed: %v", err)
		// Fallback a mensaje simple si falla el embed
		respondInteraction(s, i, "Error al mostrar la tienda. Intenta nuevamente.", true)
	}
}

func (s *Shop) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("shop")

	var result Shop
	err := collection.FindOne(ctx, bson.M{}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			s.Items = make(map[string]ShopItem)
			return nil
		}
		return fmt.Errorf("error al cargar tienda: %w", err)
	}

	s.Items = result.Items
	return nil
}

func (s *Shop) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("shop")

	_, err := collection.UpdateOne(
		ctx,
		bson.M{}, // filtro vacío para documento único
		bson.M{"$set": bson.M{"items": s.Items}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("error al guardar tienda: %w", err)
	}

	return nil
}
