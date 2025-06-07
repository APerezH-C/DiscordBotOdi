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
	"strings"
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
}

type Shop struct {
	Items map[string]ShopItem `bson:"items"`
	mu    sync.Mutex
}

func handleBuyCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	er := shop.Load()
	if er != nil {
		log.Printf("Error cargando tienda: %v", er)
	}

	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "Uso: !bosteCompra <nombre-del-objeto>")
		return
	}

	itemKey := args[0]
	userID := m.Author.ID

	shop.mu.Lock()
	item, exists := shop.Items[itemKey]
	shop.mu.Unlock()

	if !exists {
		s.ChannelMessageSend(m.ChannelID, "Ese objeto no existe en la tienda.")
		return
	}

	if item.Cantidad <= 0 {
		s.ChannelMessageSend(m.ChannelID, "Este objeto está agotado.")
		return
	}

	// Verificar y restar puntos
	if userPoints.Get(userID) < float64(item.Precio) {
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("Saldo insuficiente. Necesitas %d bostes y tienes %.2f",
				item.Precio, userPoints.Get(userID)))
		return
	}

	success := userPoints.Add(userID, -float64(item.Precio))
	if !success {
		s.ChannelMessageSend(m.ChannelID, "Error al procesar la compra")
		return
	}

	shop.mu.Lock()
	item.Cantidad--
	shop.Items[itemKey] = item
	shop.mu.Unlock()

	// Actualizar inventario (ya debe estar adaptado a MongoDB)
	inventory.AddItem(userID, item.Nombre)

	// Guardar cambios en puntos, tienda e inventario
	if err := userPoints.Save(); err != nil {
		log.Printf("Error guardando bostes: %v", err)
		s.ChannelMessageSend(m.ChannelID, "Error al guardar los puntos. Contacta con un admin.")
		return
	}

	if err := shop.Save(); err != nil {
		log.Printf("Error guardando tienda: %v", err)
	}

	if err := inventory.Save(); err != nil {
		log.Printf("Error guardando inventario: %v", err)
	}

	s.ChannelMessageSend(m.ChannelID,
		fmt.Sprintf("✅ Compra exitosa! Has adquirido **%s** por %d bostes. Tu nuevo saldo: %.2f",
			item.Nombre, item.Precio, userPoints.Get(userID)))
}

func handleShopCommand(s *discordgo.Session, m *discordgo.MessageCreate) {

	er := shop.Load()
	if er != nil {
		log.Printf("Error cargando tienda: %v", er)
	}

	shop.mu.Lock()
	defer shop.mu.Unlock()

	if len(shop.Items) == 0 {
		s.ChannelMessageSend(m.ChannelID, "La tienda está vacía.")
		return
	}

	var response strings.Builder
	response.WriteString("**🏪 Tienda**\n\n")

	for key, item := range shop.Items {
		response.WriteString(fmt.Sprintf("**%s** (%s)\n", item.Nombre, key))
		response.WriteString(fmt.Sprintf("💰 Precio: %d bostes\n", item.Precio))
		response.WriteString(fmt.Sprintf("📦 Cantidad disponible: %d\n", item.Cantidad))
		response.WriteString(fmt.Sprintf("📝 Descripción: %s\n\n", item.Descripcion))
	}

	s.ChannelMessageSend(m.ChannelID, response.String())
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
