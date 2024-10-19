package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

	session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		messageHandler(s, m)
		if err = session.Open(); err != nil {
			log.Fatalf("Error opening connection: %v", err)
		}
		defer session.Close()

		loadWords()

		fmt.Println("Bot is running...")
		select {}
	})
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

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.Bot {
		return
	}

	score := 0

	for _, word := range words.Bad {
		if strings.Contains(strings.ToLower(m.Content), word.Text) {
			score += word.Price
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("-%d social credit", word.Price))
		}
	}

	for _, word := range words.Good {
		if strings.Contains(strings.ToLower(m.Content), word.Text) {
			score += word.Price
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("+%d social credit", word.Price))
		}
	}

	log.Printf("User %s received %d social credit", m.Author.Username, score)
}
