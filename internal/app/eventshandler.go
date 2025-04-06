package app

import (
	"context"
	"fmt"

	"github.com/ZenPrivacy/zen-desktop/internal/networkrules/rule"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type eventsHandler struct {
	ctx context.Context
}

func newEventsHandler(ctx context.Context) *eventsHandler {
	return &eventsHandler{ctx: ctx}
}

type filterActionKind string

const (
	filterChannel                         = "filter:action"
	filterActionBlock    filterActionKind = "block"
	filterActionRedirect filterActionKind = "redirect"
	filterActionModify   filterActionKind = "modify"
)

type filterAction struct {
	Kind    filterActionKind `json:"kind"`
	Method  string           `json:"method"`
	URL     string           `json:"url"`
	To      string           `json:"to,omitempty"`
	Referer string           `json:"referer,omitempty"`
	Rules   []rule.Rule      `json:"rules"`
}

type proxyState string

// Only these states are handled via eventsHandler because the proxy can only start without any input from the user.
const (
	proxyChannel               = "proxy:action"
	proxyStarting   proxyState = "starting"
	proxyStarted    proxyState = "started"
	proxyStartError proxyState = "startError"
	proxyStopping   proxyState = "stopping"
	proxyStopped    proxyState = "stopped"
	proxyStopError  proxyState = "stopError"
	unsupportedDE   proxyState = "unsupportedDE"
)

type proxyAction struct {
	Kind  proxyState `json:"kind"`
	Error string     `json:"error"`
}

func (e *eventsHandler) OnFilterBlock(method, url, referer string, rules []rule.Rule) {
	runtime.EventsEmit(e.ctx, filterChannel, filterAction{
		Kind:    filterActionBlock,
		Method:  method,
		URL:     url,
		Referer: referer,
		Rules:   rules,
	})
}

func (e *eventsHandler) OnFilterRedirect(method, url, to, referer string, rules []rule.Rule) {
	runtime.EventsEmit(e.ctx, filterChannel, filterAction{
		Kind:    filterActionRedirect,
		Method:  method,
		URL:     url,
		To:      to,
		Referer: referer,
		Rules:   rules,
	})
}

func (e *eventsHandler) OnFilterModify(method, url, referer string, rules []rule.Rule) {
	runtime.EventsEmit(e.ctx, filterChannel, filterAction{
		Kind:    filterActionModify,
		Method:  method,
		URL:     url,
		Referer: referer,
		Rules:   rules,
	})
}

func (e *eventsHandler) OnProxyStarting() {
	runtime.EventsEmit(e.ctx, proxyChannel, proxyAction{
		Kind: proxyStarting,
	})
}

func (e *eventsHandler) OnProxyStarted() {
	runtime.EventsEmit(e.ctx, proxyChannel, proxyAction{
		Kind: proxyStarted,
	})
}

func (e *eventsHandler) OnProxyStartError(err error) {
	runtime.EventsEmit(e.ctx, proxyChannel, proxyAction{
		Kind:  proxyStartError,
		Error: fmt.Sprint(err),
	})
}

func (e *eventsHandler) OnProxyStopping() {
	runtime.EventsEmit(e.ctx, proxyChannel, proxyAction{
		Kind: proxyStopping,
	})
}

func (e *eventsHandler) OnProxyStopped() {
	runtime.EventsEmit(e.ctx, proxyChannel, proxyAction{
		Kind: proxyStopped,
	})
}

func (e *eventsHandler) OnProxyStopError(err error) {
	runtime.EventsEmit(e.ctx, proxyChannel, proxyAction{
		Kind:  proxyStopError,
		Error: fmt.Sprint(err),
	})
}

func (e *eventsHandler) OnUnsupportedDE(err error) {
	runtime.EventsEmit(e.ctx, proxyChannel, proxyAction{
		Kind:  unsupportedDE,
		Error: fmt.Sprint(err),
	})
}
