package main

import (
	"go-bot/handlers"
	"go-bot/submodules"
	"log"
	"os"
	"strings"
	"unicode"

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

		func() {
			var escape bool
			var reply string
			var err error

			args := strings.Fields(update.Message.CommandArguments())
			commandHandler, ok := handlers.CommandHandlers[update.Message.Command()]

			defer sendMessage(update, *bot, &reply, &escape)
			if !ok {
				reply = "Unrecognized command"
				return
			}

			if len(args) < int(commandHandler.MinimumArguments) {
				reply = commandHandler.NotEnoughParametersErrMsg
				return
			}

			reply, escape, err = commandHandler.HandlerFunc(args)

			if err != nil {
				log.Println(err)
				reply = capitalizeFirstLetter(err.Error())
			}
		}()
	}

}

func sendMessage(update tgbotapi.Update, bot tgbotapi.BotAPI, text *string, escape *bool) {
	if *escape {
		*text = tgbotapi.EscapeText("MarkdownV2", *text)
	}
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, *text)
	msg.ParseMode = "Markdownv2"
	bot.Send(msg)
}

func capitalizeFirstLetter(s string) string {
	if s == "" {
		return ""
	}

	st := make([]rune, len(s))

	for idx, l := range s {
		if idx == 0 && unicode.IsLetter(l) {
			l = unicode.ToUpper(l)
		}
		st[idx] = l
	}

	return string(st)

}
