package main

import (
	"context"
)

// App owns native desktop lifecycle state. Business bindings are added by
// their dedicated implementation tasks.
type App struct {
	ctx context.Context
}

// NewApp creates the desktop application.
func NewApp() *App {
	return &App{}
}

// startup stores the Wails lifecycle context for future native operations.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}
