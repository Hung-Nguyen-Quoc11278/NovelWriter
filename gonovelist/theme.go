package main

import (
	"image/color"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Các hằng số cấu hình cỡ chữ và khóa lưu trữ Preferences cho Chủ đề (Theme)
const (
	DefaultEditorFontSize float32 = 18.0
	MinEditorFontSize     float32 = 12.0
	MaxEditorFontSize     float32 = 32.0
	EditorFontSizeStep    float32 = 2.0

	PrefKeyEditorFontSize string = "gonovelist.editor.font_size"
	PrefKeyThemeMode      string = "gonovelist.ui.theme_mode"
)

// ThemeMode định danh chế độ màu sắc giao diện của GoNovelist.
type ThemeMode string

const (
	ThemeModeLight ThemeMode = "light" // Chế độ Sáng (Mặc định)
	ThemeModeDark  ThemeMode = "dark"  // Chế độ Tối (Ban đêm)
	ThemeModeSepia ThemeMode = "sepia" // Giấy cổ điển (Sepia chống mỏi mắt)
)

// AllThemeModes trả về danh sách các chế độ chủ đề hỗ trợ trong GoNovelist.
func AllThemeModes() []ThemeMode {
	return []ThemeMode{
		ThemeModeLight,
		ThemeModeDark,
		ThemeModeSepia,
	}
}

// ThemeModeLabel trả về tên hiển thị Tiếng Việt chuẩn của từng Chủ đề (Theme).
func ThemeModeLabel(mode ThemeMode) string {
	switch mode {
	case ThemeModeDark:
		return "Chế độ Tối"
	case ThemeModeSepia:
		return "Giấy cổ điển (Sepia)"
	default:
		return "Chế độ Sáng"
	}
}

// AllThemeModeLabels trả về danh sách nhãn Tiếng Việt cho bộ chọn Chủ đề trong Cài đặt.
func AllThemeModeLabels() []string {
	return []string{
		ThemeModeLabel(ThemeModeLight),
		ThemeModeLabel(ThemeModeDark),
		ThemeModeLabel(ThemeModeSepia),
	}
}

// ParseThemeModeLabel chuyển đổi nhãn Tiếng Việt hoặc mã chuỗi về ThemeMode chuẩn.
func ParseThemeModeLabel(raw string) ThemeMode {
	clean := strings.TrimSpace(raw)
	switch clean {
	case "Chế độ Tối", "dark", "Dark":
		return ThemeModeDark
	case "Giấy cổ điển (Sepia)", "sepia", "Sepia":
		return ThemeModeSepia
	default:
		return ThemeModeLight
	}
}

// ThemePalette chứa bảng màu chi tiết cho từng chế độ giao diện (Light / Dark / Sepia).
type ThemePalette struct {
	Background        color.NRGBA
	Surface           color.NRGBA
	HeaderBackground  color.NRGBA
	InputBackground   color.NRGBA
	OverlayBackground color.NRGBA
	Button            color.NRGBA
	DisabledButton    color.NRGBA
	Foreground        color.NRGBA
	PlaceHolder       color.NRGBA
	Disabled          color.NRGBA
	Primary           color.NRGBA
	Hover             color.NRGBA
	Pressed           color.NRGBA
	Selection         color.NRGBA
	Separator         color.NRGBA
	InputBorder       color.NRGBA
	ScrollBar         color.NRGBA
	Shadow            color.NRGBA
}

// PaletteForMode trả về bảng màu tối ưu hóa độ tương phản và chống mỏi mắt cho từng chế độ.
func PaletteForMode(mode ThemeMode) ThemePalette {
	switch mode {
	case ThemeModeDark:
		// Chế độ Tối (Đêm): Nền than chì dịu mắt, chữ trắng ngà độ tương phản cao, điểm nhấn xanh lam sáng
		return ThemePalette{
			Background:        color.NRGBA{R: 0x13, G: 0x17, B: 0x22, A: 0xff}, // #131722
			Surface:           color.NRGBA{R: 0x1b, G: 0x22, B: 0x30, A: 0xff}, // #1B2230
			HeaderBackground:  color.NRGBA{R: 0x18, G: 0x1e, B: 0x2b, A: 0xff}, // #181E2B
			InputBackground:   color.NRGBA{R: 0x1e, G: 0x26, B: 0x36, A: 0xff}, // #1E2636
			OverlayBackground: color.NRGBA{R: 0x1b, G: 0x22, B: 0x30, A: 0xf8}, // #1B2230
			Button:            color.NRGBA{R: 0x26, G: 0x30, B: 0x44, A: 0xff}, // #263044
			DisabledButton:    color.NRGBA{R: 0x1c, G: 0x22, B: 0x30, A: 0xff},
			Foreground:        color.NRGBA{R: 0xe6, G: 0xed, B: 0xf7, A: 0xff}, // #E6EDF7
			PlaceHolder:       color.NRGBA{R: 0x7c, G: 0x8b, B: 0xa1, A: 0xff}, // #7C8BA1
			Disabled:          color.NRGBA{R: 0x56, G: 0x63, B: 0x78, A: 0xff},
			Primary:           color.NRGBA{R: 0x38, G: 0xbd, B: 0xf8, A: 0xff}, // #38BDF8
			Hover:             color.NRGBA{R: 0x2d, G: 0x3a, B: 0x52, A: 0xff},
			Pressed:           color.NRGBA{R: 0x35, G: 0x46, B: 0x64, A: 0xff},
			Selection:         color.NRGBA{R: 0x38, G: 0xbd, B: 0xf8, A: 0x45},
			Separator:         color.NRGBA{R: 0x2c, G: 0x37, B: 0x4b, A: 0xff}, // #2C374B
			InputBorder:       color.NRGBA{R: 0x36, G: 0x44, B: 0x5d, A: 0xff}, // #36445D
			ScrollBar:         color.NRGBA{R: 0x4b, G: 0x5b, B: 0x78, A: 0xaa},
			Shadow:            color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x77},
		}

	case ThemeModeSepia:
		// Giấy cổ điển (Sepia): Nền giấy dó vàng ấm, chữ nâu mực tàu đậm, điểm nhấn đỏ gạch cổ điển
		return ThemePalette{
			Background:        color.NRGBA{R: 0xf4, G: 0xec, B: 0xd8, A: 0xff}, // #F4ECD8 (Giấy cổ ấm)
			Surface:           color.NRGBA{R: 0xec, G: 0xe1, B: 0xc6, A: 0xff}, // #ECE1C6
			HeaderBackground:  color.NRGBA{R: 0xe6, G: 0xd9, B: 0xb8, A: 0xff}, // #E6D9B8
			InputBackground:   color.NRGBA{R: 0xfa, G: 0xf4, B: 0xe4, A: 0xff}, // #FAF4E4 (Trang giấy viết)
			OverlayBackground: color.NRGBA{R: 0xf7, G: 0xf0, B: 0xde, A: 0xfc}, // #F7F0DE
			Button:            color.NRGBA{R: 0xe3, G: 0xd5, B: 0xb5, A: 0xff}, // #E3D5B5
			DisabledButton:    color.NRGBA{R: 0xec, G: 0xe3, B: 0xcd, A: 0xff},
			Foreground:        color.NRGBA{R: 0x3b, G: 0x2a, B: 0x1e, A: 0xff}, // #3B2A1E (Mực nâu đen đậm)
			PlaceHolder:       color.NRGBA{R: 0x84, G: 0x6d, B: 0x58, A: 0xff}, // #846D58
			Disabled:          color.NRGBA{R: 0xa3, G: 0x8f, B: 0x78, A: 0xff},
			Primary:           color.NRGBA{R: 0x8b, G: 0x3a, B: 0x2b, A: 0xff}, // #8B3A2B (Đỏ chu sa trầm)
			Hover:             color.NRGBA{R: 0xe8, G: 0xdc, B: 0xbe, A: 0xff},
			Pressed:           color.NRGBA{R: 0xd8, G: 0xc8, B: 0xa4, A: 0xff},
			Selection:         color.NRGBA{R: 0x8b, G: 0x3a, B: 0x2b, A: 0x33},
			Separator:         color.NRGBA{R: 0xd3, G: 0xc3, B: 0xa3, A: 0xff}, // #D3C3A3
			InputBorder:       color.NRGBA{R: 0xc8, G: 0xb6, B: 0x93, A: 0xff}, // #C8B693
			ScrollBar:         color.NRGBA{R: 0x9c, G: 0x84, B: 0x68, A: 0x99},
			Shadow:            color.NRGBA{R: 0x3b, G: 0x2a, B: 0x1e, A: 0x28},
		}

	default:
		// Chế độ Sáng (Light mặc định): Sạch sẽ, hiện đại, tương phản rõ nét
		return ThemePalette{
			Background:        color.NRGBA{R: 0xf8, G: 0xfa, B: 0xfc, A: 0xff}, // #F8FAFC
			Surface:           color.NRGBA{R: 0xf1, G: 0xf5, B: 0xf9, A: 0xff}, // #F1F5F9
			HeaderBackground:  color.NRGBA{R: 0xe2, G: 0xe8, B: 0xf0, A: 0xff}, // #E2E8F0
			InputBackground:   color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, // #FFFFFF
			OverlayBackground: color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xfc},
			Button:            color.NRGBA{R: 0xe2, G: 0xe8, B: 0xf0, A: 0xff}, // #E2E8F0
			DisabledButton:    color.NRGBA{R: 0xf1, G: 0xf5, B: 0xf9, A: 0xff},
			Foreground:        color.NRGBA{R: 0x0f, G: 0x17, B: 0x2a, A: 0xff}, // #0F172A
			PlaceHolder:       color.NRGBA{R: 0x64, G: 0x74, B: 0x8b, A: 0xff}, // #64748B
			Disabled:          color.NRGBA{R: 0x94, G: 0xa3, B: 0xb8, A: 0xff},
			Primary:           color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0xff}, // #2563EB
			Hover:             color.NRGBA{R: 0xe2, G: 0xe8, B: 0xf0, A: 0x99},
			Pressed:           color.NRGBA{R: 0xcb, G: 0xd5, B: 0xe1, A: 0xff},
			Selection:         color.NRGBA{R: 0x25, G: 0x63, B: 0xeb, A: 0x33},
			Separator:         color.NRGBA{R: 0xcb, G: 0xd5, B: 0xe1, A: 0xff}, // #CBD5E1
			InputBorder:       color.NRGBA{R: 0x94, G: 0xa3, B: 0xb8, A: 0xff}, // #94A3B8
			ScrollBar:         color.NRGBA{R: 0x64, G: 0x74, B: 0x8b, A: 0x88},
			Shadow:            color.NRGBA{R: 0x0f, G: 0x17, B: 0x2a, A: 0x1f},
		}
	}
}

// DynamicFontTheme triển khai fyne.Theme hỗ trợ đồng thời:
//  1. Ba chế độ màu sắc: Chế độ Sáng (Light), Chế độ Tối (Dark), Giấy cổ điển (Sepia).
//  2. Thu phóng cỡ chữ động (Dynamic Font Size / Zoom) từ 12px đến 32px theo thời gian thực.
type DynamicFontTheme struct {
	mu       sync.RWMutex
	base     fyne.Theme
	textSize float32
	mode     ThemeMode
}

// NewDynamicFontTheme khởi tạo chủ đề tùy chỉnh với cỡ chữ văn bản mong muốn và Chế độ Sáng mặc định.
func NewDynamicFontTheme(initialSize float32, optionalMode ...ThemeMode) *DynamicFontTheme {
	mode := ThemeModeLight
	if len(optionalMode) > 0 && optionalMode[0] != "" {
		mode = optionalMode[0]
	}
	return &DynamicFontTheme{
		base:     theme.DefaultTheme(),
		textSize: clampFontSize(initialSize),
		mode:     mode,
	}
}

func clampFontSize(sz float32) float32 {
	if sz < MinEditorFontSize {
		return MinEditorFontSize
	}
	if sz > MaxEditorFontSize {
		return MaxEditorFontSize
	}
	return sz
}

// Mode trả về chế độ màu sắc hiện hành (Light / Dark / Sepia).
func (t *DynamicFontTheme) Mode() ThemeMode {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.mode == "" {
		return ThemeModeLight
	}
	return t.mode
}

// SetMode cập nhật chế độ Chủ đề (Light / Dark / Sepia) ngay lập tức.
func (t *DynamicFontTheme) SetMode(mode ThemeMode) {
	t.mu.Lock()
	switch mode {
	case ThemeModeDark, ThemeModeSepia, ThemeModeLight:
		t.mode = mode
	default:
		t.mode = ThemeModeLight
	}
	t.mu.Unlock()
}

// TextSize trả về cỡ chữ hiện tại đang áp dụng.
func (t *DynamicFontTheme) TextSize() float32 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.textSize
}

// SetTextSize cập nhật cỡ chữ mới trong giới hạn an toàn [12px..32px].
func (t *DynamicFontTheme) SetTextSize(sz float32) float32 {
	clamped := clampFontSize(sz)
	t.mu.Lock()
	t.textSize = clamped
	t.mu.Unlock()
	return clamped
}

// ZoomPercent trả về tỷ lệ phần trăm thu phóng so với cỡ chữ gốc 14px của Fyne.
func (t *DynamicFontTheme) ZoomPercent() int {
	return int((t.TextSize()/14.0)*100.0 + 0.5)
}

// Color trả về màu sắc tương ứng với Chủ đề đang chọn (Chế độ Sáng / Chế độ Tối / Giấy cổ điển Sepia).
func (t *DynamicFontTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	t.mu.RLock()
	mode := t.mode
	t.mu.RUnlock()

	p := PaletteForMode(mode)

	switch name {
	case theme.ColorNameBackground:
		return p.Background
	case theme.ColorNameHeaderBackground:
		return p.HeaderBackground
	case theme.ColorNameMenuBackground:
		return p.Surface
	case theme.ColorNameOverlayBackground:
		return p.OverlayBackground
	case theme.ColorNameInputBackground:
		return p.InputBackground
	case theme.ColorNameButton:
		return p.Button
	case theme.ColorNameDisabledButton:
		return p.DisabledButton
	case theme.ColorNameForeground:
		return p.Foreground
	case theme.ColorNamePlaceHolder:
		return p.PlaceHolder
	case theme.ColorNameDisabled:
		return p.Disabled
	case theme.ColorNamePrimary, theme.ColorNameFocus, theme.ColorNameHyperlink:
		return p.Primary
	case theme.ColorNameHover:
		return p.Hover
	case theme.ColorNamePressed:
		return p.Pressed
	case theme.ColorNameSelection:
		return p.Selection
	case theme.ColorNameSeparator:
		return p.Separator
	case theme.ColorNameInputBorder:
		return p.InputBorder
	case theme.ColorNameScrollBar:
		return p.ScrollBar
	case theme.ColorNameShadow:
		return p.Shadow
	default:
		if mode == ThemeModeDark {
			return t.base.Color(name, theme.VariantDark)
		}
		return t.base.Color(name, theme.VariantLight)
	}
}

func (t *DynamicFontTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *DynamicFontTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

// Size ghi đè kích thước chữ (SizeNameText, SubHeading, Heading, LineSpacing) theo tỷ lệ thu phóng động.
func (t *DynamicFontTheme) Size(name fyne.ThemeSizeName) float32 {
	t.mu.RLock()
	sz := t.textSize
	t.mu.RUnlock()

	switch name {
	case theme.SizeNameText:
		return sz
	case theme.SizeNameSubHeadingText:
		return sz * 1.18
	case theme.SizeNameHeadingText:
		return sz * 1.42
	case theme.SizeNameCaptionText:
		if sz*0.82 < 12 {
			return 12
		}
		return sz * 0.82
	case theme.SizeNameLineSpacing:
		// Tăng khoảng cách dòng tỷ lệ thuận với cỡ chữ để hiển thị dấu Tiếng Việt rõ ràng, không dính dòng
		return t.base.Size(name) * (sz / 14.0)
	default:
		return t.base.Size(name)
	}
}
