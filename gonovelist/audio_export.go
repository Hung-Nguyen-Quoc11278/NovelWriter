package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// VietnameseVoicePreset định nghĩa thông số một mẫu giọng đọc Neural Tiếng Việt trong Edge-TTS.
type VietnameseVoicePreset struct {
	ID          string // Mã định danh kỹ thuật (VD: "vi-VN-HoaiMyNeural")
	Label       string // Tên hiển thị Tiếng Việt trên giao diện (VD: "Hoài Mỹ (Nữ - Miền Nam)")
	Gender      string // Giới tính (Nữ / Nam)
	Description string // Mô tả đặc trưng giọng đọc
}

// Danh sách các giọng đọc Tiếng Việt chất lượng cao của Edge-TTS
var VietnameseVoicePresets = []VietnameseVoicePreset{
	{
		ID:          "vi-VN-HoaiMyNeural",
		Label:       "Hoài Mỹ (Nữ - Miền Nam)",
		Gender:      "Nữ",
		Description: "Giọng nữ miền Nam dịu dàng, tự nhiên, giàu cảm xúc — rất thích hợp cho tiểu thuyết tình cảm & tự sự.",
	},
	{
		ID:          "vi-VN-NamMinhNeural",
		Label:       "Nam Minh (Nam - Miền Nam)",
		Gender:      "Nam",
		Description: "Giọng nam miền Nam ấm áp, truyền cảm, rõ chữ — rất thích hợp cho tiểu thuyết lịch sử & phiêu lưu.",
	},
}

// AllVietnameseVoiceLabels trả về danh sách các nhãn hiển thị cho widget.Select.
func AllVietnameseVoiceLabels() []string {
	labels := make([]string, len(VietnameseVoicePresets))
	for i, v := range VietnameseVoicePresets {
		labels[i] = v.Label
	}
	return labels
}

// ResolveVoiceIDFromLabel ánh xạ từ nhãn Tiếng Việt sang Voice ID của Edge-TTS.
func ResolveVoiceIDFromLabel(label string) string {
	for _, v := range VietnameseVoicePresets {
		if v.Label == label || v.ID == label {
			return v.ID
		}
	}
	return "vi-VN-HoaiMyNeural"
}

// ResolveVoiceDescriptionFromLabel trả về mô tả của giọng đọc đang chọn.
func ResolveVoiceDescriptionFromLabel(label string) string {
	for _, v := range VietnameseVoicePresets {
		if v.Label == label || v.ID == label {
			return v.Description
		}
	}
	return ""
}

// CleanProseForSpeech loại bỏ các ký tự định dạng thô (Markdown, thẻ HTML, dấu ngắt cảnh)
// để Edge-TTS đọc tự nhiên và liền mạch như một người kể chuyện thực thụ.
func CleanProseForSpeech(raw string) string {
	text := raw
	// Xóa thẻ HTML nếu có
	reHTML := regexp.MustCompile(`<[^>]*>`)
	text = reHTML.ReplaceAllString(text, "")

	// Xóa định dạng in đậm / in nghiêng markdown: **text** hoặc *text*
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "_", "")

	// Thay thế các dòng ngắt cảnh * * * hoặc ― bằng dấu ngắt câu nhẹ
	reDivider := regexp.MustCompile(`(?m)^\s*(\*\s*\*\s*\*|―+|--+)\s*$`)
	text = reDivider.ReplaceAllString(text, "\n\n")

	// Xóa ký tự trích dẫn > ở đầu dòng
	reQuote := regexp.MustCompile(`(?m)^\s*>\s*`)
	text = reQuote.ReplaceAllString(text, "")

	// Rút gọn các dòng trống liên tiếp
	reMultiLines := regexp.MustCompile(`\n{3,}`)
	text = reMultiLines.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// ExportTextToAudioWithEdgeTTS chạy tiện ích dòng lệnh edge-tts để tạo tệp âm thanh MP3.
// Cú pháp: edge-tts --voice <selected_voice> --text "<content>" --write-media <output_path.mp3>
func ExportTextToAudioWithEdgeTTS(voiceID, textContent, outputPath string) error {
	cleanText := strings.TrimSpace(textContent)
	if cleanText == "" {
		return fmt.Errorf("nội dung văn bản để xuất âm thanh không được để trống")
	}
	if voiceID == "" {
		voiceID = "vi-VN-HoaiMyNeural"
	}
	if outputPath == "" {
		return fmt.Errorf("đường dẫn tệp âm thanh đích không được để trống")
	}

	if !strings.HasSuffix(strings.ToLower(outputPath), ".mp3") {
		outputPath += ".mp3"
	}

	outDir := filepath.Dir(outputPath)
	if outDir != "" && outDir != "." {
		_ = os.MkdirAll(outDir, 0755)
	}

	edgeTTSPath, err := findEdgeTTSExecutable()
	if err != nil {
		return fmt.Errorf("không tìm thấy công cụ dòng lệnh 'edge-tts' trên hệ thống.\n\nVui lòng cài đặt bằng lệnh Terminal / CMD:\n  pip install edge-tts\n(và đảm bảo Python Scripts đã có trong PATH)")
	}

	// Nếu văn bản dài (> 800 ký tự), lưu vào tệp văn bản tạm thời rồi truyền --file
	// để tránh lỗi vượt quá giới hạn độ dài tham số dòng lệnh (ARG_MAX) của hệ điều hành.
	var cmd *exec.Cmd
	if len(cleanText) > 800 {
		tmpFile, err := os.CreateTemp("", "gonovelist_tts_*.txt")
		if err != nil {
			cmd = exec.Command(edgeTTSPath, "--voice", voiceID, "--text", cleanText, "--write-media", outputPath)
		} else {
			defer os.Remove(tmpFile.Name())
			_, _ = tmpFile.WriteString(cleanText)
			_ = tmpFile.Close()
			cmd = exec.Command(edgeTTSPath, "--voice", voiceID, "--file", tmpFile.Name(), "--write-media", outputPath)
		}
	} else {
		cmd = exec.Command(edgeTTSPath, "--voice", voiceID, "--text", cleanText, "--write-media", outputPath)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())
		if errStr != "" {
			return fmt.Errorf("lỗi từ edge-tts: %s", errStr)
		}
		return fmt.Errorf("lỗi khi xuất tệp audio: %w", err)
	}

	fi, err := os.Stat(outputPath)
	if err != nil || fi.Size() == 0 {
		return fmt.Errorf("tệp âm thanh chưa được tạo hoặc dung lượng bằng 0 byte")
	}

	return nil
}

func findEdgeTTSExecutable() (string, error) {
	// 1. Kiểm tra PATH thông thường
	if p, err := exec.LookPath("edge-tts"); err == nil {
		return p, nil
	}
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("edge-tts.exe"); err == nil {
			return p, nil
		}
	}

	// 2. Tìm trong các thư mục cài đặt Python Scripts mặc định
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", "edge-tts"),
		filepath.Join(home, "Library", "Python", "3.9", "bin", "edge-tts"),
		filepath.Join(home, "Library", "Python", "3.10", "bin", "edge-tts"),
		filepath.Join(home, "Library", "Python", "3.11", "bin", "edge-tts"),
		filepath.Join(home, "Library", "Python", "3.12", "bin", "edge-tts"),
		filepath.Join(home, "AppData", "Roaming", "Python", "Python310", "Scripts", "edge-tts.exe"),
		filepath.Join(home, "AppData", "Roaming", "Python", "Python311", "Scripts", "edge-tts.exe"),
		filepath.Join(home, "AppData", "Roaming", "Python", "Python312", "Scripts", "edge-tts.exe"),
		filepath.Join(home, "AppData", "Local", "Programs", "Python", "Python310", "Scripts", "edge-tts.exe"),
		filepath.Join(home, "AppData", "Local", "Programs", "Python", "Python311", "Scripts", "edge-tts.exe"),
		filepath.Join(home, "AppData", "Local", "Programs", "Python", "Python312", "Scripts", "edge-tts.exe"),
	}

	for _, cand := range candidates {
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return cand, nil
		}
	}

	return "", fmt.Errorf("not found")
}

// ShowAudioExportDialog mở hộp thoại trực quan để chọn giọng đọc Neural Tiếng Việt,
// tinh chỉnh văn bản, chọn nơi lưu và xuất file MP3 qua Edge-TTS.
func (ui *NovelistUI) ShowAudioExportDialog() {
	if ui.window == nil {
		return
	}

	// 1. Thu thập văn bản mặc định từ Cảnh đang mở hoặc toàn bộ tác phẩm
	defaultContent := ""
	defaultTitle := ui.activeProject.Title
	if ui.editorPanel != nil && ui.editorPanel.activeScene != nil {
		defaultContent = ui.editorPanel.proseEntry.Text
		if defaultContent == "" {
			defaultContent = ui.editorPanel.activeScene.Content
		}
		defaultTitle = fmt.Sprintf("%s_%s", ui.activeProject.Title, ui.editorPanel.activeScene.Title)
	} else {
		// Nạp cảnh đầu tiên có nội dung
		allActs, _ := ui.store.ListActs(ui.activeProject.ID)
		for _, a := range allActs {
			chaps, _ := ui.store.ListChapters(a.ID)
			for _, ch := range chaps {
				scenes, _ := ui.store.ListScenes(ch.ID)
				for _, sc := range scenes {
					if strings.TrimSpace(sc.Content) != "" {
						defaultContent = sc.Content
						defaultTitle = fmt.Sprintf("%s_%s", ui.activeProject.Title, sc.Title)
						break
					}
				}
				if defaultContent != "" {
					break
				}
			}
			if defaultContent != "" {
				break
			}
		}
	}

	cleanedInitialText := CleanProseForSpeech(defaultContent)

	// 2. Các widget giao diện
	voiceLabels := AllVietnameseVoiceLabels()
	selectedVoiceID := VietnameseVoicePresets[0].ID
	voiceSelect := widget.NewSelect(voiceLabels, nil)
	voiceSelect.SetSelected(VietnameseVoicePresets[0].Label)

	voiceDescLabel := widget.NewLabel(VietnameseVoicePresets[0].Description)
	voiceDescLabel.Wrapping = fyne.TextWrapWord
	voiceDescLabel.TextStyle = fyne.TextStyle{Italic: true}

	voiceSelect.OnChanged = func(chosen string) {
		selectedVoiceID = ResolveVoiceIDFromLabel(chosen)
		voiceDescLabel.SetText(ResolveVoiceDescriptionFromLabel(chosen))
	}

	// Khung soạn thảo / xem trước văn bản cần đọc
	contentEntry := NewVietMultiLineEntry()
	contentEntry.SetPlaceHolder("Nhập hoặc dán văn bản cần đọc thành tiếng...")
	contentEntry.SetText(cleanedInitialText)
	contentEntry.SetMinRowsVisible(7)

	statsLabel := widget.NewLabel("")
	updateStats := func(text string) {
		words := len(strings.Fields(text))
		chars := utf8.RuneCountInString(text)
		statsLabel.SetText(fmt.Sprintf("📊 Độ dài: %d từ  •  %d ký tự (khoảng %.1f phút nghe)",
			words, chars, float64(words)/160.0))
	}
	updateStats(cleanedInitialText)

	contentEntry.SetOnChangedCallback(func(text string) {
		updateStats(text)
	})

	// Đường dẫn lưu file MP3 đích
	suggestedFileName := sanitizeFileName(defaultTitle) + ".mp3"
	pathEntry := NewVietEntry()
	pathEntry.SetText(suggestedFileName)

	browseBtn := widget.NewButtonWithIcon("Chọn nơi lưu...", theme.FolderOpenIcon(), func() {
		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, ui.window)
				return
			}
			if writer == nil {
				return
			}
			p := writer.URI().Path()
			_ = writer.Close()
			if !strings.HasSuffix(strings.ToLower(p), ".mp3") {
				p += ".mp3"
			}
			pathEntry.SetText(p)
		}, ui.window)
		fd.SetFileName(filepath.Base(pathEntry.Text))
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".mp3"}))
		fd.Show()
	})

	// Trạng thái và thanh tiến trình (ẩn ban đầu)
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Hide()

	statusLabel := widget.NewLabelWithStyle("Sẵn sàng xuất audio bằng giọng đọc Neural Tiếng Việt.", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	// Nút nạp lại nội dung cảnh hiện tại
	reloadSceneBtn := widget.NewButtonWithIcon("Nạp lại cảnh hiện tại", theme.ViewRefreshIcon(), func() {
		if ui.editorPanel != nil && ui.editorPanel.activeScene != nil {
			txt := CleanProseForSpeech(ui.editorPanel.proseEntry.Text)
			contentEntry.SetText(txt)
			updateStats(txt)
		}
	})
	reloadSceneBtn.Importance = widget.LowImportance

	// 3. Xây dựng bố cục Dialog
	headerBox := container.NewVBox(
		widget.NewLabelWithStyle("🎧 XUẤT BẢN FILE AUDIO (MP3) — GIỌNG ĐỌC AI TIẾNG VIỆT", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Sử dụng công nghệ Neural Text-to-Speech miễn phí từ Edge-TTS với giọng đọc tự nhiên, chuẩn sắc thái Tiếng Việt."),
		widget.NewSeparator(),
	)

	voiceCard := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabelWithStyle("Chọn giọng đọc:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, voiceSelect),
		voiceDescLabel,
	)

	contentHeader := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("Nội dung văn bản để đọc thành tiếng:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		reloadSceneBtn,
	)

	pathCard := container.NewVBox(
		widget.NewLabelWithStyle("Đường dẫn lưu tệp MP3:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, browseBtn, pathEntry),
	)

	var audioDialog dialog.Dialog

	cancelBtn := widget.NewButton("Đóng", func() {
		if audioDialog != nil {
			audioDialog.Hide()
		}
	})

	exportBtn := widget.NewButtonWithIcon("Xuất file Audio", theme.MediaPlayIcon(), nil)
	exportBtn.Importance = widget.HighImportance

	exportBtn.OnTapped = func() {
		textToRead := strings.TrimSpace(contentEntry.Text)
		if textToRead == "" {
			dialog.ShowInformation("Thông báo", "Vui lòng nhập nội dung văn bản để chuyển đổi sang âm thanh.", ui.window)
			return
		}

		outPath := strings.TrimSpace(pathEntry.Text)
		if outPath == "" {
			outPath = suggestedFileName
		}
		if !strings.HasSuffix(strings.ToLower(outPath), ".mp3") {
			outPath += ".mp3"
		}

		// Khóa giao diện và bắt đầu chạy ngầm
		exportBtn.Disable()
		cancelBtn.Disable()
		progressBar.Show()
		statusLabel.SetText("⏳ Đang kết nối Edge-TTS và tổng hợp âm thanh MP3...")

		voiceChosen := selectedVoiceID
		go func() {
			err := ExportTextToAudioWithEdgeTTS(voiceChosen, textToRead, outPath)

			// Cập nhật giao diện trên Main Thread của Fyne
			fyne.Do(func() {
				exportBtn.Enable()
				cancelBtn.Enable()
				progressBar.Hide()

				if err != nil {
					statusLabel.SetText("❌ Có lỗi xảy ra khi tạo audio.")
					dialog.ShowError(err, ui.window)
					return
				}

				PlayUIClickSound()
				statusLabel.SetText("✅ Xuất audio thành công!")

				fi, _ := os.Stat(outPath)
				fileSizeMB := float64(0)
				if fi != nil {
					fileSizeMB = float64(fi.Size()) / (1024 * 1024)
				}

				if ui.statusFooter != nil {
					ui.statusFooter.SetText(fmt.Sprintf("Đã xuất audio: %s (%.2f MB)", filepath.Base(outPath), fileSizeMB))
				}

				dialog.ShowInformation(
					"Xuất Audio Thành Công!",
					fmt.Sprintf("Đã xuất file âm thanh thành công!\n\n• Tệp đích: %s\n• Giọng đọc: %s\n• Dung lượng: %.2f MB\n• Số từ: %d từ",
						outPath,
						voiceSelect.Selected,
						fileSizeMB,
						len(strings.Fields(textToRead)),
					),
					ui.window,
				)
			})
		}()
	}

	actionsRow := container.NewHBox(
		cancelBtn,
		layout.NewSpacer(),
		exportBtn,
	)

	dialogBody := container.NewVScroll(container.NewVBox(
		headerBox,
		voiceCard,
		widget.NewSeparator(),
		contentHeader,
		contentEntry,
		statsLabel,
		widget.NewSeparator(),
		pathCard,
		widget.NewSeparator(),
		progressBar,
		statusLabel,
		widget.NewSeparator(),
		actionsRow,
	))

	audioDialog = dialog.NewCustomWithoutButtons("Xuất Bản Audio (Edge-TTS)", dialogBody, ui.window)
	audioDialog.Resize(fyne.NewSize(760, 640))
	audioDialog.Show()
}
