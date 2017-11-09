package slackbot

import (
	"fmt"

	"github.com/nlopes/slack"
)

type InteractiveElement interface {
	toAction() slack.AttachmentAction
}

type MessageButton struct {
	name  string
	text  string
	value string
}

func NewButton(name string, text string, value string) MessageButton {
	return MessageButton{
		name:  name,
		text:  text,
		value: value,
	}
}

type MessageMenu struct {
	name   string
	text   string
	values map[string]string
}

func NewMenu(name string, text string, entries map[string]string) MessageMenu {
	return MessageMenu{
		name:   name,
		text:   text,
		values: entries,
	}
}

func (bot *Bot) Message(channel string, msg string) {
	if bot.config.Offline {
		fmt.Printf("< %s\n", msg)
	} else {
		bot.client.PostMessage(channel, msg, slack.NewPostMessageParameters())
	}
}

func (bot *Bot) InteractiveMessage(channel string, text string, callback string, elems []InteractiveElement) {
	if bot.config.Offline {
		fmt.Println("# interactive messages not supported in offline mode")
		return
	}

	parm := slack.NewPostMessageParameters()

	attch := slack.Attachment{
		Fallback:   text,
		CallbackID: callback,
		Actions:    make([]slack.AttachmentAction, len(elems)),
	}

	for i, e := range elems {
		attch.Actions[i] = e.toAction()
	}

	parm.Attachments = []slack.Attachment{attch}

	bot.client.PostMessage(channel, text, parm)
}

func (mb MessageButton) toAction() slack.AttachmentAction {
	return slack.AttachmentAction{
		Name:  mb.name,
		Text:  mb.text,
		Type:  "button",
		Value: mb.value,
	}
}

func (mm MessageMenu) toAction() slack.AttachmentAction {
	opts := make([]slack.AttachmentActionOption, 0, len(mm.values))
	for value, name := range mm.values {
		opts = append(opts, slack.AttachmentActionOption{
			Value: value,
			Text:  name,
		})
	}

	return slack.AttachmentAction{
		Name:    mm.name,
		Text:    mm.text,
		Type:    "select",
		Options: opts,
	}
}
