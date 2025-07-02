package main

import (
	"RuletaRusaOdi/database"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
)

var (
	matchID string

	userPoints = &UserPoints{}
	userStats  = &UserStats{}

	token    string
	mongoURI string

	commandsRegistered bool = false
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error cargando .env")
	}
	token = os.Getenv("DISCORD_TOKEN")
	mongoURI = os.Getenv("MONGO_URI")
	riotApiKey = os.Getenv("RIOT_API_KEY")

	log.Println("Conectando a MongoDB...")
	err = database.Connect(mongoURI)
	if err != nil {
		LogError("Error conectandose a MongoDB: %v", err)
	}
	defer database.Close()

	log.Println("Creando sesión de Discord...")
	// Crear sesión de Discord
	dg, err := discordgo.New(token)
	if err != nil {
		LogError("Error al crear sesión de Discord: %v", err)
	}
	log.Println("Sesión de Discord creada")

	initDiscordAndStats(dg)

	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if i.Type == discordgo.InteractionApplicationCommand {
			switch i.ApplicationCommandData().Name {
			case "bostes":
				handlePuntosCommands(s, i)
			case "ranking":
				handleRankingCommands(s, i)
			case "cargar", "disparar", "terminar":
				handleRuletaCommands(s, i)
			case "bostedice":
				handleDiceCommand(s, i)
			case "revertirapuesta", "apuesta":
				handleBetCommands(s, i)
			case "bostetienda":
				handleShopCommand(s, i)
			case "bostecompra":
				handleBuyCommand(s, i)
			case "bosteinventario":
				handleInventoryCommand(s, i)
			case "bostehelp":
				handleHelpCommand(s, i)
			case "bostestats":
				handleStatsCommand(s, i)
			case "notificaciones":
				if len(i.ApplicationCommandData().Options) == 0 {
					showNotificationStatus(s, i)
				} else {
					switch i.ApplicationCommandData().Options[0].Name {
					case "on":
						handleNotificationSubscribe(s, i)
					case "off":
						handleNotificationUnsubscribe(s, i)
					}
				}
			case "bosteseed":
				handleNewSeedCommand(s, i)
			case "verify":
				handleVerifyCommand(s, i)
			case "quienes":
				handleWhoIsCommand(s, i)
			case "bote":
				handleBoteCommand(s, i)
			case "resetbote":
				handleResetBoteCommand(s, i)
			case "givebote":
				handleGiveBoteCommand(s, i)
			}
		}
	})

	log.Println("Abriendo conexión a Discord...")
	// Abrir conexión
	err = dg.Open()
	if err != nil {
		LogError("Error al abrir conexión: %v", err)

	}
	log.Println("Conexión a Discord abierta correctamente")
	defer dg.Close()

	// Iniciar el checker de voz en segundo plano
	go voiceChannelChecker(dg)
	go checkSpecialUser(dg)
	go watchForGame(dg)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc
}

func initDiscordAndStats(dg *discordgo.Session) {

	log.Println("Registrando handlers de comandos y eventos...")
	dg.AddHandler(readyHandler)

	log.Println("Cargando puntos de usuario...")
	err := userPoints.Load()
	if err != nil {
		LogError("Error cargando puntos de usuario: %v", err)
	}

	log.Println("Cargando tienda...")
	err = shop.Load()
	if err != nil {
		LogError("Error cargando tienda: %v", err)
	}

	log.Println("Cargando inventarios de usuario...")
	err = inventory.Load()
	if err != nil {
		LogError("Error cargando inventario: %v", err)
	}

	log.Println("Cargando stats de usuario...")
	err = userStats.load()
	if err != nil {
		LogError("Error cargando estadísticas: %v", err)
	}
}

func readyHandler(s *discordgo.Session, event *discordgo.Ready) {
	log.Println("Evento 'ready' recibido")
	log.Printf("Bot conectado como %s#%s", event.User.Username, event.User.Discriminator)
	if !commandsRegistered {
		LogInfo("─────────────────────────────────────────────────────────────────")
		log.Printf("\n")
		LogInfo("------------------Registrando comandos slash...------------------")
		commandsRegistered = true
		registerSlashCommands(s)
	}
}

func cleanAllCommands(s *discordgo.Session) {
	// Limpiar comandos globales
	if cmds, err := s.ApplicationCommands(s.State.User.ID, ""); err == nil {
		for _, cmd := range cmds {
			s.ApplicationCommandDelete(s.State.User.ID, "", cmd.ID)
			time.Sleep(500 * time.Millisecond) // Evitar rate limits
		}
	}

	// Limpiar en servidores específicos
	guilds := []string{"518873978572374066"} // Añade todos tus server IDs
	for _, guildID := range guilds {
		if cmds, err := s.ApplicationCommands(s.State.User.ID, guildID); err == nil {
			for _, cmd := range cmds {
				s.ApplicationCommandDelete(s.State.User.ID, guildID, cmd.ID)
				time.Sleep(200 * time.Millisecond)
			}
		}
	}
}

func registerSlashCommands(s *discordgo.Session) {
	guildID := "518873978572374066" // Tu server ID

	// 1. Verificar conexión
	if s.State.User == nil {
		LogError("Error: Bot no está listo")
		return
	}

	// 2. Limpiar comandos existentes con retry
	var existingCommands []*discordgo.ApplicationCommand
	var err error

	LogInfo("------------------Eliminando comandos antiguos------------------")
	for i := 0; i < 3; i++ { // 3 intentos
		existingCommands, err = s.ApplicationCommands(s.State.User.ID, guildID)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}

	for _, cmd := range existingCommands {
		log.Printf("Eliminando comando: %s (%s)", cmd.Name, cmd.ID)
		err := s.ApplicationCommandDelete(s.State.User.ID, guildID, cmd.ID)
		if err != nil {
			LogError("Error eliminando %s: %v", cmd.Name, err)
		}
	}
	LogInfo("------------------Comandos eliminados------------------")

	// 3. Registrar nuevos comandos
	newCommands := []*discordgo.ApplicationCommand{
		{
			Name:        "bostes",
			Description: "Consulta cuántos bostes tienes.",
		},
		{
			Name:        "ranking",
			Description: "Muestra el ranking global de puntos.",
		},
		{
			Name:        "cargar",
			Description: "Inicia un juego de ruleta rusa con N balas",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "balas",
					Description: "Número de balas (1-9)",
					Required:    true,
					MinValue:    func() *float64 { v := 1.0; return &v }(),
					MaxValue:    9,
				},
			},
		},
		{
			Name:        "disparar",
			Description: "Dispara la ruleta rusa",
		},
		{
			Name:        "terminar",
			Description: "Termina el juego actual",
		},
		{
			Name:        "bostedice",
			Description: "Juega a los dados (under/over)",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "tipo",
					Description: "Tipo de apuesta (under/over)",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "Under",
							Value: "under",
						},
						{
							Name:  "Over",
							Value: "over",
						},
					},
				},
				{
					Type: discordgo.ApplicationCommandOptionString,
					Name: "numero",
					Description: fmt.Sprintf("Número objetivo (under: %.2f-%.2f | over: %.2f-%.2f)",
						underminNumber, undermaxNumber, overminNumber, overmaxNumber),
					Required: true,
					MinValue: func() *float64 { v := underminNumber; return &v }(), // Asegúrate de definir estas variables
					MaxValue: overmaxNumber,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "cantidad",
					Description: "Cantidad a apostar",
					Required:    true,
					MinValue:    func() *float64 { v := 1.0; return &v }(),
				},
			},
		},
		{
			Name:        "apuesta",
			Description: "Realizar una apuesta en la partida actual",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "tipo",
					Description: "Tipo de apuesta (win/lose)",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "Win",
							Value: "win",
						},
						{
							Name:  "Lose",
							Value: "lose",
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionNumber,
					Name:        "cantidad",
					Description: "Cantidad a apostar",
					Required:    true,
					MinValue:    func() *float64 { v := 0.01; return &v }(),
				},
			},
		},
		{
			Name:        "revertirapuesta",
			Description: "[ADMIN] Revertir todas las apuestas",
		},
		{
			Name:        "bostetienda",
			Description: "Muestra los artículos disponibles en la tienda",
		},
		{
			Name:        "bostecompra",
			Description: "Compra un objeto de la tienda",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "objeto",
					Description: "Nombre del objeto a comprar",
					Required:    true,
					// Opcional: añadir choices con los objetos disponibles
					// Choices: []*discordgo.ApplicationCommandOptionChoice{...}
				},
			},
		},
		{
			Name:        "bosteinventario",
			Description: "Muestra los objetos que tienes en tu inventario",
		},
		{
			Name:        "bostehelp",
			Description: "Muestra todos los comandos disponibles del bot",
		},
		{
			Name:        "bostestats",
			Description: "Muestra tus estadísticas personales de apuestas",
		},
		{
			Name:        "notificaciones",
			Description: "Gestiona tus notificaciones",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "on",
					Description: "Activa las notificaciones importantes",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "off",
					Description: "Desactiva las notificaciones importantes",
				},
			},
		},
		{
			Name:        "bosteseed",
			Description: "Regenera la semilla del sistema (solo admin)",
		},
		{
			Name:        "verify",
			Description: "Verifica un resultado usando la semilla del servidor, cliente y nonce",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "server_seed",
					Description: "Semilla del servidor",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "client_seed",
					Description: "Semilla del cliente",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "nonce",
					Description: "Número de nonce",
					Required:    true,
				},
			},
		},
		{
			Name:        "quienes",
			Description: "Muestra información de un usuario por su ID",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "user_id",
					Description: "ID del usuario a consultar",
					Required:    true,
				},
			},
		},
		{
			Name:        "bote",
			Description: "Muestra la cantidad acumulada en el bote",
		},
		{
			Name:        "resetbote",
			Description: "[ADMIN] Reinicia el bote a 0",
		},
		{
			Name:        "givebote",
			Description: "[ADMIN] Entrega todo el bote a un usuario",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "usuario",
					Description: "Usuario que recibirá el bote",
					Required:    true,
				},
			},
		},
	}

	time.Sleep(1 * time.Second) // Espera para evitar rate limits

	LogInfo("------------------Registrando nuevos comandos---------------------")
	for _, cmd := range newCommands {
		log.Printf("Registrando comando: %s", cmd.Name)
		_, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, cmd)
		if err != nil {
			LogError("Fallo al registrar %s: %v", cmd.Name, err)
		}
	}
	LogInfo("------------------Comandos registrados--------------------------")
	log.Printf("\n")
	LogInfo("──────────────────────────INICIALIZACION FINALIZADA──────────────────────────────────")

}

func respondInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string, user bool) {
	if user {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
	} else {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: content,
			},
		})
	}

}

func LogInfo(format string, args ...any) {
	log.Println(Green + "[INFO] " + fmt.Sprintf(format, args...) + Reset)
}

func LogError(format string, args ...any) {
	log.Println(Red + "[ERROR] " + fmt.Sprintf(format, args...) + Reset)
}
