package main

import (
	"fmt"
	"go-bot/submodules"
	"log"
	"os"
	"strings"

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
		if update.Message == nil {
			continue
		}

		split := strings.Fields(update.Message.Text)
		reply := "Unrecognized or malformed command"
		escape := false

		switch split[0] {
		case "/info":
			if len(split) < 2 {
				reply = "Plase specify the route"
			} else if directions, err := submodules.GetLineInfo(split[1]); err == nil {
				reply = fmt.Sprintf(
					"Choose a direction:\n\t`/current %v %v`\n\t`/current %v %v`",
					split[1], directions[0], split[1], directions[1],
				)
			} else {
				reply = err.Error()
			}
		case "/current":
			if len(split) < 3 {
				reply = "Please specify both the route and a direction"
			} else if updates, err := submodules.GetCurrentPosition(split[1], split[2]); err == nil {
				var msg strings.Builder
				for _, update := range updates {
					msg.WriteString(update.Name)
					for _, status := range update.Status {
						var s rune
						switch status {
						case "INCOMING_AT":
							s = '↘'
						case "STOPPED_AT":
							s = '⏸'
						case "IN_TRANSIT_TO":
							s = '↗'
						}

						msg.WriteRune(s)
						msg.WriteRune(' ')
					}
					msg.WriteString("\n")
				}
				reply = msg.String()
				escape = true
			}
		}

		send_message(update, *bot, reply, escape)

	}

}

func send_message(update tgbotapi.Update, bot tgbotapi.BotAPI, text string, escape bool) {
	if escape {
		text = tgbotapi.EscapeText("MarkdownV2", text)
	}
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	msg.ParseMode = "Markdownv2"
	bot.Send(msg)
}
