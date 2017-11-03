package slackbot

import (
	"errors"
	"reflect"

	"github.com/nlopes/slack"
)

type FlowAction func(*Bot, *slack.Msg, interface{}) bool
type Trigger func(*Bot, *slack.Msg) bool

type State struct {
	name string
	act  FlowAction
	dst  string
}

type StateBuilder struct {
	state *State
}

func (sb *StateBuilder) To(state string) *StateBuilder {
	sb.state.dst = state
	return sb
}

func (sb *StateBuilder) Build() *State {
	return sb.state
}

type flowContext struct {
	currentState *State
	userCtx      interface{}
}

type Flow struct {
	bot *Bot

	name         string
	states       map[string]*State
	initialState *State
	userCtxTmpl  interface{}

	trigger Trigger
	filter  Filterer

	ctx *flowContext
}

type FlowBuilder struct {
	flow *Flow
}

func (bot *Bot) RegisterFlow(f *Flow) error {
	if _, exists := bot.registeredFlows[f.name]; exists {
		return errors.New("flow already registered")
	}
	bot.registeredFlows[f.name] = f
	bot.registeredFlows[f.name].bot = bot
	return nil
}

func (bot *Bot) findFlow(ev *slack.MessageEvent) *Flow {
	if f, ok := bot.activeFlows[ev.User]; ok {
		return f
	}

	for _, f := range bot.registeredFlows {
		if f.filter.filter(&ev.Msg) && f.trigger(bot, &ev.Msg) {
			nf := f.dup()

			bot.activeFlows[ev.User] = nf
			return nf
		}
	}

	return nil
}

func NewFlow(name string) *FlowBuilder {
	return NewFlowWithContext(name, nil)
}

func NewFlowWithContext(name string, ctx interface{}) *FlowBuilder {
	return &FlowBuilder{
		flow: &Flow{
			name:        name,
			userCtxTmpl: ctx,
		},
	}
}

func NewState(name string, act FlowAction) *StateBuilder {
	return &StateBuilder{
		state: &State{
			name: name,
			act:  act,
		},
	}
}

func (fb *FlowBuilder) AddStates(states ...*State) *FlowBuilder {
	for _, state := range states {
		if _, exists := fb.flow.states[state.name]; !exists {
			fb.flow.states[state.name] = state
		}
	}
	return fb
}

func (fb *FlowBuilder) SetTrigger(t Trigger) *FlowBuilder {
	fb.flow.trigger = t
	return fb
}

func (fb *FlowBuilder) FilterBy(f Filterer) *FlowBuilder {
	fb.flow.filter = f
	return fb
}

func (fb *FlowBuilder) Build(initialState string) *Flow {
	validInitialState := false

	for _, s := range fb.flow.states {
		if s.name == initialState {
			validInitialState = true
		}
	}
	if !validInitialState {
		return nil
	}

	fb.flow.initialState = fb.flow.states[initialState]

	return fb.flow
}

func (f *Flow) step(ev *slack.MessageEvent) {
	if f.ctx.currentState.act != nil {
		handled := f.ctx.currentState.act(f.bot, &ev.Msg, f.ctx.userCtx)

		if handled {
			f.ctx.currentState = f.states[f.ctx.currentState.dst]
		}
	}
}

func (f *Flow) dup() *Flow {
	nf := f

	nf.ctx = &flowContext{
		currentState: f.initialState,
		userCtx:      reflect.New(reflect.TypeOf(nf.userCtxTmpl)).Interface(),
	}

	return nf
}
