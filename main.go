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
		var reply string = "Unrecognized or malformed command"

		switch split[0] {
		case "/info":
			if len(split) < 2 {
				reply = "Plase specify the route"
			} else if directions, err := submodules.GetLineInfo(split[1]); err == nil {
				reply = fmt.Sprintf(
					"Choose a direction:\n\t/current %v %v\n\t/current %v %v",
					split[1], directions[0], split[1], directions[1],
				)
			} else {
				reply = err.Error()
			}
		case "/current":
			if len(split) < 3 {
				reply = "Please specify both the route and a direction"
			} else if ordered_stops, positions, err := submodules.GetCurrentPosition(split[1], split[2]); err == nil {
				var tmp strings.Builder

				for _, stop_name := range ordered_stops {
					bus_states := positions[stop_name]
					tmp.WriteString(stop_name)
					tmp.WriteString(" ")

					for _, bus_state := range bus_states {
						switch bus_state {
						case "INCOMING_AT":
							tmp.WriteString("↘")
						case "STOPPED_AT":
							tmp.WriteString("⏸")
						case "IN_TRANSIT_TO":
							tmp.WriteString("↗")
						}
					}

					tmp.WriteString("\n")

				}

				reply = tmp.String()
			}
		}

		send_message(update, *bot, reply)

	}

}

func send_message(update tgbotapi.Update, bot tgbotapi.BotAPI, text string) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	bot.Send(msg)
}
