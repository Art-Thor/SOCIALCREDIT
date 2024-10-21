package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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
var originalNicknames = make(map[string]string)

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

	// Check for bad words
	for _, word := range words.Bad {
		if strings.Contains(contentLower, strings.ToLower(word.Text)) {
			score -= word.Price
			s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("-%d social credit", word.Price), reference)
		}
	}

	// Check for good words
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

	// Threshold for changing nickname
	negativeThreshold := -1000

	// Check if the user has reached the negative threshold
	if newCredit <= negativeThreshold && prevCredit > negativeThreshold {
		// Save the original nickname if it hasn't been saved yet
		if _, exists := originalNicknames[userID]; !exists {
			member, err := s.GuildMember(m.GuildID, userID)
			if err != nil {
				log.Printf("Error fetching guild member: %v", err)
				return
			}
			if member.Nick != "" {
				originalNicknames[userID] = member.Nick
			} else {
				originalNicknames[userID] = member.User.Username
			}
		}

		// Generate a unique nickname "prisoner #N"
		prisonerNumber := 1
		existingNicknames := make(map[string]bool)

		// Get the list of all server members
		after := ""
		for {
			members, err := s.GuildMembers(m.GuildID, after, 1000)
			if err != nil {
				log.Printf("Error fetching guild members: %v", err)
				break
			}
			if len(members) == 0 {
				break
			}
			for _, member := range members {
				nick := member.Nick
				if nick == "" {
					nick = member.User.Username
				}
				existingNicknames[nick] = true
				after = member.User.ID
			}
			if len(members) < 1000 {
				break
			}
		}

		// Find a unique prisoner number
		for {
			newNickname := fmt.Sprintf("prisoner #%d", prisonerNumber)
			if !existingNicknames[newNickname] {
				break
			}
			prisonerNumber++
		}

		// Change the user's nickname
		err := s.GuildMemberNickname(m.GuildID, userID, fmt.Sprintf("prisoner #%d", prisonerNumber))
		if err != nil {
			log.Printf("Error changing nickname: %v", err)
			return
		}

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s has been renamed to prisoner #%d", m.Author.Mention(), prisonerNumber))
	}

	// Check if the user has been rehabilitated
	if newCredit > negativeThreshold && prevCredit <= negativeThreshold {
		if originalNickname, exists := originalNicknames[userID]; exists {
			err := s.GuildMemberNickname(m.GuildID, userID, originalNickname)
			if err != nil {
				log.Printf("Error restoring nickname: %v", err)
				return
			}
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("%s has been rehabilitated, and their nickname has been restored.", m.Author.Mention()))
			delete(originalNicknames, userID)
		}
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
