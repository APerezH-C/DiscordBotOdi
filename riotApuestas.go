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
	summonerName   = "肿瘤学家"
	summonerTag    = "CNCR"
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
					"<@&%s>\n"+
						"🎮 %s ha empezado una partida (%s). ¡Apuesten usando `!apuesta win|lose cantidad`!\n"+
						"⚠️ Solo puedes apostar UNA vez por partida.\n"+
						"⏰ Las apuestas se cerrarán en 4 minutos! \n"+
						"%s",
					notificationRoleID, summonerName, gameType, fmt.Sprintf("https://op.gg/es/lol/summoners/euw/%s-%s/ingame", summonerName, summonerTag)))

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
func handleBetCommands(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "apuesta":
		options := i.ApplicationCommandData().Options

		if !activeGame {
			respondInteraction(s, i, "❌ No hay partida activa.", true)
			return
		}

		if !bettingOpen {
			respondInteraction(s, i, "❌ Las apuestas están cerradas para esta partida.", true)
			return
		}

		if len(options) != 2 {
			respondInteraction(s, i, "Uso: `/apuesta win|lose cantidad`", true)
			return
		}

		userID := i.Member.User.ID
		if _, exists := currentBets[userID]; exists {
			respondInteraction(s, i, "❌ Ya has apostado en esta partida.", true)
			return
		}

		choice := strings.ToLower(options[0].StringValue())
		if choice != "win" && choice != "lose" {
			respondInteraction(s, i, "Debes apostar por `win` o `lose`.", true)
			return
		}

		amount := options[1].FloatValue()
		if amount <= 0 {
			respondInteraction(s, i, "Cantidad inválida.", true)
			return
		}

		points := userPoints.Get(userID)
		if points < amount {
			respondInteraction(s, i, fmt.Sprintf("❌ No tienes suficientes bostes (tienes %.2f).", points), true)
			return
		}

		userPoints.Add(userID, -amount)
		currentBets[userID] = Bet{
			UserID:   userID,
			Username: i.Member.User.Username,
			Amount:   amount,
			Choice:   choice,
		}

		respondInteraction(s, i, fmt.Sprintf("✅ Apuesta registrada: %.2f bostes por %s", amount, choice), true)
		_ = userPoints.Save()

	case "revertirapuesta":
		if i.Member.User.ID != "431796013934837761" {
			respondInteraction(s, i, "❌ Solo el dueño del bot puede usar este comando.", true)
			return
		}

		if !activeGame || len(currentBets) == 0 {
			respondInteraction(s, i, "❌ No hay apuestas activas para revertir.", true)
			return
		}

		revertedCount := 0
		totalReturned := 0.0
		for userID, bet := range currentBets {
			userPoints.Add(userID, bet.Amount)
			revertedCount++
			totalReturned += bet.Amount

			// Enviar mensaje individual como followup
			s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
				Content: fmt.Sprintf("🔄 Apuesta revertida: <@%s> (%s) ha recuperado %.2f bostes",
					userID, bet.Username, bet.Amount),
			})
		}

		_ = userPoints.Save()
		currentBets = make(map[string]Bet)

		_, _ = s.FollowupMessageCreate(i.Interaction, false, &discordgo.WebhookParams{
			Content: fmt.Sprintf("✅ %d apuestas revertidas. Total devuelto: %.2f bostes",
				revertedCount, totalReturned),
		})

		bettingOpen = false
		if bettingCloseCh != nil {
			closeBettingChannel()
		}
	}
}

func waitForGameToEnd(s *discordgo.Session) {

	defer func() {
		if bettingCloseCh != nil {
			closeBettingChannel()
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
		} else if gameInfo.GameQueueConfigId == 440 {
			return true, fmt.Sprintf("%d", gameInfo.GameId), "FLEX"
		} else if gameInfo.GameQueueConfigId == 400 {
			return true, fmt.Sprintf("%d", gameInfo.GameId), "NORMAL"
		} else if gameInfo.GameQueueConfigId == 430 {
			return true, fmt.Sprintf("%d", gameInfo.GameId), "NORMAL BLIND"
		} else if gameInfo.GameQueueConfigId == 450 {
			return true, fmt.Sprintf("%d", gameInfo.GameId), "ARAM"
		}

		return true, fmt.Sprintf("%d", gameInfo.GameId), "RANKED"
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

func closeBettingChannel() {
	if bettingCloseCh != nil {
		select {
		case <-bettingCloseCh:
			// Ya fue cerrado
		default:
			close(bettingCloseCh)
		}
		bettingCloseCh = nil
	}
}
