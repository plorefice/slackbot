package slackbot

import (
	"fmt"

	"github.com/nlopes/slack"
)

type InteractiveElement interface {
	toAction() slack.AttachmentAction
}

type MessageButton struct {
	Name  string
	Text  string
	Value string
}

type MessageMenu struct {
	Name   string
	Text   string
	Values map[string]string
}

type MessageFormat struct {
	Callback string
	Elements []InteractiveElement
}

func (bot *Bot) Message(channel string, msg string) {
	if bot.config.Offline {
		fmt.Printf("< %s\n", msg)
	} else {
		bot.client.PostMessage(channel, msg, slack.NewPostMessageParameters())
	}
}

func (bot *Bot) InteractiveMessage(channel string, text string, msg MessageFormat) {
	if bot.config.Offline {
		fmt.Println("# interactive messages not supported in offline mode")
		return
	}

	parm := slack.NewPostMessageParameters()

	attch := slack.Attachment{
		Fallback:   text,
		CallbackID: msg.Callback,
		Actions:    make([]slack.AttachmentAction, len(msg.Elements)),
	}

	for i, e := range msg.Elements {
		attch.Actions[i] = e.toAction()
	}

	parm.Attachments = []slack.Attachment{attch}

	bot.client.PostMessage(channel, text, parm)
}

func (mb MessageButton) toAction() slack.AttachmentAction {
	return slack.AttachmentAction{
		Name:  mb.Name,
		Text:  mb.Text,
		Type:  "button",
		Value: mb.Value,
	}
}

func (mm MessageMenu) toAction() slack.AttachmentAction {
	opts := make([]slack.AttachmentActionOption, 0, len(mm.Values))
	for value, name := range mm.Values {
		opts = append(opts, slack.AttachmentActionOption{
			Value: value,
			Text:  name,
		})
	}

	return slack.AttachmentAction{
		Name:    mm.Name,
		Text:    mm.Text,
		Type:    "select",
		Options: opts,
	}
}
