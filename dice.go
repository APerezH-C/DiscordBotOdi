package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strconv"
	"strings"
)

type UserStats struct {
	ApuestasTotales int     `json:"apuestas_totales"`
	TotalApostado   float64 `json:"total_apostado"`
	TotalGanado     float64 `json:"total_ganado"`
	NonceActual     int     `json:"nonce_actual"`
}

var (
	diceEngine   = NewDiceEngine()
	estadisticas = make(map[string]UserStats)
)

func handleDiceCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 3 {
		s.ChannelMessageSend(m.ChannelID, "Uso: `!bosteDice <under/over> <número> <cantidad>`")
		return
	}

	userID := m.Author.ID
	stats := estadisticas[userID]

	// Validar tipo de apuesta
	betType := strings.ToLower(args[0])
	if betType != "under" && betType != "over" {
		s.ChannelMessageSend(m.ChannelID, "Tipo de apuesta inválido. Usa `under` o `over`")
		return
	}

	// Validar número objetivo
	target, err := strconv.ParseFloat(args[1], 64)
	if betType == "under" && (target < underminNumber || target > undermaxNumber) {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Número inválido. Para under debe ser entre %.2f y %.2f", underminNumber, undermaxNumber))
		return
	}
	if betType == "over" && (target < overminNumber || target > overmaxNumber) {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Número inválido. Para over debe ser entre %.2f y %.2f", overminNumber, overmaxNumber))
		return
	}

	// Validar cantidad
	amount, err := strconv.ParseFloat(args[2], 64)
	if err != nil || amount < 1 {
		s.ChannelMessageSend(m.ChannelID, "Cantidad inválida. Mínimo 1 punto")
		return
	}

	// Verificar saldo
	if userPoints.Get(userID) < float64(amount) {
		s.ChannelMessageSend(m.ChannelID, "Saldo insuficiente")
		return
	}

	// Incrementar nonce
	stats.NonceActual++
	clientSeed := fmt.Sprintf("%s_%d", userID, stats.NonceActual)

	// Calcular resultado
	rollResult := diceEngine.CalculateRoll(clientSeed, stats.NonceActual)
	over := betType == "over"
	win := (over && rollResult > target) || (!over && rollResult < target)
	multiplier := diceEngine.GetMultiplier(target, over)
	payout := amount * multiplier

	// Actualizar estadísticas
	stats.ApuestasTotales++
	stats.TotalApostado += amount

	if win {
		userPoints.Add(userID, float64(payout)-amount)
		stats.TotalGanado += payout - amount
	} else {
		userPoints.Add(userID, -float64(amount))
	}

	estadisticas[userID] = stats
	saveStats()
	userPoints.Save("points.json")

	// Mostrar resultado
	embed := createDiceEmbed(m.Author, rollResult, win, amount, payout, target, betType, clientSeed, stats.NonceActual)
	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func handleVerifyCommand(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	if len(args) < 3 {
		s.ChannelMessageSend(m.ChannelID, "Uso: `!verify <server_seed> <client_seed> <nonce>`")
		return
	}

	serverSeed := args[0]
	clientSeed := args[1]
	nonce, err := strconv.Atoi(args[2])
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Nonce inválido")
		return
	}

	rollResult, _ := diceEngine.VerifyResult(clientSeed, nonce, serverSeed)

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("✅ Resultado verificado: %.2f", rollResult))
}

func createDiceEmbed(user *discordgo.User, roll float64, win bool, amount, payout, target float64, betType, clientSeed string, nonce int) *discordgo.MessageEmbed {
	color := 0xFF0000 // Rojo
	resultText := "PERDISTE"
	if win {
		color = 0x00FF00 // Verde
		resultText = "GANASTE"
	}

	return &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🎲 Provably Fair Dice (%.1f%% Edge)", float64(houseEdge)),
		Color: color,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Apuesta", Value: fmt.Sprintf("```%.2f en %s %.2f```", amount, betType, target), Inline: true},
			{Name: "Multiplicador", Value: fmt.Sprintf("```%.2fx```", payout/amount), Inline: true},
			{Name: "Resultado", Value: fmt.Sprintf("```%.2f```", roll), Inline: true},
			{Name: "Estado", Value: fmt.Sprintf("```%s```", resultText), Inline: true},
			{Name: "Balance", Value: fmt.Sprintf("```%+.2f```", map[bool]float64{true: payout, false: -amount}[win]), Inline: true},
			{Name: "Verificación", Value: fmt.Sprintf("```Client Seed: %s\nNonce: %d```", clientSeed, nonce), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Server Seed: %s... (usa !verify cuando cambie)", diceEngine.serverSeed[:8]),
			IconURL: user.AvatarURL(""),
		},
	}
}
