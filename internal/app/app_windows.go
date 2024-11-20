package app

import "context"

func (a *App) Startup(ctx context.Context) {
	runShutdownOnWmEndsession(ctx)
	a.commonStartup(ctx)
}
