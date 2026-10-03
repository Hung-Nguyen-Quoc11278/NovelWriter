package main

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ApplyThemeMode áp dụng ngay lập tức Chủ đề (Chế độ Sáng / Chế độ Tối / Giấy cổ điển Sepia)
// trên toàn bộ cửa sổ ứng dụng Fyne mà không cần khởi động lại.
func (ui *NovelistUI) ApplyThemeMode(mode ThemeMode) {
	if ui.fontTheme == nil {
		ui.fontTheme = NewDynamicFontTheme(DefaultEditorFontSize, mode)
	} else {
		ui.fontTheme.SetMode(mode)
	}

	if ui.app != nil {
		ui.app.Preferences().SetString(PrefKeyThemeMode, string(mode))
		ui.app.Settings().SetTheme(ui.fontTheme)
	}

	ui.ForceLayoutRefresh()
	if ui.window != nil && ui.window.Canvas() != nil && ui.window.Content() != nil {
		ui.window.Canvas().Refresh(ui.window.Content())
	}
	if ui.statusFooter != nil {
		ui.statusFooter.SetText(fmt.Sprintf("Đã áp dụng chủ đề: %s.", ThemeModeLabel(mode)))
	}
}

// buildThemePreviewCard dựng thẻ minh họa trực quan màu nền, màu giấy soạn thảo và màu chữ của từng Chủ đề.
func buildThemePreviewCard(mode ThemeMode, onSelect func()) fyne.CanvasObject {
	pal := PaletteForMode(mode)

	outerBg := canvas.NewRectangle(pal.Background)
	outerBg.CornerRadius = 8
	outerBg.StrokeColor = pal.Primary
	outerBg.StrokeWidth = 1.5

	paperBg := canvas.NewRectangle(pal.InputBackground)
	paperBg.CornerRadius = 5
	paperBg.StrokeColor = pal.InputBorder
	paperBg.StrokeWidth = 1

	titleText := canvas.NewText(ThemeModeLabel(mode), pal.Foreground)
	titleText.TextStyle = fyne.TextStyle{Bold: true}
	titleText.TextSize = 14

	sampleText := canvas.NewText("“Ánh đèn dầu lạc phản chiếu qua phiến thủy tinh cổ...”", pal.Foreground)
	sampleText.TextSize = 12

	accentDot := canvas.NewCircle(pal.Primary)
	dotWrap := container.NewGridWrap(fyne.NewSize(14, 14), accentDot)

	applyBtn := widget.NewButton("Áp dụng "+ThemeModeLabel(mode), func() {
		PlayUIClickSound()
		if onSelect != nil {
			onSelect()
		}
	})
	applyBtn.Importance = widget.LowImportance

	innerPaper := container.NewMax(
		paperBg,
		container.NewPadded(sampleText),
	)

	cardBody := container.NewVBox(
		container.NewHBox(container.NewCenter(dotWrap), titleText),
		innerPaper,
		applyBtn,
	)

	return container.NewMax(outerBg, container.NewPadded(cardBody))
}

// ShowSettingsDialog hiển thị hộp thoại "Cài đặt" (Settings Hub) đa tab cho phép tùy chỉnh
// Chủ đề (Theme: Chế độ Sáng / Chế độ Tối / Giấy cổ điển Sepia), Cỡ chữ và Âm thanh giao diện.
func (ui *NovelistUI) ShowSettingsDialog() {
	PlayUIClickSound()

	currentMode := ThemeModeLight
	currentFontSize := DefaultEditorFontSize
	if ui.fontTheme != nil {
		currentMode = ui.fontTheme.Mode()
		currentFontSize = ui.fontTheme.TextSize()
	}

	// ==================== TAB 1: CHỦ ĐỀ (THEME) & HIỂN THỊ ====================
	themeLabels := AllThemeModeLabels()
	syncingThemeWidgets := false

	var themeRadio *widget.RadioGroup
	var themeSelect *widget.Select

	applySelectedTheme := func(label string) {
		if syncingThemeWidgets || label == "" {
			return
		}
		syncingThemeWidgets = true
		mode := ParseThemeModeLabel(label)
		if themeRadio != nil && themeRadio.Selected != ThemeModeLabel(mode) {
			themeRadio.SetSelected(ThemeModeLabel(mode))
		}
		if themeSelect != nil && themeSelect.Selected != ThemeModeLabel(mode) {
			themeSelect.SetSelected(ThemeModeLabel(mode))
		}
		ui.ApplyThemeMode(mode)
		syncingThemeWidgets = false
	}

	themeSelect = widget.NewSelect(themeLabels, func(selected string) {
		PlayUIClickSound()
		applySelectedTheme(selected)
	})
	themeSelect.SetSelected(ThemeModeLabel(currentMode))

	themeRadio = widget.NewRadioGroup(themeLabels, func(selected string) {
		PlayUIClickSound()
		applySelectedTheme(selected)
	})
	themeRadio.Horizontal = true
	themeRadio.SetSelected(ThemeModeLabel(currentMode))

	// Các thẻ xem trước 3 chủ đề: Chế độ Sáng, Chế độ Tối, Giấy cổ điển (Sepia)
	previewCardsGrid := container.NewGridWithColumns(3,
		buildThemePreviewCard(ThemeModeLight, func() {
			applySelectedTheme(ThemeModeLabel(ThemeModeLight))
		}),
		buildThemePreviewCard(ThemeModeDark, func() {
			applySelectedTheme(ThemeModeLabel(ThemeModeDark))
		}),
		buildThemePreviewCard(ThemeModeSepia, func() {
			applySelectedTheme(ThemeModeLabel(ThemeModeSepia))
		}),
	)

	// Thanh trượt điều chỉnh cỡ chữ trình soạn thảo (12px - 32px)
	fontSizeLabel := widget.NewLabelWithStyle(
		fmt.Sprintf("Cỡ chữ hiện tại: %dpx (%d%%)", int(currentFontSize), ui.fontTheme.ZoomPercent()),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	fontSlider := widget.NewSlider(float64(MinEditorFontSize), float64(MaxEditorFontSize))
	fontSlider.Step = float64(EditorFontSizeStep)
	fontSlider.Value = float64(currentFontSize)
	fontSlider.OnChanged = func(val float64) {
		delta := float32(val) - ui.fontTheme.TextSize()
		if delta != 0 {
			ui.AdjustFontSize(delta)
			fontSizeLabel.SetText(fmt.Sprintf("Cỡ chữ hiện tại: %dpx (%d%%)", int(ui.fontTheme.TextSize()), ui.fontTheme.ZoomPercent()))
		}
	}

	resetFontBtn := widget.NewButtonWithIcon("Đặt lại cỡ chữ mặc định (18px)", theme.ViewRefreshIcon(), func() {
		PlayUIClickSound()
		ui.ResetFontSize()
		fontSlider.SetValue(float64(DefaultEditorFontSize))
		fontSizeLabel.SetText(fmt.Sprintf("Cỡ chữ hiện tại: %dpx (%d%%)", int(ui.fontTheme.TextSize()), ui.fontTheme.ZoomPercent()))
	})

	themeTabContent := container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("CHỦ ĐỀ GIAO DIỆN (THEME)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Chọn chế độ màu sắc phù hợp với không gian sáng tác của bạn. Chủ đề mới sẽ áp dụng ngay lập tức trên toàn bộ cửa sổ:"),
		widget.NewForm(
			widget.NewFormItem("Chủ đề (Theme)", themeSelect),
			widget.NewFormItem("Chuyển nhanh chế độ", themeRadio),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("XEM TRƯỚC BẢNG MÀU CHỦ ĐỀ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		previewCardsGrid,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("THU PHÓNG CỠ CHỮ TRÌNH SOẠN THẢO", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		fontSizeLabel,
		fontSlider,
		container.NewHBox(resetFontBtn, layout.NewSpacer()),
	))

	// ==================== TAB 2: ÂM THANH GIAO DIỆN & BỘ GÕ ====================
	masterSoundCheck := widget.NewCheck("Bật âm thanh giao diện (Gõ phím & Nhấp chuột)", nil)
	masterSoundCheck.SetChecked(GlobalSoundManager.IsEnabled())

	typingSoundCheck := widget.NewCheck("Bật âm thanh gõ phím trong trình soạn thảo (Typewriter / Mechanical)", func(checked bool) {
		GlobalSoundManager.SetTypingSoundEnabled(checked)
		if ui.app != nil {
			GlobalSoundManager.SaveToPreferences(ui.app.Preferences())
		}
		if checked {
			PlayTypingSound('a', nil)
		}
	})
	typingSoundCheck.SetChecked(GlobalSoundManager.IsTypingSoundEnabled())

	clickSoundCheck := widget.NewCheck("Bật âm thanh khi nhấp nút & chuyển mục giao diện", func(checked bool) {
		GlobalSoundManager.SetClickSoundEnabled(checked)
		if ui.app != nil {
			GlobalSoundManager.SaveToPreferences(ui.app.Preferences())
		}
		if checked {
			PlayUIClickSound()
		}
	})
	clickSoundCheck.SetChecked(GlobalSoundManager.IsClickSoundEnabled())

	masterSoundCheck.OnChanged = func(checked bool) {
		GlobalSoundManager.SetEnabled(checked)
		if checked {
			typingSoundCheck.Enable()
			clickSoundCheck.Enable()
			PlayUIClickSound()
		} else {
			typingSoundCheck.Disable()
			clickSoundCheck.Disable()
		}
		if ui.app != nil {
			GlobalSoundManager.SaveToPreferences(ui.app.Preferences())
		}
	}
	if !GlobalSoundManager.IsEnabled() {
		typingSoundCheck.Disable()
		clickSoundCheck.Disable()
	}

	profileSelect := widget.NewSelect(AllSoundProfileLabels(), func(selected string) {
		prof := ParseSoundProfileLabel(selected)
		GlobalSoundManager.SetProfile(prof)
		if ui.app != nil {
			GlobalSoundManager.SaveToPreferences(ui.app.Preferences())
		}
		PlayTypingSound('a', nil)
	})
	profileSelect.SetSelected(SoundProfileLabel(GlobalSoundManager.Profile()))

	volPercent := int(GlobalSoundManager.Volume()*100.0 + 0.5)
	volLabel := widget.NewLabelWithStyle(
		fmt.Sprintf("Âm lượng hiệu ứng: %d%%", volPercent),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
	volSlider := widget.NewSlider(10, 100)
	volSlider.Step = 5
	volSlider.Value = float64(volPercent)
	volSlider.OnChanged = func(val float64) {
		volLabel.SetText(fmt.Sprintf("Âm lượng hiệu ứng: %d%%", int(val)))
		GlobalSoundManager.SetVolume(val / 100.0)
		if ui.app != nil {
			GlobalSoundManager.SaveToPreferences(ui.app.Preferences())
		}
	}

	testTypingSoundBtn := widget.NewButtonWithIcon("Nghe thử tiếng gõ phím", theme.MediaPlayIcon(), func() {
		PlayTypingSound('a', nil)
	})
	testSpaceSoundBtn := widget.NewButtonWithIcon("Nghe thử phím Cách / Enter", theme.MediaPlayIcon(), func() {
		PlayTypingSound(' ', nil)
	})
	testClickSoundBtn := widget.NewButtonWithIcon("Nghe thử tiếng nhấp nút", theme.MediaPlayIcon(), func() {
		PlayUIClickSound()
	})

	telexToggleCheck := widget.NewCheck("Bật bộ gõ Tiếng Việt Telex nội bộ (Chỉ bật khi máy không dùng Fcitx5/IBus)", func(checked bool) {
		GlobalTelexEnabled = checked
		PlayUIClickSound()
	})
	telexToggleCheck.SetChecked(GlobalTelexEnabled)

	audioTabContent := container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("ÂM THANH GIAO DIỆN (UI SOUND EFFECTS)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Tạo cảm giác gõ máy chữ cơ học chân thực khi sáng tác bản thảo và phản hồi âm thanh nhẹ nhàng khi bấm nút:"),
		masterSoundCheck,
		typingSoundCheck,
		clickSoundCheck,
		widget.NewSeparator(),
		widget.NewForm(
			widget.NewFormItem("Kiểu âm thanh gõ phím", profileSelect),
			widget.NewFormItem("Điều chỉnh âm lượng", container.NewVBox(volLabel, volSlider)),
		),
		container.NewHBox(testTypingSoundBtn, testSpaceSoundBtn, testClickSoundBtn),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("CẤU HÌNH BỘ GÕ TIẾNG VIỆT", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		telexToggleCheck,
	))

	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Chủ đề (Theme)", theme.ColorPaletteIcon(), themeTabContent),
		container.NewTabItemWithIcon("Âm thanh giao diện", theme.VolumeUpIcon(), audioTabContent),
	)

	d := dialog.NewCustom("Cài Đặt Hệ Thống — GoNovelist", "Đóng", tabs, ui.window)
	ui.attachLayoutRefreshOnClose(d)
	d.Resize(fyne.NewSize(780, 560))
	d.Show()
}
