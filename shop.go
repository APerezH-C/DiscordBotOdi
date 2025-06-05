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
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
)

var (
	shop Shop
)

type ShopItem struct {
	Nombre      string `json:"Nombre"`
	Precio      int    `json:"Precio"`
	Cantidad    int    `json:"Cantidad"`
	Descripcion string `json:"Descripcion"`
}

type Shop struct {
	Items map[string]ShopItem `json:"items"`
	mu    sync.Mutex
}

func handleBuyCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 1 {
		s.ChannelMessageSend(m.ChannelID, "Uso: !bosteCompra <nombre-del-objeto>")
		return
	}

	itemKey := args[0]
	userID := m.Author.ID

	// Verificar objeto
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
	if userPoints.Get(m.Author.ID) < float64(item.Precio) {
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("Saldo insuficiente. Necesitas %.2f bostes y tienes %.2f",
				float64(item.Precio), userPoints.Get(m.Author.ID)))
		return
	}

	// Restar puntos (nota: item.Precio sigue siendo int en ShopItem)
	success := userPoints.Add(m.Author.ID, -float64(item.Precio))
	if !success {
		s.ChannelMessageSend(m.ChannelID, "Error al procesar la compra")
		return
	}

	// Actualizar tienda
	shop.mu.Lock()
	item.Cantidad--
	shop.Items[itemKey] = item
	shop.mu.Unlock()

	// Actualizar inventario
	inventory.AddItem(userID, item.Nombre)

	// Guardar cambios
	if err := userPoints.Save("points.json"); err != nil {
		log.Printf("Error guardando bostes: %v", err)
		s.ChannelMessageSend(m.ChannelID, "Error al guardar los puntos. Contacta con un admin.")
		return
	}

	if err := shop.Save("shop.json"); err != nil {
		log.Printf("Error guardando tienda: %v", err)
	}

	if err := inventory.Save("inventario.json"); err != nil {
		log.Printf("Error guardando inventario: %v", err)
	}

	// Mensaje de confirmación con nuevo balance
	s.ChannelMessageSend(m.ChannelID,
		fmt.Sprintf("✅ Compra exitosa! Has adquirido **%s** por %d bostes. Tu nuevo saldo: %.2f",
			item.Nombre, item.Precio, userPoints.Get(userID)))
}

func handleShopCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
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

func (s *Shop) Load(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			s.Items = make(map[string]ShopItem)
			return nil
		}
		return err
	}

	return json.Unmarshal(data, s)
}

func (s *Shop) Save(filename string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, data, 0644)
}
