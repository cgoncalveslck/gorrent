package main

import (
	"context"
	"fmt"
	"gorrent/backend"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/exp/rand"
)

// App struct
type App struct {
	ctx    context.Context
	Client *backend.Client
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// ,so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	backend.InitDB()
}

func (a *App) OpenFileDialog() *backend.Torrent {
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

	torrent, err := backend.HandleFile(a.ctx, path)
	if err != nil {
		panic(err)
	}

	backend.NewClient(torrent)
	return torrent
}

func (a *App) GetDevTorrent() (*backend.Torrent, error) {
	files, err := os.ReadDir("_dev")
	if err != nil {
		return nil, err
	}

	var torrentFiles []string
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".torrent" {
			torrentFiles = append(torrentFiles, filepath.Join("_dev", file.Name()))
		}
	}

	if len(torrentFiles) == 0 {
		return nil, fmt.Errorf("no torrent files found in _dev directory")
	}

	rand.Seed(uint64(time.Now().UnixNano()))
	selectedFile := torrentFiles[rand.Intn(len(torrentFiles))]

	return backend.HandleFile(context.Background(), selectedFile)
}
