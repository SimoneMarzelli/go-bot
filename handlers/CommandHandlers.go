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
		HandlerFunc:               get_line_info,
	},
	"current": {
		MinimumArguments:          2,
		NotEnoughParametersErrMsg: "Please specify both the route and direction",
		HandlerFunc:               get_current_info,
	},
}

func get_line_info(args []string) (string, bool, error) {

	directions, err := submodules.GetLineInfo(args[0])

	if err != nil {
		return "", false, err
	}

	return fmt.Sprintf(
		"Choose a direction:\n\t`/current %v %v`\n\t`/current %v %v`",
		args[0], directions[0], args[0], directions[1],
	), false, nil
}

func get_current_info(args []string) (string, bool, error) {

	updates, err := submodules.GetCurrentPosition(args[0], args[1])

	if err != nil {
		return "", false, err
	}

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

	return msg.String(), true, nil
}
