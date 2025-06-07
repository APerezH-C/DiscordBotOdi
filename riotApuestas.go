package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"net/http"
	"strings"
	"time"
)

var (
	activeGame     = false
	riotApiKey     = "RGAPI-11188b96-55f9-4ae9-ab43-d28389f55c81"
	summonerName   = "Maestro Shensual"
	summonerTag    = "PALO"
	region         = "euw1"
	channelID      = "519551092166623247"
	currentBets    = make(map[string]Bet)
	bettingOpen    bool
	bettingCloseCh chan struct{}
)

type Bet struct {
	UserID   string
	Username string
	Amount   float64
	Choice   string // "win" o "lose"
}

func watchForGame(s *discordgo.Session) {

	for {
		time.Sleep(1 * time.Minute)

		if !activeGame {
			inGame, gameID, gameType := checkIfInGame()
			if inGame {
				activeGame = true
				matchID = gameID
				currentBets = make(map[string]Bet) // Inicializar mapa vacío
				bettingOpen = true

				s.ChannelMessageSend(channelID, fmt.Sprintf(
					"🎮 %s ha empezado una partida (%s). ¡Apuesten usando `!apuesta win|lose cantidad`!\n"+
						"⚠️ Solo puedes apostar UNA vez por partida.\n"+
						"⏰ Las apuestas se cerrarán en 4 minutos!",
					summonerName, gameType))

				// Iniciar temporizador para cerrar apuestas
				bettingCloseCh = make(chan struct{})
				go closeBettingAfterDelay(s, 4*time.Minute)

				go waitForGameToEnd(s)
			}
		}
	}
}

func closeBettingAfterDelay(s *discordgo.Session, delay time.Duration) {
	select {
	case <-time.After(delay):
		if bettingOpen {
			bettingOpen = false
			s.ChannelMessageSend(channelID, "⏰ Las apuestas están CERRADAS. No se aceptan más apuestas para esta partida.")
		}
	case <-bettingCloseCh:
		// Si el canal se cierra, salir sin hacer nada
		return
	}
}

// Lógica de apuestas y comandos
// Lógica de apuestas y comandos
func riot(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	args := strings.Fields(m.Content)
	if len(args) == 0 {
		return
	}

	switch args[0] {
	case "!apuesta":
		if !activeGame {
			s.ChannelMessageSend(m.ChannelID, "❌ No hay partida activa.")
			return
		}

		if !bettingOpen {
			s.ChannelMessageSend(m.ChannelID, "❌ Las apuestas están cerradas para esta partida.")
			return
		}

		if len(args) != 3 {
			s.ChannelMessageSend(m.ChannelID, "Uso: `!apuesta win|lose cantidad`")
			return
		}

		// Verificar si ya apostó
		if _, exists := currentBets[m.Author.ID]; exists {
			s.ChannelMessageSend(m.ChannelID, "❌ Ya has apostado en esta partida.")
			return
		}

		choice := strings.ToLower(args[1])
		if choice != "win" && choice != "lose" {
			s.ChannelMessageSend(m.ChannelID, "Debes apostar por `win` o `lose`.")
			return
		}

		var amount float64
		_, err := fmt.Sscanf(args[2], "%f", &amount)
		if err != nil || amount <= 0 {
			s.ChannelMessageSend(m.ChannelID, "Cantidad inválida.")
			return
		}

		points := userPoints.Get(m.Author.ID)
		if points < amount {
			s.ChannelMessageSend(m.ChannelID,
				fmt.Sprintf("❌ No tienes suficientes bostes (tienes %.2f).", points))
			return
		}

		userPoints.Add(m.Author.ID, -amount)
		currentBets[m.Author.ID] = Bet{
			UserID:   m.Author.ID,
			Username: m.Author.Username,
			Amount:   amount,
			Choice:   choice,
		}

		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("✅ Apuesta registrada: %.2f bostes por %s", amount, choice))
		_ = userPoints.Save()

	case "!revertirApuesta":
		// Verificar que solo el dueño pueda usar este comando
		if m.Author.ID != "431796013934837761" {
			s.ChannelMessageSend(m.ChannelID, "❌ Solo el dueño del bot puede usar este comando.")
			return
		}

		if !activeGame || len(currentBets) == 0 {
			s.ChannelMessageSend(m.ChannelID, "❌ No hay apuestas activas para revertir.")
			return
		}

		// Revertir todas las apuestas
		revertedCount := 0
		totalReturned := 0.0
		for userID, bet := range currentBets {
			userPoints.Add(userID, bet.Amount)
			revertedCount++
			totalReturned += bet.Amount
			s.ChannelMessageSend(channelID,
				fmt.Sprintf("🔄 Apuesta revertida: <@%s> (%s) ha recuperado %.2f bostes",
					userID, bet.Username, bet.Amount))
		}

		_ = userPoints.Save()
		currentBets = make(map[string]Bet) // Limpiar apuestas

		// Mensaje de resumen
		s.ChannelMessageSend(m.ChannelID,
			fmt.Sprintf("✅ %d apuestas revertidas. Total devuelto: %.2f bostes",
				revertedCount, totalReturned))

		// Opcional: Cerrar apuestas después de revertir
		bettingOpen = false
		if bettingCloseCh != nil {
			close(bettingCloseCh)
		}
	}
}

func waitForGameToEnd(s *discordgo.Session) {

	defer func() {
		if bettingCloseCh != nil {
			close(bettingCloseCh)
			bettingCloseCh = nil
		}
	}()

	// Esperar hasta que la partida termine
	for {
		inGame, _, _ := checkIfInGame()
		if !inGame {
			break
		}
		time.Sleep(1 * time.Minute)
	}

	if bettingOpen {
		bettingOpen = false
		s.ChannelMessageSend(channelID, "⏰ La partida ha terminado. Apuestas cerradas.")
	}

	// Procesar resultados
	won := checkGameResult()

	for userID, bet := range currentBets {
		if (won && bet.Choice == "win") || (!won && bet.Choice == "lose") {
			userPoints.Add(userID, bet.Amount*2)
			s.ChannelMessageSend(channelID,
				fmt.Sprintf("🏆 <@%s> ganó %.2f bostes!", userID, bet.Amount*2))
		} else {
			s.ChannelMessageSend(channelID,
				fmt.Sprintf("💸 <@%s> perdió su apuesta de %.2f bostes", userID, bet.Amount))
		}
	}

	_ = userPoints.Save()
	activeGame = false
	currentBets = make(map[string]Bet) // Resetear apuestas
}

func getSummonerID() (string, string) {
	url := fmt.Sprintf("https://europe.api.riotgames.com/riot/account/v1/accounts/by-riot-id/%s/%s", summonerName, summonerTag)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Riot-Token", riotApiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return "", ""
	}
	defer resp.Body.Close()
	var res struct {
		ID            string `json:"id"`
		Puuid         string `json:"puuid"`
		SummonerLevel int    `json:"summonerLevel"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	return res.ID, res.Puuid
}

func checkIfInGame() (bool, string, string) {
	_, puuid := getSummonerID()
	if puuid == "" {
		return false, "", ""
	}
	url := fmt.Sprintf("https://%s.api.riotgames.com/lol/spectator/v5/active-games/by-summoner/%s", region, puuid)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Riot-Token", riotApiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, "", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		var gameInfo struct {
			GameQueueConfigId int64  `json:"gameQueueConfigId"`
			GameId            int64  `json:"gameId"`
			GameType          string `json:"gameType"`
			GameMode          string `json:"gameMode"`
		}

		err := json.NewDecoder(resp.Body).Decode(&gameInfo)
		if err != nil {
			return false, "", ""
		}

		// Verificar si es una partida custom (personalizada)
		if gameInfo.GameQueueConfigId == 0 || gameInfo.GameType == "CUSTOM_GAME" {
			return false, "", ""
		}

		return true, fmt.Sprintf("%d", gameInfo.GameId), gameInfo.GameType
	}
	return false, "", ""
}

func checkGameResult() bool {
	_, puuid := getSummonerID()
	if puuid == "" {
		return false
	}
	url := fmt.Sprintf("https://%s.api.riotgames.com/lol/match/v5/matches/by-puuid/%s/ids?start=0&count=1", "europe", puuid)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Riot-Token", riotApiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var matchIDs []string
	_ = json.NewDecoder(resp.Body).Decode(&matchIDs)
	if len(matchIDs) == 0 {
		return false
	}

	// Ahora obtenemos el resultado de la partida
	matchURL := fmt.Sprintf("https://%s.api.riotgames.com/lol/match/v5/matches/%s", "europe", matchIDs[0])
	req, _ = http.NewRequest("GET", matchURL, nil)
	req.Header.Set("X-Riot-Token", riotApiKey)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var matchData struct {
		Info struct {
			Participants []struct {
				Puuid        string `json:"puuid"`
				Win          bool   `json:"win"`
				ChampionName string `json:"championName"`
			} `json:"participants"`
		} `json:"info"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&matchData)

	for _, p := range matchData.Info.Participants {
		if p.Puuid == puuid {
			return p.Win
		}
	}
	return false
}
