package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"strconv"
	"sync"
)

const (
	houseEdge        = 1     // 1% de ventaja para la casa
	serverSeedLength = 32    // Longitud del server seed (64 caracteres hex)
	underminNumber   = 0.02  // Número mínimo para apostar
	undermaxNumber   = 98.00 // Número máximo para apostar
	overminNumber    = 1.99  // Número mínimo para apostar
	overmaxNumber    = 99.98 // Número máximo para apostar
)

type DiceEngine struct {
	serverSeed   string
	previousSeed string
	seedMutex    sync.Mutex
}

func NewDiceEngine() *DiceEngine {
	de := &DiceEngine{}
	de.RotateSeed()
	return de
}

// RotateSeed genera una nueva semilla del servidor
func (de *DiceEngine) RotateSeed() {
	de.seedMutex.Lock()
	defer de.seedMutex.Unlock()

	b := make([]byte, serverSeedLength)
	_, err := rand.Read(b)
	if err != nil {
		panic("Error generando semilla: " + err.Error())
	}

	de.previousSeed = de.serverSeed
	de.serverSeed = hex.EncodeToString(b)

	fmt.Printf("\n🔐 SEMILLA ACTUAL: %s\n", de.serverSeed)
	fmt.Printf("🔓 SEMILLA ANTERIOR: %s\n\n", de.previousSeed)
}

// CalculateRoll calcula el resultado basado en las semillas
func (de *DiceEngine) CalculateRoll(clientSeed string, nonce int) float64 {
	h := hmac.New(sha256.New, []byte(de.serverSeed))
	h.Write([]byte(fmt.Sprintf("%s-%d", clientSeed, nonce)))
	hash := hex.EncodeToString(h.Sum(nil))

	rollHex := hash[:5]
	roll, _ := strconv.ParseInt(rollHex, 16, 64)
	return float64(roll%10000) / 100.0 // Resultado entre 0.00 y 99.99
}

// GetMultiplier calcula el multiplicador con house edge
func (de *DiceEngine) GetMultiplier(target float64, over bool) float64 {
	if over {
		return (100.0 - houseEdge) / (100.0 - target)
	}
	return (100.0 - houseEdge) / target
}

// VerifyResult permite verificar un resultado con semilla anterior
func (de *DiceEngine) VerifyResult(clientSeed string, nonce int, serverSeed string) (float64, bool) {
	if serverSeed != de.previousSeed {
		return 0, false
	}

	h := hmac.New(sha256.New, []byte(serverSeed))
	h.Write([]byte(fmt.Sprintf("%s-%d", clientSeed, nonce)))
	hash := hex.EncodeToString(h.Sum(nil))

	rollHex := hash[:5]
	roll, _ := strconv.ParseInt(rollHex, 16, 64)
	return float64(roll%10000) / 100.0, true
}

func handleNewSeedCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	userID := i.Member.User.ID

	if userID != "431796013934837761" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ No tienes permiso.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	oldSeed := diceEngine.serverSeed
	diceEngine.RotateSeed()

	newSeedPreview := ""
	if len(diceEngine.serverSeed) >= 8 {
		newSeedPreview = diceEngine.serverSeed[:8]
	} else {
		newSeedPreview = diceEngine.serverSeed
	}

	msg := fmt.Sprintf(
		"🔁 Semilla regenerada.\n🔓 Semilla anterior (para verificar): `%s`\n🔐 Nueva semilla (solo primeros 8): `%s...`",
		oldSeed,
		newSeedPreview,
	)

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
}
