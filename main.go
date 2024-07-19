package main

import (
	"log"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/joho/godotenv"
	"github.com/pluveto/ankiterm/x/ankicc"
	"github.com/pluveto/ankiterm/x/automata"
	"github.com/pluveto/ankiterm/x/reviewer/chatgpt"
	"github.com/sashabaranov/go-openai"
)

type Args struct {
	BaseURL  string `arg:"-u,--baseURL" help:"Base URL for the server" default:"http://127.0.0.1:8765"`
	Deck     string `arg:"required,positional" help:"Deck name"`
	LANGUAGE string `arg:"-l,--language" help:"Chat response language" default:"English"`
}

func main() {
	var args Args
	arg.MustParse(&args)

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	openaiClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	am := automata.NewAutomata(ankicc.Client{BaseURL: args.BaseURL})
	var reviewer = chatgpt.NewChatGPTReviewer(openaiClient, am, args.LANGUAGE)
	reviewer.Execute(args.Deck)
	return
}
