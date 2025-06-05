package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"io/ioutil"
	"os"
)

func handleStatsCommand(s *discordgo.Session, m *discordgo.MessageCreate) {
	stats, exists := estadisticas[m.Author.ID]
	if !exists {
		s.ChannelMessageSend(m.ChannelID, "📊 No tienes estadísticas registradas aún")
		return
	}

	profit := stats.TotalGanado - stats.TotalApostado
	winRate := 0.0
	if stats.ApuestasTotales > 0 {
		winRate = (stats.TotalGanado / stats.TotalApostado) * 100
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 Estadísticas de %s", m.Author.Username),
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Apuestas Totales",
				Value: fmt.Sprintf("%d", stats.ApuestasTotales),
			},
			{
				Name:   "Total Apostado",
				Value:  fmt.Sprintf("%.2f", stats.TotalApostado),
				Inline: true,
			},
			{
				Name:   "Total Ganado",
				Value:  fmt.Sprintf("%.2f", stats.TotalGanado),
				Inline: true,
			},
			{
				Name:  "Profit Neto",
				Value: fmt.Sprintf("%.2f", profit),
			},
			{
				Name:  "Porcentaje de Retorno",
				Value: fmt.Sprintf("%.2f%%", winRate),
			},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: m.Author.AvatarURL(""),
		},
	}

	s.ChannelMessageSendEmbed(m.ChannelID, embed)
}

func saveStats() error {
	data, err := json.MarshalIndent(estadisticas, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(statsFile, data, 0644)
}

// Cargar estadísticas desde archivo
func loadStats() error {
	if _, err := os.Stat(statsFile); os.IsNotExist(err) {
		// Archivo no existe, no hay nada que cargar
		return nil
	}

	data, err := ioutil.ReadFile(statsFile)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &estadisticas)
}
