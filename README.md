# go-telegram-bot-api-questions
Library to deal with questions and answers in go-telegram-bot

Example
```
package main

import (
	"os"
	"log"
	"github.com/GolangResources/go-telegram-bot-api-questions"
	"github.com/dgraph-io/ristretto"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	tgq, err := tgquestions.Init(tgquestions.Config{
		Bot: bot,
		RistrettoConfig: &ristretto.Config{
				NumCounters: 1e7,     // number of keys to track frequency of (10M).
				MaxCost:     1 << 30, // maximum cost of cache (1GB).
				BufferItems: 64,      // number of keys per Get buffer.
				},
	})
	if err != nil {
		panic(err)
	}

	for update := range updates {
		if tgq.Update(update) != false {
			continue
		}
		if update.Message != nil {
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "bool":
					tgq.DoBoolQuestion(tgquestions.BoolQuestionConfig{
						ID: update.Message.Chat.ID,
						Question: "¿Esto es una prueba?",
						TextTrue: "Si",
						CallbackTrue: func() {
							log.Println("OK")
						},
						TextFalse: "No",
						CallbackFalse: func() {
							log.Println("KO")
						},
					})
				case "string":
					resultStrChan := make(chan string, 1)
					tgq.DoQuestion(tgquestions.QuestionConfig{
						ID: update.Message.Chat.ID,
						Result: resultStrChan,
						Question: "¿Como te llamas?",
						Callback: func() {
							name := <-resultStrChan
							log.Println("Name", name)
						},
						DoubleCheck: true,
						DoubleCheckQuestion: "¿Estás seguro que te llamas %s?",
						DoubleCheckButtonYes: "Si",
						DoubleCheckButtonNo: "No",
					})
				}
			}
		}
	}
}
```
