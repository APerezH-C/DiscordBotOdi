package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var (
	matchID string

	userPoints UserPoints

	token = "Bot MTM3OTQ1OTMzOTM3ODc1NzY4Mg.G3KBK1.XhyuO1RIGksFrRteHWzAic4y6PWdMu3tL03Sg8"
)

const statsFile = "dice_stats.json"

func main() {
	// Crear sesión de Discord
	dg, err := discordgo.New(token)
	if err != nil {
		log.Fatal("Error al crear sesión de Discord:", err)
	}

	userPoints.Load("points.json")
	shop.Load("shop.json")
	inventory.Load("inventario.json")

	err = loadStats()
	if err != nil {
		log.Printf("Error cargando estadísticas: %v", err)
	}

	// Registrar handlers
	dg.AddHandler(ruleta)
	dg.AddHandler(readyHandler)
	dg.AddHandler(riot)
	dg.AddHandler(messageCreate)
	dg.AddHandler(getId)

	// Abrir conexión
	err = dg.Open()
	if err != nil {
		log.Fatal("Error al abrir conexión:", err)
	}
	defer dg.Close()

	// Iniciar el checker de voz en segundo plano
	go voiceChannelChecker(dg)
	go watchForGame(dg)

	fmt.Println("Bot listo. Presiona CTRL+C para salir.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

// Handlers
func readyHandler(s *discordgo.Session, event *discordgo.Ready) {
	fmt.Println("Bot conectado como", event.User.Username)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!bosteTienda") {
		handleShopCommand(s, m)
	} else if strings.HasPrefix(m.Content, "!bosteCompra") {
		args := strings.Fields(m.Content)[1:]
		handleBuyCommand(s, m, args)
	} else if strings.HasPrefix(m.Content, "!bosteInventario") {
		handleInventoryCommand(s, m)
	} else if strings.HasPrefix(m.Content, "!bosteHelp") {
		handleHelpCommand(s, m)
	} else if strings.HasPrefix(m.Content, "!bosteStats") {
		handleStatsCommand(s, m)
	} else if strings.HasPrefix(m.Content, "!bosteDice") {
		args := strings.Fields(m.Content)[1:]
		handleDiceCommand(s, m, args)
	} else if strings.HasPrefix(m.Content, "!verify") {
		args := strings.Fields(m.Content)[1:]
		handleVerifyCommand(s, m, args)
	} else if strings.HasPrefix(m.Content, "!bosteSeed") {
		handleNewSeedCommand(s, m)
	}
}
