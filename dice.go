package main

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"strconv"
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

func parseDecimal(input string) (float64, error) {
	// Reemplaza coma por punto para manejar decimal europeo
	cleanInput := strings.ReplaceAll(input, ",", ".")
	val, err := strconv.ParseFloat(cleanInput, 64)
	if err != nil {
		return 0, errors.New("Formato decimal inválido")
	}
	return val, nil
}

func handleDiceCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Extraer opciones
	options := i.ApplicationCommandData().Options

	userID := i.Member.User.ID

	betType := strings.ToLower(options[0].StringValue())
	targetStr := options[1].StringValue()
	target, err := parseDecimal(targetStr)
	if err != nil {
		respondInteraction(s, i, "Número inválido.", true)
		return
	}

	if betType == "under" && (target < underminNumber || target > undermaxNumber) {
		respondInteraction(s, i, fmt.Sprintf("El número para 'under' debe estar entre %.2f y %.2f.", underminNumber, undermaxNumber), true)
		return
	}

	if betType == "over" && (target < overminNumber || target > overmaxNumber) {
		respondInteraction(s, i, fmt.Sprintf("El número para 'over' debe estar entre %.2f y %.2f.", overminNumber, overmaxNumber), true)
		return
	}

	// Verificar saldo también antes
	amountStr := options[2].StringValue()
	amount, err := parseDecimal(amountStr)
	if userPoints.Get(userID) < float64(amount) {
		respondInteraction(s, i, "Saldo insuficiente.", true)
		return
	}

	// 1. Respuesta inmediata
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	}); err != nil {
		log.Printf("Error respondiendo interacción: %v", err)
		return
	}

	go func() {
		stats, exists := userStats.get(userID)
		if !exists {
			stats = UserStat{}
		}

		// Obtener tipo de apuesta
		betType := strings.ToLower(options[0].StringValue())

		// Obtener número objetivo
		targetStr := options[1].StringValue()
		target, err := parseDecimal(targetStr)

		// Obtener cantidad
		amountStr := options[2].StringValue()
		amount, err := parseDecimal(amountStr)

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
			boteAmount := amount * 0.01
			userPoints.Add(userID, payout-amount)
			stats.TotalGanado += payout - amount

			if err := AddToBote(boteAmount); err != nil {
				log.Printf("Error añadiendo al bote: %v", err)
			}
		} else {
			boteAmount := amount * 0.25
			userPoints.Add(userID, -amount)

			if err := AddToBote(boteAmount); err != nil {
				log.Printf("Error añadiendo al bote: %v", err)
			}
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
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
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
