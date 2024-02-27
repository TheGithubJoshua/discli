package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	configFile = "config.json"
)

type Config struct {
	Token string `json:"token"`
}

type Message struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Author    Author `json:"author"`
	Timestamp string `json:"timestamp"`
}

type Author struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

func main() {
	config, err := loadConfig(configFile)
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter channel ID: ")
	channelID, _ := reader.ReadString('\n')
	channelID = strings.TrimSpace(channelID)

	messageChan := make(chan struct{})
	go pollMessages(config.Token, channelID, messageChan)

	for {
		sendMessage(config.Token, channelID)
		<-messageChan // Wait for a new message to be sent before sending another
	}
}

func loadConfig(filename string) (Config, error) {
	var config Config
	file, err := os.Open(filename)
	if err != nil {
		return config, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&config)
	return config, err
}

func pollMessages(token, channelID string, messageChan chan struct{}) {
	lastMessageID := ""
	for {
		messages, err := getMessages(token, channelID, lastMessageID)
		if err != nil {
			fmt.Println("Error retrieving messages:", err)
			continue
		}

		for _, msg := range messages {
			fmt.Printf("[%s] %s: %s\n", msg.Timestamp, msg.Author.Username, msg.Content)
			lastMessageID = msg.ID
		}

		if len(messages) > 0 {
			messageChan <- struct{}{} // Signal that a new message has been received
		}

		time.Sleep(1 * time.Second) // Polling interval
	}
}

func getMessages(token, channelID, lastMessageID string) ([]Message, error) {
	url := fmt.Sprintf("https://discord.com/api/v9/channels/%s/messages?limit=1", channelID)

	if lastMessageID != "" {
		url += "&after=" + lastMessageID
	}

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %s", res.Status)
	}

	var messages []Message
	if err := json.NewDecoder(res.Body).Decode(&messages); err != nil {
		return nil, err
	}

	return messages, nil
}

func sendMessage(token, channelID string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter message: ")
	message, _ := reader.ReadString('\n')
	message = strings.TrimSpace(message)

	if message == "" {
		return
	}

	url := fmt.Sprintf("https://discord.com/api/v9/channels/%s/messages", channelID)
	payload := strings.NewReader(fmt.Sprintf(`{"content":"%s"}`, message))

	req, _ := http.NewRequest("POST", url, payload)
	req.Header.Add("Authorization", token)
	req.Header.Add("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error sending message:", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		fmt.Println("Error:", res.Status)
		return
	}

	fmt.Println("Message sent successfully!")
}
