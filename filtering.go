package slackbot

import (
	"strings"

	"github.com/nlopes/slack"
)

type Filterer interface {
	filter(msg *slack.Msg) bool
}

type dmfilter struct{}

func (f dmfilter) filter(msg *slack.Msg) bool {
	return msg.Type == "message" &&
		(msg.SubType != "message_deleted" && msg.SubType != "bot_message") &&
		strings.HasPrefix(msg.Channel, "D")
}

var DMFilter = dmfilter{}

type SingleUserFilter struct {
	ID string
}

func (f SingleUserFilter) filter(msg *slack.Msg) bool {
	return msg.Type == "message" &&
		(msg.SubType != "message_deleted" && msg.SubType != "bot_message") &&
		msg.User != f.ID &&
		(strings.HasPrefix(msg.Text, "<@"+f.ID+">") || strings.HasPrefix(msg.Channel, "D"))
}
