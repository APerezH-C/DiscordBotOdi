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
	inventory Inventory
)

type Inventory struct {
	Users map[string][]string `bson:"users"`
	mu    sync.Mutex
}

func handleInventoryCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Cargar inventario (igual que tu versión original)
	er := inventory.Load()
	if er != nil {
		log.Printf("Error cargando inventario: %v", er)
	}

	userID := i.Member.User.ID

	inventory.mu.Lock()
	defer inventory.mu.Unlock()

	userItems, exists := inventory.Users[userID]
	if !exists || len(userItems) == 0 {
		respondInteraction(s, i, "No tienes ningún bosteObjeto en tu inventario.", true)
		return
	}

	// Construir respuesta (misma lógica que tu versión)
	var response strings.Builder
	response.WriteString(fmt.Sprintf("**🎒 Inventario de %s**\n\n", i.Member.User.Username))

	itemCounts := make(map[string]int)
	for _, item := range userItems {
		itemCounts[item]++
	}

	for item, count := range itemCounts {
		response.WriteString(fmt.Sprintf("- %s (x%d)\n", item, count))
	}

	// Responder usando tu método respondInteraction
	respondInteraction(s, i, response.String(), false)
}

func (i *Inventory) Load() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("inventory")
	var result Inventory

	err := collection.FindOne(ctx, bson.M{}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			i.Users = make(map[string][]string)
			return nil
		}
		return fmt.Errorf("error al cargar inventario: %w", err)
	}

	i.Users = result.Users
	return nil
}

func (i *Inventory) Save() error {
	i.mu.Lock()
	defer i.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("inventory")

	_, err := collection.UpdateOne(
		ctx,
		bson.M{},
		bson.M{"$set": bson.M{"users": i.Users}},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("error al guardar inventario: %w", err)
	}

	return nil
}

func (i *Inventory) AddItem(userID, itemName string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.Users == nil {
		i.Users = make(map[string][]string)
	}

	i.Users[userID] = append(i.Users[userID], itemName)
}
