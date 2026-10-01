package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	novelistApp := app.NewWithID("io.gonovelist.desktop")
	mainWindow := novelistApp.NewWindow("GoNovelist — Cross-Platform Novel Writing Studio")
	mainWindow.Resize(fyne.NewSize(1360, 860))
	mainWindow.CenterOnScreen()

	dbPath := resolveDatabasePath()
	store, err := NewStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize GoNovelist SQLite store at %s: %v", dbPath, err)
	}
	defer func() {
		_ = store.Close()
	}()

	ui, err := NewNovelistUI(novelistApp, mainWindow, store)
	if err != nil {
		log.Fatalf("Failed to build GoNovelist Fyne UI: %v", err)
	}

	mainWindow.SetOnClosed(func() {
		ui.editorPanel.FlushPendingSave()
	})

	mainWindow.ShowAndRun()
}

func resolveDatabasePath() string {
	if custom := os.Getenv("GONOVELIST_DB"); custom != "" {
		return custom
	}
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "gonovelist.db"
	}
	appDir := filepath.Join(cfgDir, "GoNovelist")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "gonovelist.db"
	}
	return filepath.Join(appDir, "gonovelist.db")
}
