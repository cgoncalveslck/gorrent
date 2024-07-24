package main

import (
	"context"
	"encoding/json"
	"gorrent/backend"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx   context.Context
	State *backend.State
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// ,so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetStateJSON() string {
	jason, err := json.Marshal(a.State)
	if err != nil {
		panic(err)
	}

	return string(jason)
}

func (a *App) OpenFileDialog() {
	options := runtime.OpenDialogOptions{
		Title: "Open File",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Torrent Files",
				Pattern:     "*.torrent",
			},
		},
	}

	path, err := runtime.OpenFileDialog(a.ctx, options)
	if err != nil {
		panic(err)
	}

	_, err = backend.HandleFile(a.ctx, path)
	if err != nil {
		panic(err)
	}

	torrents := make([]backend.Torrent, 0)
	torrents = append(torrents, backend.Torrent{
		Name: "test",
	})

	s := &backend.State{
		Torrents: torrents,
	}

	a.State = s
	a.State.Write()
}
