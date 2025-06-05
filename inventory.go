package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

var (
	inventory Inventory
)

type PurchasedItem struct {
	ProductosCanjeados []string `json:"productos_Canjeados"`
}

type Inventory struct {
	Users map[string]PurchasedItem `json:"users"`
	mu    sync.Mutex
}

func handleInventoryCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	userID := m.Author.ID

	inventory.mu.Lock()
	defer inventory.mu.Unlock()

	userInventory, exists := inventory.Users[userID]
	if !exists || len(userInventory.ProductosCanjeados) == 0 {
		s.ChannelMessageSend(m.ChannelID, "No tienes ningún bosteObjeto en tu inventario.")
		return
	}

	var response strings.Builder
	response.WriteString(fmt.Sprintf("**🎒 Inventario de %s**\n\n", m.Author.Username))

	// Contar objetos duplicados
	itemCounts := make(map[string]int)
	for _, item := range userInventory.ProductosCanjeados {
		itemCounts[item]++
	}

	// Mostrar objetos y cantidades
	for item, count := range itemCounts {
		response.WriteString(fmt.Sprintf("- %s (x%d)\n", item, count))
	}

	s.ChannelMessageSend(m.ChannelID, response.String())
}

func (i *Inventory) Load(filename string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			i.Users = make(map[string]PurchasedItem)
			return nil
		}
		return err
	}

	return json.Unmarshal(data, i)
}

func (i *Inventory) Save(filename string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	data, err := json.MarshalIndent(i, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}

func (i *Inventory) AddItem(userID, itemName string) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.Users == nil {
		i.Users = make(map[string]PurchasedItem)
	}

	if _, exists := i.Users[userID]; !exists {
		i.Users[userID] = PurchasedItem{
			ProductosCanjeados: []string{},
		}
	}

	userItems := i.Users[userID]
	userItems.ProductosCanjeados = append(userItems.ProductosCanjeados, itemName)
	i.Users[userID] = userItems
}
