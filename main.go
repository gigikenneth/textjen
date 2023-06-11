package main

import (
	"fmt"
	"log"
	"time"

	"github.com/NicoNex/echotron/v3"
	cohere "github.com/cohere-ai/cohere-go"
)

// Struct useful for managing internal states in your bot, but it could be of
// any type such as `type bot int64` if you only need to store the chatID.
type bot struct {
	chatID       int64
	echotron.API
	cohereClient *cohere.Client
	promptInput  bool
}

const (
	token        = "TOKEN KEY"
	cohereAPIKey = "API KEY"
)

// This function needs to be of type 'echotron.NewBotFn' and is called by
// the echotron dispatcher upon any new message from a chatID that has never
// interacted with the bot before.
// This means that echotron keeps one instance of the echotron.Bot implementation
// for each chat where the bot is used.
func newBot(chatID int64) echotron.Bot {
	cohereClient, err := cohere.CreateClient(cohereAPIKey)
	if err != nil {
		log.Println(err)
		return nil
	}

	return &bot{
		chatID:       chatID,
		API:          echotron.NewAPI(token),
		cohereClient: cohereClient,
		promptInput:  false,
	}
}

// This method is needed to implement the echotron.Bot interface.
func (b *bot) Update(update *echotron.Update) {
	if update.Message.Text == "/start" {
		b.SendMessage("Hello world", b.chatID, nil)
	} else if update.Message.Text == "/generate" {
		b.SendMessage("Please enter a prompt:", b.chatID, nil)
		b.promptInput = true
	} else if b.promptInput {
		prompt := update.Message.Text
		response, err := b.generateText(prompt)
		if err != nil {
			b.SendMessage(fmt.Sprintf("Error: %s", err.Error()), b.chatID, nil)
			return
		}

		b.SendMessage(response, b.chatID, nil)
		b.promptInput = false
	}
}

// Generate text using the Cohere API
func (b *bot) generateText(prompt string) (string, error) {
	options := cohere.GenerateOptions{
		Model:             "command",
		Prompt:            prompt,
		MaxTokens:         300,
		Temperature:       0.9,
		K:                 0,
		StopSequences:     []string{},
		ReturnLikelihoods: "NONE",
	}

	response, err := b.cohereClient.Generate(options)
	if err != nil {
		return "", err
	}

	return response.Generations[0].Text, nil
}

func main() {
	// This is the entry point of echotron library.
	dsp := echotron.NewDispatcher(token, newBot)
	for {
		err := dsp.Poll()
		if err != nil {
			log.Println("Error polling updates:", err)
		}

		// In case of connection issues, wait 5 seconds before trying to reconnect.
		time.Sleep(5 * time.Second)
	}
}
