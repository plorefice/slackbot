package slackbot

import (
	"errors"
	"reflect"

	"github.com/nlopes/slack"
)

// FlowAction is the state callback. It is called whenever a new message
// is received, and contains the user context. Return true to perform a
// transition to the next state.
type FlowAction func(*Bot, *slack.Msg, interface{}) bool

// Guard is called to check if the associated flow can be activated. It is
// called after all the filters. Return true to enable the flow.
type Guard func(*Bot, *slack.Msg) bool

// State represents a specific point in the execution of a flow.
type State struct {
	name string
	act  FlowAction
	dst  string
}

// StateBuilder implements the builder pattern for State objects.
type StateBuilder struct {
	state *State
}

// To defines the state to which the flow should transition if the state callback
// for this state allows it.
func (sb *StateBuilder) To(state string) *StateBuilder {
	sb.state.dst = state
	return sb
}

// Build assembles a new State.
func (sb *StateBuilder) Build() *State {
	return &State{
		name: sb.state.name,
		act:  sb.state.act,
		dst:  sb.state.dst,
	}
}

// flowContext contains all those intstance-specific parts of a flow. It is
// the only part of the flow that must be duped when allocating a new flow instance.
type flowContext struct {
	currentState *State
	userCtx      interface{}
}

// Flow is the building block of a flow-based Slackbot. It describes the sequence
// of states the bot should go through in order to complete a certain task.
// A flow is described by a well-defined sequence of states, a filter to ignore
// unwanted messages, and a guard that acts as a conditional access to the flow.
// Each instance of the flow has an associated user context that will be shared
// among all the states belonging of the same flow.
type Flow struct {
	bot *Bot

	name         string
	states       map[string]*State
	initialState *State
	userCtxTmpl  reflect.Type

	guard  Guard
	filter Filterer

	ctx *flowContext
}

// FlowBuilder implements the builder pattern for Flow objects.
type FlowBuilder struct {
	flow *Flow
}

// RegisterFlow associates a flow with a bot. After registration, each new
// non-filtered message will trigger the flow guard, if no other flows are active.
func (bot *Bot) RegisterFlow(f *Flow) error {
	if _, exists := bot.registeredFlows[f.name]; exists {
		return errors.New("flow already registered")
	}
	bot.registeredFlows[f.name] = f
	bot.registeredFlows[f.name].bot = bot
	return nil
}

func (bot *Bot) findFlow(ev *slack.MessageEvent) *Flow {
	if f, ok := bot.activeFlows[ev.User]; ok && f != nil {
		return f
	}

	for _, f := range bot.registeredFlows {
		if f.filter.filter(&ev.Msg) && f.guard(bot, &ev.Msg) {
			nf := f.dup()

			bot.activeFlows[ev.User] = nf
			return nf
		}
	}

	return nil
}

// NewFlow creates a new flow with an empty user context.
func NewFlow(name string) *FlowBuilder {
	return NewFlowWithContext(name, nil)
}

// NewFlowWithContext creates a new flow using ctx as the type template for
// allocating each flow instance's user context.
func NewFlowWithContext(name string, ctx interface{}) *FlowBuilder {
	return &FlowBuilder{
		flow: &Flow{
			name:        name,
			states:      make(map[string]*State),
			userCtxTmpl: reflect.Indirect(reflect.ValueOf(ctx)).Type(),
		},
	}
}

// NewState creates a new state with an associated state action.
func NewState(name string, act FlowAction) *StateBuilder {
	return &StateBuilder{
		state: &State{
			name: name,
			act:  act,
		},
	}
}

// AddStates associates the given states to the flow.
func (fb *FlowBuilder) AddStates(states ...*State) *FlowBuilder {
	for _, state := range states {
		if _, exists := fb.flow.states[state.name]; !exists {
			fb.flow.states[state.name] = state
		}
	}
	return fb
}

// SetGuard associates the given guard to the flow.
func (fb *FlowBuilder) SetGuard(g Guard) *FlowBuilder {
	fb.flow.guard = g
	return fb
}

// FilterBy associates the given filter to the flow.
func (fb *FlowBuilder) FilterBy(f Filterer) *FlowBuilder {
	fb.flow.filter = f
	return fb
}

// Build assembles a new Flow.
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
	cs := f.ctx.currentState

	if cs.act != nil {
		if handled := cs.act(f.bot, &ev.Msg, f.ctx.userCtx); !handled {
			return
		}

		if ds, ok := f.states[cs.dst]; !ok {
			f.bot.activeFlows[ev.User] = nil
		} else {
			f.ctx.currentState = ds
		}
	}
}

func (f *Flow) dup() *Flow {
	nf := f
	nf.ctx = new(flowContext)

	nf.ctx.currentState = f.initialState

	if nf.userCtxTmpl != nil {
		nf.ctx.userCtx = reflect.New(nf.userCtxTmpl).Interface()
	}

	return nf
}
