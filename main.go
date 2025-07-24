package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	gtfs "github.com/MobilityData/gtfs-realtime-bindings/golang/gtfs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	proto "google.golang.org/protobuf/proto"
)

const LIVE_UPDATE_URI = "proto/rome_rtgtfs_vehicle_positions_feed.pb"
const LIVE_UPDATE_URL = "https://romamobilita.it/sites/default/files/rome_rtgtfs_vehicle_positions_feed.pb"

var BOT_API_KEY = os.Getenv("BOT_API_KEY")

var feed gtfs.FeedMessage

func main() {

	go fetch_routine()
	go run_bot()

	select {}

}

func get_line_info(route_id string) (string, error) {

	var ret strings.Builder

	for _, entity := range feed.Entity {
		vehicle_info := entity.Vehicle

		if strings.Compare(vehicle_info.Trip.GetRouteId(), route_id) != 0 {
			continue
		}

		ret.WriteString(fmt.Sprintf(
			"%v stop %v, destination %v\n\n",
			strings.Map(
				func(r rune) rune {
					if r == '_' {
						return ' '
					} else {
						return unicode.ToLower(r)
					}
				},
				vehicle_info.GetCurrentStatus().String(),
			),
			vehicle_info.GetStopId(),
			vehicle_info.Trip.GetDirectionId(),
		))

	}

	return ret.String(), nil
}

func download_data() {
	response, err := http.Get(LIVE_UPDATE_URL)
	if err != nil {
		log.Fatal("Error downloading update")
	}

	defer response.Body.Close()

	out, err := os.Create(LIVE_UPDATE_URI)
	if err != nil {
		log.Fatal("Could not fetch live data")
	}

	_, err = io.Copy(out, response.Body)
	if err == nil {
		log.Println("Refreshed live updates")
		data, _ := os.ReadFile(LIVE_UPDATE_URI)
		proto.Unmarshal(data, &feed)
	}
}

func fetch_routine() {
	ticker := time.NewTicker(60 * time.Second)

	for {
		go download_data()
		<-ticker.C
	}
}

func send_message(update tgbotapi.Update, bot tgbotapi.BotAPI, text string) {
	msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
	bot.Send(msg)
}

func run_bot() {
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

			info, err := get_line_info(bus_code)
			if err != nil {
				send_message(update, *bot, "There was an error")
			}
			send_message(update, *bot, info)
		}
	}
}
