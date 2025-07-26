package main

import (
	"fmt"
	"go-bot/submodules"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var BOT_API_KEY = os.Getenv("BOT_API_KEY")

func main() {

	go submodules.StartFetching()

	bot, err := tgbotapi.NewBotAPI(BOT_API_KEY)
	if err != nil {
		log.Panic(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			bus_code := update.Message.Text

			info, err := submodules.GetLineInfo(bus_code)
			if err != nil {
				send_message(update, *bot, "There was an error")
			}
			send_message(update, *bot, info)
		}
	}

	fmt.Println("Ziocan")

}

func send_message(update tgbotapi.Update, bot tgbotapi.BotAPI, text string) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	bot.Send(msg)
}
