package database

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	dbName = "DiscordBotOdi"
)

var (
	client *mongo.Client
)

// Connect establece la conexión con MongoDB (llamar esto al inicio del bot)
func Connect(mongoURI string) error {
	clientOptions := options.Client().ApplyURI(mongoURI)

	var err error
	client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return fmt.Errorf("error al conectar con MongoDB: %v", err)
	}

	// Verificar conexión
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("error al verificar conexión: %v", err)
	}

	log.Println("✅ Conexión a MongoDB establecida")
	return nil
}

// GetCollection devuelve una colección para usar en otros archivos
func GetCollection(collectionName string) *mongo.Collection {
	return client.Database(dbName).Collection(collectionName)
}

// Close cierra la conexión (llamar al cerrar el bot)
func Close() error {
	if client != nil {
		return client.Disconnect(context.Background())
	}
	return nil
}
