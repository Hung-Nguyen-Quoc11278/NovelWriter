package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	novelistApp := app.NewWithID("io.gonovelist.desktop.vi")
	mainWindow := novelistApp.NewWindow("GoNovelist — Phần Mềm Sáng Tác Tiểu Thuyết")
	mainWindow.Resize(fyne.NewSize(1360, 840))
	mainWindow.CenterOnScreen()

	// Lưu cơ sở dữ liệu SQLite trong thư mục người dùng (~/.gonovelist/gonovelist.db)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	dataDir := filepath.Join(homeDir, ".gonovelist")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("không thể tạo thư mục dữ liệu: %v", err)
	}
	dbPath := filepath.Join(dataDir, "gonovelist.db")

	store, err := NewStore(dbPath)
	if err != nil {
		log.Fatalf("khởi tạo cơ sở dữ liệu thất bại: %v", err)
	}
	defer func() {
		_ = store.Close()
	}()

	ui, err := NewNovelistUI(novelistApp, mainWindow, store)
	if err != nil {
		log.Fatalf("khởi tạo giao diện thất bại: %v", err)
	}

	// Đảm bảo lưu mọi thay đổi đang chờ trong bộ đệm trước khi đóng cửa sổ
	mainWindow.SetCloseIntercept(func() {
		ui.editorPanel.FlushPendingSave()
		mainWindow.Close()
	})

	mainWindow.ShowAndRun()
}
