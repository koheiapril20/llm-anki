package chatgpt

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/pluveto/ankiterm/x/automata"
	"github.com/pluveto/ankiterm/x/reviewer"
	"github.com/pluveto/ankiterm/x/xmisc"
	"github.com/sashabaranov/go-openai"
)

const INSTRUCTION = "You assist the user in memorizing with the use of flashcards. You have access to the flashcard database, so you can generate questions from it, evaluate the user’s answers, and engage in conversations that help reinforce the user’s memory based on the answers and your knowledge."

type ChatGPTReviewer struct {
	client   *openai.Client
	am       *automata.Automata
	messages []openai.ChatCompletionMessage
	language string
}

func printUsage() {
	fmt.Println("Enter anything to converse with the chatbot. Enter ‘n’ to move to the next card, and ‘a’ to quit.")
}

func NewChatGPTReviewer(client *openai.Client, am *automata.Automata, language string) *ChatGPTReviewer {
	return &ChatGPTReviewer{
		client:   client,
		am:       am,
		language: language,
	}
}

func (r *ChatGPTReviewer) clearHistory() {
	r.messages = []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: fmt.Sprintf("%s Speak in %s", INSTRUCTION, r.language),
		},
	}
}

func (r *ChatGPTReviewer) createChatCompletion() (
	reply string,
	err error,
) {
	resp, err := r.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    "gpt-4o-mini",
			Messages: r.messages,
		},
	)
	if err != nil {
		return "", err
	}
	var replyMessage = resp.Choices[0].Message
	var replyStr = replyMessage.Content
	if replyStr != "" {
		r.appendMessage(replyMessage)
	}

	return replyStr, nil
}

func (r *ChatGPTReviewer) createFeedbackForUserAnswer(userAnswer string) (
	reply string,
	err error,
) {
	resp, err := r.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "gpt-4o-mini",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: fmt.Sprintf("You are presenting a question to the user and will receive the user's answer to that question. Please provide an evaluative comment on the user's response based on your knowledge and the given correct answer. The question and the correct answer are as follows. Just explain the correct answer when the user's input is empty, as it means that the user doesn't have any idea. Speak in %s. \n\nQuestion:\n\n%s\n\n\nAnswer:\n\n%s", r.language, format(r.am.CurrentCard().Question), format(r.am.CurrentCard().Answer)),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userAnswer,
				},
			},
		},
	)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

func (r *ChatGPTReviewer) appendMessage(message openai.ChatCompletionMessage) {
	r.messages = append(r.messages, message)
}

func awaitInput() (input string, err error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	input, err = reader.ReadString('\n')
	if err != nil {
		return
	}
	// Remove the trailing newline character
	input = input[:len(input)-1]
	return
}

func (r *ChatGPTReviewer) awaitActionAnswer() {
	for {
		input, err := awaitInput()
		action := reviewer.ActionFromString(input)
		if action == nil || action.GetCode() != reviewer.ActionAnswer {
			fmt.Println("Please select the number.")
			continue
		}
		if r.am.AnswerCard(action.(reviewer.AnswerAction).CardEase); err != nil {
			fmt.Println("Please select the number.")
			continue
		}
		r.appendMessage(openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: input,
		})
		return
	}
}

func (r *ChatGPTReviewer) Execute(deck string) (err error) {
	r.clearHistory()
	printUsage()
	if err := r.am.StartReview(deck); err != nil {
		return err
	}
	defer r.am.StopReview()
	for {

		input, err := awaitInput()

		if r.am.NeedsAnswer() {
			var reply string
			// add feedback
			if feedback, err := r.createFeedbackForUserAnswer(input); err != nil {
				return err
			} else {
				reply = fmt.Sprintf("[assistant] %s\n", feedback)
			}
			// add correct answer
			reply += fmt.Sprintf("Answer:\n%s\n\n", format(r.am.CurrentCard().Answer))
			// add buttons info
			reply += "Select:\n"
			lookup := []string{"Again", "Hard", "Good", "Easy"}
			for i, button := range r.am.CurrentCard().Buttons {
				reply += fmt.Sprintf("[%d] %s (%s)\n", button, lookup[i], r.am.CurrentCard().NextReviews[i])
			}
			fmt.Println(reply)
			r.appendMessage(openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: reply,
			})
			r.am.PlayAudio()
			r.awaitActionAnswer()
			r.am.StopAudio()
			continue
		}

		action := reviewer.ActionFromString(input)

		if action != nil {
			switch code := action.GetCode(); code {
			case reviewer.ActionAbort:
				fmt.Println("action: abort")
				return nil
			case reviewer.ActionNext:
				fmt.Println("action: next")
				r.clearHistory()
				if _, err := r.am.NextCard(); err != nil {
					return err
				}
				fmt.Println(r.am.CurrentCard().GetAudioFilenames())
				fmt.Printf("Question: %s\n", format(r.am.CurrentCard().Question))
				break
			case reviewer.ActionAnswer:
				num := action.(reviewer.AnswerAction).CardEase
				fmt.Printf("action: answer %d", num)
				r.am.AnswerCard(num)
				break
			default:
				return fmt.Errorf("Unknown action: %s", code)
			}
			continue
		}

		r.appendMessage(openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: input,
		})
		reply, err := r.createChatCompletion()
		if err != nil {
			fmt.Printf("Completion error: %v\n", err)
			return err
		}

		fmt.Printf("[assistant] %s\n", reply)

	}
}

func format(text string) string {
	text = xmisc.PurgeStyle(text)
	text = xmisc.TtyColor(text)
	return text
}
