package main

import (
	_ "embed"
	"log"
	"strings"
	"time"

	"github.com/NicoNex/echotron/v3"
	"github.com/cohere-ai/cohere-go"
)

// Recursive type definition of the bot state function.
type stateFn func(*echotron.Update) stateFn

// Struct useful for managing internal states in your bot, but it could be of
// any type such as `type bot int64` if you only need to store the chatID.
type bot struct {
	chatID int64
	echotron.API
	cohereClient *cohere.Client
	state        stateFn
}

var (
	//go:embed tgtoken
	token string
	//go:embed chtoken
	cohereAPIKey string

	commands = []echotron.BotCommand{
		{Command: "/start", Description: "Activate the bot."},
		{Command: "/generate", Description: "Generate an answer."},
	}
)

// This function needs to be of type 'echotron.NewBotFn' and is called by
// the echotron dispatcher upon any new message from a chatID that has never
// interacted with the bot before.
// This means that echotron keeps one instance of the echotron.Bot implementation
// for each chat where the bot is used.
func newBot(chatID int64) echotron.Bot {
	cohereClient, err := cohere.CreateClient(cohereAPIKey)
	if err != nil {
		log.Fatalln(err)
	}

	b := &bot{
		chatID:       chatID,
		API:          echotron.NewAPI(token),
		cohereClient: cohereClient,
	}
	b.state = b.handleMessage
	return b
}

func (b *bot) handlePrompt(update *echotron.Update) stateFn {
	b.SendChatAction(echotron.Typing, b.chatID, nil)
	response, err := b.generateText(message(update))
	if err != nil {
		log.Println("handlePrompt", err)
		b.SendMessage("An error occurred!", b.chatID, nil)
		return b.handleMessage
	}
	b.SendMessage(response, b.chatID, nil)
	return b.handleMessage
}

func (b *bot) handleMessage(update *echotron.Update) stateFn {
	switch m := message(update); {

	case strings.HasPrefix(m, "/start"):
		b.SendMessage("Hello world", b.chatID, nil)

	case strings.HasPrefix(m, "/generate"):
		b.SendMessage("Please enter a prompt:", b.chatID, nil)
		return b.handlePrompt
	}
	return b.handleMessage
}

// This method is needed to implement the echotron.Bot interface.
func (b *bot) Update(update *echotron.Update) {
	b.state = b.state(update)
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

// Returns the message from the given update.
func message(u *echotron.Update) string {
	if u.Message != nil {
		return u.Message.Text
	} else if u.EditedMessage != nil {
		return u.EditedMessage.Text
	} else if u.CallbackQuery != nil {
		return u.CallbackQuery.Data
	}
	return ""
}

func main() {
	echotron.NewAPI(token).SetMyCommands(nil, commands...)

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
