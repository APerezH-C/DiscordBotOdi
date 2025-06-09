package main

import (
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"strings"
	"sync"
)

type UserStat struct {
	ApuestasTotales int     `bson:"apuestas_totales"`
	TotalApostado   float64 `bson:"total_apostado"`
	TotalGanado     float64 `bson:"total_ganado"`
	NonceActual     int     `bson:"nonce_actual"`
}

type UserStats struct {
	Stats map[string]UserStat `bson:"stats"`
	mu    sync.Mutex
}

var (
	diceEngine = NewDiceEngine()
)

func handleDiceCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {

	// 1. Respuesta inmediata
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		log.Printf("Error respondiendo interacción: %v", err)
		return
	}

	go func() {
		// Extraer opciones
		options := i.ApplicationCommandData().Options

		// Validar que tenemos las 3 opciones requeridas
		if len(options) != 3 {
			respondInteraction(s, i, "Uso: `/dice <under/over> <número> <cantidad>`")
			return
		}

		userID := i.Member.User.ID

		stats, exists := userStats.get(userID)
		if !exists {
			stats = UserStat{}
		}

		// Obtener tipo de apuesta
		betType := strings.ToLower(options[0].StringValue())
		if betType != "under" && betType != "over" {
			respondInteraction(s, i, "Tipo de apuesta inválido. Usa `under` o `over`")
			return
		}

		// Obtener número objetivo
		target := options[1].FloatValue()
		if betType == "under" && (target < underminNumber || target > undermaxNumber) {
			respondInteraction(s, i, fmt.Sprintf("Número inválido. Para under debe ser entre %.2f y %.2f", underminNumber, undermaxNumber))
			return
		}
		if betType == "over" && (target < overminNumber || target > overmaxNumber) {
			respondInteraction(s, i, fmt.Sprintf("Número inválido. Para over debe ser entre %.2f y %.2f", overminNumber, overmaxNumber))
			return
		}

		// Obtener cantidad
		amount := options[2].FloatValue()
		if amount < 1 {
			respondInteraction(s, i, "Cantidad inválida. Mínimo 1 punto")
			return
		}

		// Verificar saldo
		if userPoints.Get(userID) < float64(amount) {
			respondInteraction(s, i, "Saldo insuficiente")
			return
		}

		// Incrementar nonce
		stats.NonceActual++
		clientSeed := fmt.Sprintf("%s_%d", userID, stats.NonceActual)

		// Calcular resultado (manteniendo tu lógica original)
		rollResult := diceEngine.CalculateRoll(clientSeed, stats.NonceActual)
		over := betType == "over"
		win := (over && rollResult > target) || (!over && rollResult < target)
		multiplier := diceEngine.GetMultiplier(target, over)
		payout := amount * multiplier

		// Actualizar estadísticas
		stats.ApuestasTotales++
		stats.TotalApostado += amount

		if win {
			userPoints.Add(userID, payout-amount)
			stats.TotalGanado += payout - amount
		} else {
			userPoints.Add(userID, -amount)
		}

		// Guardar estadísticas
		userStats.mu.Lock()
		userStats.Stats[userID] = stats
		userStats.mu.Unlock()

		userStats.save()
		userPoints.Save()

		// Crear y enviar embed con el resultado
		embed := createDiceEmbed(i.Member.User, rollResult, win, amount, payout, target, betType, clientSeed, stats.NonceActual)

		// Usar followup para enviar el embed después de la respuesta inicial
		_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{embed},
		})

		if err != nil {
			log.Printf("Error enviando resultado de dados: %v", err)
		}
	}()
}

func handleVerifyCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) < 3 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Faltan parámetros. Uso: `/verify <server_seed> <client_seed> <nonce>`",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	serverSeed := options[0].StringValue()
	clientSeed := options[1].StringValue()
	nonce := int(options[2].IntValue())

	rollResult, _ := diceEngine.VerifyResult(clientSeed, nonce, serverSeed)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Resultado verificado: %.2f", rollResult),
		},
	})
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
