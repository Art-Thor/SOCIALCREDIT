package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

type Word struct {
	Text  string `json:"text"`
	Regex string `json:"regex,omitempty"`
	Price int    `json:"price"`
}

type Words struct {
	Good []Word `json:"good"`
	Bad  []Word `json:"bad"`
}

var words Words
var token string
var userCredits = make(map[string]int)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	token = os.Getenv("DISCORD_TOKEN")

	session, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("Error creating Discord session: %v", err)
	}

	session.AddHandler(messageHandler)

	loadWords()
	loadUserCredits()

	err = session.Open()
	if err != nil {
		log.Fatalf("Error opening connection: %v", err)
	}
	defer session.Close()

	fmt.Println("Bot is running...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
}

func loadWords() {
	data, err := ioutil.ReadFile("words.json")
	if err != nil {
		log.Fatalf("Error reading words.json: %v", err)
	}

	if err := json.Unmarshal(data, &words); err != nil {
		log.Fatalf("Error parsing words.json: %v", err)
	}
}

func loadUserCredits() {
	data, err := ioutil.ReadFile("userCredits.json")
	if err != nil {
		if os.IsNotExist(err) {
			userCredits = make(map[string]int)
			return
		}
		log.Fatalf("Error reading userCredits.json: %v", err)
	}

	if err := json.Unmarshal(data, &userCredits); err != nil {
		log.Fatalf("Error parsing userCredits.json: %v", err)
	}
}

func saveUserCredits() {
	data, err := json.MarshalIndent(userCredits, "", "  ")
	if err != nil {
		log.Printf("Error marshaling userCredits: %v", err)
		return
	}

	if err := ioutil.WriteFile("userCredits.json", data, 0644); err != nil {
		log.Printf("Error writing userCredits.json: %v", err)
	}
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	var score int
	contentLower := strings.ToLower(m.Content)
	reference := &discordgo.MessageReference{
		MessageID: m.ID,
		ChannelID: m.ChannelID,
		GuildID:   m.GuildID,
	}

	// Check bad words
	for _, word := range words.Bad {
		if strings.Contains(contentLower, strings.ToLower(word.Text)) {
			score -= word.Price
			s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("-%d social credit", word.Price), reference)
		}
	}

	// Check good words
	for _, word := range words.Good {
		if strings.Contains(contentLower, strings.ToLower(word.Text)) {
			score += word.Price
			s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("+%d social credit", word.Price), reference)
		}
	}

	userID := m.Author.ID
	prevCredit := userCredits[userID]
	userCredits[userID] += score
	newCredit := userCredits[userID]

	saveUserCredits()

	// Check thresholds
	prevThreshold := int(math.Floor(float64(prevCredit) / -1000.0))
	newThreshold := int(math.Floor(float64(newCredit) / -1000.0))

	if newThreshold > prevThreshold {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, you have reached %d social credit! Threatening to send to gulag!", m.Author.Mention(), newCredit))
	}

	// Check for very low rating
	if newCredit <= -5000 {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s, the prisoner has escaped! Return to the cell!", m.Author.Mention()))
	}

	// Check for the word "diggers"
	if strings.Contains(contentLower, "diggers") {
		file, err := os.Open("images/apes.jpg")
		if err != nil {
			log.Printf("Error opening image: %v", err)
			return
		}
		defer file.Close()

		_, err = s.ChannelFileSend(m.ChannelID, "apes.jpg", file)
		if err != nil {
			log.Printf("Error sending image: %v", err)
		}
	}
}
