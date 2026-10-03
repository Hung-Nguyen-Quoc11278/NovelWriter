package main

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestAudioExportSuccessStatusDoesNotExpandSidebarMinSize(t *testing.T) {
	test.NewTempApp(t)
	status := widget.NewLabel("Tổng số từ: 0")
	tree := container.NewGridWrap(fyne.NewSize(345, 24), widget.NewLabel("Cây tác phẩm"))
	sidebar := container.NewBorder(nil, container.NewVBox(widget.NewSeparator(), status), nil, nil, tree)
	initialSidebarMinWidth := sidebar.MinSize().Width
	status.SetText("Đã xuất audio: Bản Đồ Thủy Tinh Thành Hội An_Toan_Bo_Tac_Pham.mp3 (0.49 MB)")
	longStatusMinWidth := sidebar.MinSize().Width
	if longStatusMinWidth <= initialSidebarMinWidth {
		t.Fatalf("fixture không tái hiện việc footer dài làm nở sidebar: ban đầu %.2f, sau %.2f", initialSidebarMinWidth, longStatusMinWidth)
	}

	status.SetText(audioExportSuccessFooterText)
	if got := sidebar.MinSize().Width; got != initialSidebarMinWidth {
		t.Fatalf("status export làm tăng MinSize sidebar: trước %.2f, sau %.2f", initialSidebarMinWidth, got)
	}
}
