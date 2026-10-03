package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

// SplitTextIntoTTSChunks phân đoạn văn bản theo ranh giới câu và đoạn văn để giới hạn kích thước yêu cầu Edge-TTS.
func SplitTextIntoTTSChunks(text string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = 2500
	}
	clean := strings.TrimSpace(text)
	if clean == "" {
		return nil
	}
	if utf8.RuneCountInString(clean) <= maxRunes {
		return []string{clean}
	}

	paragraphs := strings.Split(clean, "\n")
	var chunks []string
	var currentChunk strings.Builder

	for _, p := range paragraphs {
		trimmedP := strings.TrimSpace(p)
		if trimmedP == "" {
			continue
		}

		pRunes := utf8.RuneCountInString(trimmedP)
		curRunes := utf8.RuneCountInString(currentChunk.String())

		if pRunes > maxRunes {
			sentences := splitParagraphIntoSentences(trimmedP)
			for _, s := range sentences {
				sRunes := utf8.RuneCountInString(s)
				if sRunes > maxRunes {
					if curRunes > 0 {
						chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
						currentChunk.Reset()
						curRunes = 0
					}
					chunks = append(chunks, splitTextAtRuneLimit(s, maxRunes)...)
					continue
				}
				if curRunes+sRunes+1 > maxRunes && curRunes > 0 {
					chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
					currentChunk.Reset()
					curRunes = 0
				}
				if currentChunk.Len() > 0 {
					currentChunk.WriteString(" ")
				}
				currentChunk.WriteString(s)
				curRunes += sRunes + 1
			}
			continue
		}

		if curRunes+pRunes+2 > maxRunes && curRunes > 0 {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(trimmedP)
	}

	if currentChunk.Len() > 0 {
		trimmed := strings.TrimSpace(currentChunk.String())
		if trimmed != "" {
			chunks = append(chunks, trimmed)
		}
	}

	return chunks
}

func splitTextAtRuneLimit(text string, maxRunes int) []string {
	runes := []rune(text)
	chunks := make([]string, 0, (len(runes)+maxRunes-1)/maxRunes)
	for start := 0; start < len(runes); start += maxRunes {
		end := start + maxRunes
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
	}
	return chunks
}

func splitParagraphIntoSentences(p string) []string {
	re := regexp.MustCompile(`[^.!?…]+[.!?…]*`)
	matches := re.FindAllString(p, -1)
	if len(matches) == 0 {
		return []string{p}
	}
	var res []string
	for _, m := range matches {
		t := strings.TrimSpace(m)
		if t != "" {
			res = append(res, t)
		}
	}
	return res
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

// ShowAudioExportDialog mở hộp thoại chọn giọng đọc và phạm vi xuất audio theo cấu trúc tác phẩm.
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
				ui.showErrorDialog(err)
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
		attachWindowRefreshOnClose(fd, ui.window)
		fd.SetFileName(filepath.Base(pathEntry.Text))
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".mp3"}))
		fd.Show()
	})

	// 5. Trạng thái và thanh tiến trình
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	statusLabel := widget.NewLabelWithStyle("Sẵn sàng xuất audio bằng công cụ Edge-TTS.", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

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
		widget.NewLabel("Sử dụng Edge-TTS với các giọng đọc tiếng Việt ba miền; ứng dụng ưu tiên công cụ được đóng gói kèm theo."),
		widget.NewSeparator(),
	)

	scopeCard := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabelWithStyle("Phạm vi xuất bản:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, scopeSelect),
		scopeInfoLabel,
	)

	rateLabel := widget.NewLabel("Tốc độ đọc: +0%")
	rateSlider := widget.NewSlider(-50, 50)
	rateSlider.Step = 5
	rateSlider.OnChanged = func(value float64) {
		rateLabel.SetText("Tốc độ đọc: " + formatEdgeTTSAdjustment(int(value)))
	}

	volumeLabel := widget.NewLabel("Âm lượng giọng đọc: +0%")
	volumeSlider := widget.NewSlider(-50, 50)
	volumeSlider.Step = 5
	volumeSlider.OnChanged = func(value float64) {
		volumeLabel.SetText("Âm lượng giọng đọc: " + formatEdgeTTSAdjustment(int(value)))
	}

	voiceCard := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabelWithStyle("Chọn giọng đọc:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, voiceSelect),
		voiceDescLabel,
		rateLabel,
		rateSlider,
		volumeLabel,
		volumeSlider,
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
			statusLabel.SetText("Vui lòng nhập nội dung văn bản để chuyển đổi sang âm thanh.")
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
		rateSlider.Disable()
		volumeSlider.Disable()
		progressBar.Show()

		statusLabel.SetText(fmt.Sprintf("⏳ Đang khởi tạo công cụ giọng đọc (%s — %d từ)...", scopeSelect.Selected, wordCount))

		voiceChosen := selectedVoiceID
		scopeChosen := scopeSelect.Selected
		voiceLabelChosen := voiceSelect.Selected
		rateChosen := int(rateSlider.Value)
		volumeChosen := int(volumeSlider.Value)
		go func() {
			err := ExportTextToAudioWithOptionsProgress(voiceChosen, textToRead, outPath, rateChosen, volumeChosen, func(curr, total int, msg string) {
				fyne.Do(func() {
					statusLabel.SetText(fmt.Sprintf("⏳ %s", msg))
					if total > 0 {
						progressBar.SetValue(float64(curr) / float64(total))
					}
				})
			})

			fileInfo, _ := os.Stat(outPath)
			fileSizeMB := float64(0)
			if fileInfo != nil {
				fileSizeMB = float64(fileInfo.Size()) / (1024 * 1024)
			}
			fyne.Do(func() {
				exportBtn.Enable()
				cancelBtn.Enable()
				scopeSelect.Enable()
				voiceSelect.Enable()
				rateSlider.Enable()
				volumeSlider.Enable()
				progressBar.Hide()

				if err != nil {
					statusLabel.SetText("❌ Không thể tạo tệp audio.")
					audioDialog.Hide()
					ui.showErrorDialog(err)
					return
				}

				PlayUIClickSound()
				statusLabel.SetText("✅ Xuất file thành công!")
				if ui.statusFooter != nil {
					ui.statusFooter.SetText(fmt.Sprintf("Đã xuất audio: %s (%.2f MB)", filepath.Base(outPath), fileSizeMB))
				}

				audioDialog.Hide()
				ui.showInformationDialog(
					"Xuất Audio Thành Công!",
					fmt.Sprintf("Đã xuất file âm thanh thành công.\n\n• Tệp đích: %s\n• Phạm vi xuất bản: %s\n• Giọng đọc: %s\n• Dung lượng: %.2f MB\n• Số từ: %d từ\n• Thời lượng ước tính: %.1f phút nghe",
						outPath,
						scopeChosen,
						voiceLabelChosen,
						fileSizeMB,
						wordCount,
						float64(wordCount)/160.0,
					),
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
	ui.attachLayoutRefreshOnClose(audioDialog)
	audioDialog.Resize(fyne.NewSize(780, 680))
	audioDialog.Show()
}
