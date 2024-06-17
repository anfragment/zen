package app

import (
	"context"

	"github.com/anfragment/zen/internal/rule"
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

const filterChannel = "filter:action"

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
