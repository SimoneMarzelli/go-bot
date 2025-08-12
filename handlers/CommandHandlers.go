package handlers

import (
	"fmt"
	"go-bot/submodules"
	"strings"
)

type CommandHandler struct {
	MinimumArguments          uint
	NotEnoughParametersErrMsg string
	HandlerFunc               func(command []string) (string, bool, error)
}

var CommandHandlers = map[string]CommandHandler{
	"info": {
		MinimumArguments:          1,
		NotEnoughParametersErrMsg: "Please specify the route",
		HandlerFunc:               getLineInfo,
	},
	"current": {
		MinimumArguments:          2,
		NotEnoughParametersErrMsg: "Please specify both the route and direction",
		HandlerFunc:               getCurrentInfo,
	},
}

func getLineInfo(args []string) (string, bool, error) {

	directions, err := submodules.GetLineInfo(args[0])

	if err != nil {
		return "", false, err
	}

	var msg strings.Builder
	msg.WriteString("Choose a direction:")
	for _, dir := range directions {
		msg.WriteString(fmt.Sprintf("\n\t`/current %v %v`", args[0], dir))
	}

	return msg.String(), false, nil
}

func getCurrentInfo(args []string) (string, bool, error) {

	updates, err := submodules.GetCurrentPosition(args[0], args[1])

	if err != nil {
		return "", false, err
	}

	var msg strings.Builder
	for _, update := range updates {
		msg.WriteString(update.StopName)
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

	return msg.String(), true, nil
}
