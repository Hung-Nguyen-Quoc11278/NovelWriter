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
	"fyne.io/fyne/v2/layout"
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
	{
		ID:          "vi-VN-HoangMaiNeural",
		Label:       "Hoàng Mai (Nữ - Miền Bắc)",
		Gender:      "Nữ",
		Description: "Giọng nữ Hà Nội chuẩn mực, thanh lịch, phát âm tròn vành rõ chữ — rất thích hợp cho truyện văn học kinh điển & ký sự.",
	},
	{
		ID:          "vi-VN-NamKhanhNeural",
		Label:       "Nam Khánh (Nam - Miền Bắc)",
		Gender:      "Nam",
		Description: "Giọng nam miền Bắc đĩnh đạc, đầm ấm, quyền uy và cuốn hút — thích hợp cho truyện kỳ ảo, hành động & trinh thám.",
	},
	{
		ID:          "vi-VN-ThuTrangNeural",
		Label:       "Thu Trang (Nữ - Miền Trung)",
		Gender:      "Nữ",
		Description: "Giọng nữ miền Trung nhẹ nhàng, mộc mạc, tha thiết đậm chất thơ — thích hợp cho truyện đồng quê, hồi ức & chiêm nghiệm.",
	},
}

// AudioExportScope định nghĩa phạm vi phân cấp nội dung cần xuất sang âm thanh.
type AudioExportScope string

const (
	AudioScopeCurrentScene   AudioExportScope = "Cảnh hiện tại"
	AudioScopeCurrentChapter AudioExportScope = "Chương hiện tại"
	AudioScopeCurrentAct     AudioExportScope = "Hồi hiện tại"
	AudioScopeFullNovel      AudioExportScope = "Toàn bộ tác phẩm"
)

// AllAudioExportScopes trả về danh sách các phạm vi xuất bản audio cho widget.Select.
var AllAudioExportScopes = []string{
	string(AudioScopeCurrentScene),
	string(AudioScopeCurrentChapter),
	string(AudioScopeCurrentAct),
	string(AudioScopeFullNovel),
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

// resolveCurrentSceneChapterAct xác định ngữ cảnh thực tế của Cảnh, Chương và Hồi đang được chọn.
func (ui *NovelistUI) resolveCurrentSceneChapterAct() (*Scene, *Chapter, *Act) {
	var curScene *Scene
	var curChapter *Chapter
	var curAct *Act

	// 1. Thử lấy từ editorPanel nếu đang mở cảnh
	if ui.editorPanel != nil && ui.editorPanel.activeScene != nil {
		curScene = ui.editorPanel.activeScene
		if curScene.ChapterID > 0 {
			curChapter, _ = ui.store.GetChapter(curScene.ChapterID)
			if curChapter != nil && curChapter.ActID > 0 {
				curAct, _ = ui.store.GetAct(curChapter.ActID)
			}
		}
	}

	// 2. Nếu chưa có cảnh hoặc đang chọn nút khác trên cây phân cấp
	if curScene == nil && ui.selectedUID != "" {
		kind, id, err := ParseNodeUID(ui.selectedUID)
		if err == nil {
			switch kind {
			case "scene":
				curScene, _ = ui.store.GetScene(id)
				if curScene != nil && curScene.ChapterID > 0 {
					curChapter, _ = ui.store.GetChapter(curScene.ChapterID)
					if curChapter != nil && curChapter.ActID > 0 {
						curAct, _ = ui.store.GetAct(curChapter.ActID)
					}
				}
			case "chapter":
				curChapter, _ = ui.store.GetChapter(id)
				if curChapter != nil {
					if curChapter.ActID > 0 {
						curAct, _ = ui.store.GetAct(curChapter.ActID)
					}
					scenes, _ := ui.store.ListScenes(curChapter.ID)
					if len(scenes) > 0 {
						curScene = &scenes[0]
					}
				}
			case "act":
				curAct, _ = ui.store.GetAct(id)
				if curAct != nil {
					chaps, _ := ui.store.ListChapters(curAct.ID)
					if len(chaps) > 0 {
						curChapter = &chaps[0]
						scenes, _ := ui.store.ListScenes(curChapter.ID)
						if len(scenes) > 0 {
							curScene = &scenes[0]
						}
					}
				}
			}
		}
	}

	// 3. Dự phòng nếu chưa có gì được chọn: lấy Hồi, Chương và Cảnh đầu tiên của tác phẩm
	if curAct == nil {
		allActs, _ := ui.store.ListActs(ui.activeProject.ID)
		if len(allActs) > 0 {
			curAct = &allActs[0]
			chaps, _ := ui.store.ListChapters(curAct.ID)
			if len(chaps) > 0 {
				curChapter = &chaps[0]
				scenes, _ := ui.store.ListScenes(curChapter.ID)
				if len(scenes) > 0 {
					curScene = &scenes[0]
				}
			}
		}
	}

	return curScene, curChapter, curAct
}

// CompileAudioTextForScope duyệt cơ sở dữ liệu SQLite theo phân cấp Hồi -> Chương -> Cảnh
// và tổng hợp nội dung văn bản sạch phù hợp cho giọng đọc Edge-TTS.
func (ui *NovelistUI) CompileAudioTextForScope(scope AudioExportScope) (compiledText string, scopeTitle string, totalScenes int, summaryInfo string) {
	curScene, curChapter, curAct := ui.resolveCurrentSceneChapterAct()
	var sb strings.Builder

	switch scope {
	case AudioScopeCurrentScene:
		scTitle := "Cảnh"
		scContent := ""
		if curScene != nil {
			scTitle = curScene.Title
			scContent = curScene.Content
			if ui.editorPanel != nil && ui.editorPanel.activeScene != nil && ui.editorPanel.activeScene.ID == curScene.ID {
				if ui.editorPanel.proseEntry != nil && strings.TrimSpace(ui.editorPanel.proseEntry.Text) != "" {
					scContent = ui.editorPanel.proseEntry.Text
				}
			}
		}
		sb.WriteString(scTitle + ".\n\n")
		sb.WriteString(CleanProseForSpeech(scContent))
		scopeTitle = fmt.Sprintf("%s_%s", ui.activeProject.Title, scTitle)
		summaryInfo = fmt.Sprintf("Cảnh đơn: %s", scTitle)
		return sb.String(), scopeTitle, 1, summaryInfo

	case AudioScopeCurrentChapter:
		if curChapter == nil {
			return ui.CompileAudioTextForScope(AudioScopeFullNovel)
		}
		scopeTitle = fmt.Sprintf("%s_%s", ui.activeProject.Title, curChapter.Title)
		sb.WriteString(curChapter.Title + ".\n\n")

		scenes, _ := ui.store.ListScenes(curChapter.ID)
		count := 0
		for _, sc := range scenes {
			text := sc.Content
			if ui.editorPanel != nil && ui.editorPanel.activeScene != nil && ui.editorPanel.activeScene.ID == sc.ID {
				if ui.editorPanel.proseEntry != nil && strings.TrimSpace(ui.editorPanel.proseEntry.Text) != "" {
					text = ui.editorPanel.proseEntry.Text
				}
			}
			clean := CleanProseForSpeech(text)
			if clean == "" {
				continue
			}
			count++
			if sc.Title != "" {
				sb.WriteString(sc.Title + ".\n\n")
			}
			sb.WriteString(clean)
			sb.WriteString("\n\n---\n\n")
		}
		summaryInfo = fmt.Sprintf("Chương: %s (%d cảnh)", curChapter.Title, count)
		return sb.String(), scopeTitle, count, summaryInfo

	case AudioScopeCurrentAct:
		if curAct == nil {
			return ui.CompileAudioTextForScope(AudioScopeFullNovel)
		}
		scopeTitle = fmt.Sprintf("%s_%s", ui.activeProject.Title, curAct.Title)
		sb.WriteString(curAct.Title + ".\n\n")

		chaps, _ := ui.store.ListChapters(curAct.ID)
		count := 0
		for _, ch := range chaps {
			sb.WriteString(ch.Title + ".\n\n")
			scenes, _ := ui.store.ListScenes(ch.ID)
			for _, sc := range scenes {
				text := sc.Content
				if ui.editorPanel != nil && ui.editorPanel.activeScene != nil && ui.editorPanel.activeScene.ID == sc.ID {
					if ui.editorPanel.proseEntry != nil && strings.TrimSpace(ui.editorPanel.proseEntry.Text) != "" {
						text = ui.editorPanel.proseEntry.Text
					}
				}
				clean := CleanProseForSpeech(text)
				if clean == "" {
					continue
				}
				count++
				if sc.Title != "" {
					sb.WriteString(sc.Title + ".\n\n")
				}
				sb.WriteString(clean)
				sb.WriteString("\n\n---\n\n")
			}
		}
		summaryInfo = fmt.Sprintf("Hồi: %s (%d chương, %d cảnh)", curAct.Title, len(chaps), count)
		return sb.String(), scopeTitle, count, summaryInfo

	case AudioScopeFullNovel:
		fallthrough
	default:
		scopeTitle = fmt.Sprintf("%s_Toan_Bo_Tac_Pham", ui.activeProject.Title)
		sb.WriteString(ui.activeProject.Title + ".\n")
		if ui.activeProject.Author != "" {
			sb.WriteString("Tác giả: " + ui.activeProject.Author + ".\n")
		}
		if strings.TrimSpace(ui.activeProject.Synopsis) != "" {
			sb.WriteString("Tóm tắt tác phẩm:\n" + CleanProseForSpeech(ui.activeProject.Synopsis) + "\n\n")
		}
		sb.WriteString("\n====================\n\n")

		allActs, _ := ui.store.ListActs(ui.activeProject.ID)
		count := 0
		for _, act := range allActs {
			sb.WriteString(act.Title + ".\n\n")
			chaps, _ := ui.store.ListChapters(act.ID)
			for _, ch := range chaps {
				sb.WriteString(ch.Title + ".\n\n")
				scenes, _ := ui.store.ListScenes(ch.ID)
				for _, sc := range scenes {
					text := sc.Content
					if ui.editorPanel != nil && ui.editorPanel.activeScene != nil && ui.editorPanel.activeScene.ID == sc.ID {
						if ui.editorPanel.proseEntry != nil && strings.TrimSpace(ui.editorPanel.proseEntry.Text) != "" {
							text = ui.editorPanel.proseEntry.Text
						}
					}
					clean := CleanProseForSpeech(text)
					if clean == "" {
						continue
					}
					count++
					if sc.Title != "" {
						sb.WriteString(sc.Title + ".\n\n")
					}
					sb.WriteString(clean)
					sb.WriteString("\n\n---\n\n")
				}
			}
		}
		summaryInfo = fmt.Sprintf("Toàn bộ tác phẩm: %s (%d hồi, %d cảnh)", ui.activeProject.Title, len(allActs), count)
		return sb.String(), scopeTitle, count, summaryInfo
	}
}

// ShowAudioExportDialog mở hộp thoại trực quan hỗ trợ đa giọng đọc Neural Tiếng Việt
// và lựa chọn phạm vi phân cấp linh hoạt (Cảnh, Chương, Hồi, Toàn bộ tác phẩm).
func (ui *NovelistUI) ShowAudioExportDialog() {
	if ui.window == nil {
		return
	}

	// Xác định phạm vi ban đầu: ưu tiên Cảnh hiện tại nếu đang mở, ngược lại chọn Toàn bộ tác phẩm
	initialScope := AudioScopeCurrentScene
	if ui.editorPanel == nil || ui.editorPanel.activeScene == nil {
		initialScope = AudioScopeFullNovel
	}

	initialContent, initialTitle, _, initialSummary := ui.CompileAudioTextForScope(initialScope)

	// 1. Selector Phạm vi xuất bản (Audio Export Scope)
	scopeSelect := widget.NewSelect(AllAudioExportScopes, nil)
	scopeSelect.SetSelected(string(initialScope))

	scopeInfoLabel := widget.NewLabel(fmt.Sprintf("📌 %s", initialSummary))
	scopeInfoLabel.Wrapping = fyne.TextWrapWord
	scopeInfoLabel.TextStyle = fyne.TextStyle{Italic: true}

	// 2. Selector Giọng đọc Neural Tiếng Việt (Hoài Mỹ, Nam Minh, Hoàng Mai, Nam Khánh, Thu Trang)
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

	// 3. Khung soạn thảo / xem trước văn bản cần chuyển thành audio
	contentEntry := NewVietMultiLineEntry()
	contentEntry.SetPlaceHolder("Nội dung văn bản được biên dịch tự động theo phạm vi đã chọn...")
	contentEntry.SetText(initialContent)
	contentEntry.SetMinRowsVisible(7)

	statsLabel := widget.NewLabel("")
	updateStats := func(text string) {
		words := len(strings.Fields(text))
		chars := utf8.RuneCountInString(text)
		statsLabel.SetText(fmt.Sprintf("📊 Thống kê: %d từ  •  %d ký tự  •  Thời lượng ước tính: khoảng %.1f phút nghe (chuẩn 160 từ/phút)",
			words, chars, float64(words)/160.0))
	}
	updateStats(initialContent)

	contentEntry.SetOnChangedCallback(func(text string) {
		updateStats(text)
	})

	// 4. Đường dẫn lưu tệp MP3 đích
	suggestedFileName := sanitizeFileName(initialTitle) + ".mp3"
	pathEntry := NewVietEntry()
	pathEntry.SetText(suggestedFileName)

	// Lắng nghe thay đổi Phạm vi xuất bản: Tự động tổng hợp lại nội dung và cập nhật tên file
	scopeSelect.OnChanged = func(chosen string) {
		sc := AudioExportScope(chosen)
		compiled, title, totalScenes, summary := ui.CompileAudioTextForScope(sc)
		contentEntry.SetText(compiled)
		updateStats(compiled)
		suggestedFileName = sanitizeFileName(title) + ".mp3"
		pathEntry.SetText(suggestedFileName)
		scopeInfoLabel.SetText(fmt.Sprintf("📌 %s (tổng cộng %d cảnh được biên dịch)", summary, totalScenes))
	}

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

	// 5. Trạng thái và thanh tiến trình
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Hide()

	statusLabel := widget.NewLabelWithStyle("Sẵn sàng xuất audio bằng giọng đọc Neural Tiếng Việt.", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	reloadScopeBtn := widget.NewButtonWithIcon("Nạp lại theo phạm vi đang chọn", theme.ViewRefreshIcon(), func() {
		sc := AudioExportScope(scopeSelect.Selected)
		compiled, title, totalScenes, summary := ui.CompileAudioTextForScope(sc)
		contentEntry.SetText(compiled)
		updateStats(compiled)
		suggestedFileName = sanitizeFileName(title) + ".mp3"
		pathEntry.SetText(suggestedFileName)
		scopeInfoLabel.SetText(fmt.Sprintf("📌 %s (tổng cộng %d cảnh được biên dịch)", summary, totalScenes))
	})
	reloadScopeBtn.Importance = widget.LowImportance

	// 6. Xây dựng bố cục Dialog
	headerBox := container.NewVBox(
		widget.NewLabelWithStyle("🎧 XUẤT BẢN FILE AUDIO (MP3) — ĐA GIỌNG ĐỌC AI & PHẠM VI LINH HOẠT", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Sử dụng công nghệ Neural Text-to-Speech (Edge-TTS) miễn phí với đầy đủ giọng điệu 3 miền Bắc - Trung - Nam."),
		widget.NewSeparator(),
	)

	scopeCard := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabelWithStyle("Phạm vi xuất bản:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, scopeSelect),
		scopeInfoLabel,
	)

	voiceCard := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabelWithStyle("Chọn giọng đọc:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, voiceSelect),
		voiceDescLabel,
	)

	contentHeader := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("Nội dung văn bản biên dịch để đọc thành tiếng:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		reloadScopeBtn,
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

		wordCount := len(strings.Fields(textToRead))

		// Khóa giao diện và hiển thị tiến trình tổng hợp
		exportBtn.Disable()
		cancelBtn.Disable()
		scopeSelect.Disable()
		voiceSelect.Disable()
		progressBar.Show()

		statusLabel.SetText(fmt.Sprintf("⏳ Đang tổng hợp nội dung và khởi chạy Edge-TTS Neural (%s — %d từ)...", scopeSelect.Selected, wordCount))

		voiceChosen := selectedVoiceID
		go func() {
			err := ExportTextToAudioWithEdgeTTS(voiceChosen, textToRead, outPath)

			exportBtn.Enable()
			cancelBtn.Enable()
			scopeSelect.Enable()
			voiceSelect.Enable()
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
				fmt.Sprintf("Đã xuất file âm thanh thành công!\n\n• Tệp đích: %s\n• Phạm vi xuất bản: %s\n• Giọng đọc: %s\n• Dung lượng: %.2f MB\n• Số từ: %d từ\n• Thời lượng ước tính: %.1f phút nghe",
					outPath,
					scopeSelect.Selected,
					voiceSelect.Selected,
					fileSizeMB,
					wordCount,
					float64(wordCount)/160.0,
				),
				ui.window,
			)
		}()
	}

	actionsRow := container.NewHBox(
		cancelBtn,
		layout.NewSpacer(),
		exportBtn,
	)

	dialogBody := container.NewVScroll(container.NewVBox(
		headerBox,
		scopeCard,
		widget.NewSeparator(),
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
	audioDialog.Resize(fyne.NewSize(780, 680))
	audioDialog.Show()
}
