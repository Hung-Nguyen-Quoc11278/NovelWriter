package main

import (
	"os"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// GlobalTelexEnabled cho phép bật/tắt bộ gõ Tiếng Việt Telex tích hợp sẵn
// trong trường hợp hệ điều hành (Linux X11/Wayland GLFW) không chuyển tiếp IBus/Fcitx5 vào Fyne.
var GlobalTelexEnabled = true

// ConfigureVietnameseFont tự động phát hiện và thiết lập phông chữ hệ thống hỗ trợ đầy đủ Unicode Tiếng Việt
// nếu người dùng chưa đặt biến môi trường FYNE_FONT.
func ConfigureVietnameseFont() {
	if os.Getenv("FYNE_FONT") != "" {
		return
	}
	candidateFonts := []string{
		// Linux (Arch, Ubuntu, Fedora, Debian)
		"/usr/share/fonts/noto/NotoSans-Regular.ttf",
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
		"/usr/share/fonts/liberation/LiberationSans-Regular.ttf",
		// macOS
		"/System/Library/Fonts/Supplemental/Arial.ttf",
		"/System/Library/Fonts/Helvetica.ttc",
		// Windows
		`C:\Windows\Fonts\segoeui.ttf`,
		`C:\Windows\Fonts\arial.ttf`,
	}
	for _, path := range candidateFonts {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			_ = os.Setenv("FYNE_FONT", path)
			break
		}
	}
}

// NewVietEntry tạo một widget.Entry một dòng hỗ trợ nhập tiếng Việt (cả IME hệ thống lẫn bộ gõ Telex tích hợp).
func NewVietEntry() *widget.Entry {
	e := widget.NewEntry()
	AttachVietnameseTelex(e, nil)
	return e
}

// NewVietMultiLineEntry tạo một widget.Entry nhiều dòng hỗ trợ nhập tiếng Việt hoàn chỉnh.
func NewVietMultiLineEntry() *widget.Entry {
	e := widget.NewMultiLineEntry()
	e.Wrapping = fyne.TextWrapWord
	AttachVietnameseTelex(e, nil)
	return e
}

// AttachVietnameseTelex gắn bộ xử lý gõ tiếng Việt Telex trực tiếp vào một widget.Entry
// đồng thời gọi lại callback onTextChanged (nếu có) sau khi đã biến đổi âm tiết tiếng Việt.
func AttachVietnameseTelex(entry *widget.Entry, onTextChanged func(string)) {
	var internalUpdating bool

	entry.OnChanged = func(current string) {
		if internalUpdating {
			return
		}
		if GlobalTelexEnabled && current != "" {
			composed := composeLastVietnameseWord(current)
			if composed != current {
				internalUpdating = true
				entry.SetText(composed)
				// Đưa con trỏ về cuối từ vừa gõ
				runes := []rune(composed)
				entry.CursorRow, entry.CursorColumn = computeCursorPos(runes)
				entry.Refresh()
				internalUpdating = false
				if onTextChanged != nil {
					onTextChanged(composed)
				}
				return
			}
		}
		if onTextChanged != nil {
			onTextChanged(current)
		}
	}
}

func computeCursorPos(runes []rune) (int, int) {
	row := 0
	col := 0
	for _, r := range runes {
		if r == '\n' {
			row++
			col = 0
		} else {
			col++
		}
	}
	return row, col
}

// Bảng nguyên âm tiếng Việt theo 6 thanh điệu: [ngang, sắc(s), huyền(f), hỏi(r), ngã(x), nặng(j)]
var vietVowelTable = [][]rune{
	{'a', 'á', 'à', 'ả', 'ã', 'ạ'},
	{'ă', 'ắ', 'ằ', 'ẳ', 'ẵ', 'ặ'},
	{'â', 'ấ', 'ầ', 'ẩ', 'ẫ', 'ậ'},
	{'e', 'é', 'è', 'ẻ', 'ẽ', 'ẹ'},
	{'ê', 'ế', 'ề', 'ể', 'ễ', 'ệ'},
	{'i', 'í', 'ì', 'ỉ', 'ĩ', 'ị'},
	{'o', 'ó', 'ò', 'ỏ', 'õ', 'ọ'},
	{'ô', 'ố', 'ồ', 'ổ', 'ỗ', 'ộ'},
	{'ơ', 'ớ', 'ờ', 'ở', 'ỡ', 'ợ'},
	{'u', 'ú', 'ù', 'ủ', 'ũ', 'ụ'},
	{'ư', 'ứ', 'ừ', 'ử', 'ữ', 'ự'},
	{'y', 'ý', 'ỳ', 'ỷ', 'ỹ', 'ỵ'},
	{'A', 'Á', 'À', 'Ả', 'Ã', 'Ạ'},
	{'Ă', 'Ắ', 'Ằ', 'Ẳ', 'Ẵ', 'Ặ'},
	{'Â', 'Ấ', 'Ầ', 'Ẩ', 'Ẫ', 'Ậ'},
	{'E', 'É', 'È', 'Ẻ', 'Ẽ', 'Ẹ'},
	{'Ê', 'Ế', 'Ề', 'Ể', 'Ễ', 'Ệ'},
	{'I', 'Í', 'Ì', 'Ỉ', 'Ĩ', 'Ị'},
	{'O', 'Ó', 'Ò', 'Ỏ', 'Õ', 'Ọ'},
	{'Ô', 'Ố', 'Ồ', 'Ổ', 'Ỗ', 'Ộ'},
	{'Ơ', 'Ớ', 'Ờ', 'Ở', 'Ỡ', 'Ợ'},
	{'U', 'Ú', 'Ù', 'Ủ', 'Ũ', 'Ụ'},
	{'Ư', 'Ứ', 'Ừ', 'Ử', 'Ữ', 'Ự'},
	{'Y', 'Ý', 'Ỳ', 'Ỷ', 'Ỹ', 'Ỵ'},
}

func findVowelRowAndTone(r rune) (int, int) {
	for rowIdx, row := range vietVowelTable {
		for toneIdx, cell := range row {
			if cell == r {
				return rowIdx, toneIdx
			}
		}
	}
	return -1, -1
}

// composeLastVietnameseWord áp dụng quy tắc Telex cho từ cuối cùng vừa được gõ.
func composeLastVietnameseWord(text string) string {
	runes := []rune(text)
	n := len(runes)
	if n < 2 {
		return text
	}

	// Tìm vị trí bắt đầu của từ cuối cùng
	start := n - 1
	for start >= 0 && unicode.IsLetter(runes[start]) {
		start--
	}
	wordStart := start + 1
	if n-wordStart < 2 || n-wordStart > 10 {
		return text
	}

	word := runes[wordStart:]
	transformed, changed := applyTelexToWord(word)
	if !changed {
		return text
	}
	return string(runes[:wordStart]) + string(transformed)
}

func applyTelexToWord(word []rune) ([]rune, bool) {
	m := len(word)
	if m < 2 {
		return word, false
	}

	last := word[m-1]
	lastLower := unicode.ToLower(last)
	prefix := make([]rune, m-1)
	copy(prefix, word[:m-1])

	// 1. Xử lý dd / DD -> đ / Đ
	if lastLower == 'd' && len(prefix) >= 1 {
		prev := prefix[len(prefix)-1]
		if prev == 'd' {
			prefix[len(prefix)-1] = 'đ'
			return prefix, true
		}
		if prev == 'D' {
			prefix[len(prefix)-1] = 'Đ'
			return prefix, true
		}
	}

	// 2. Xử lý mũ nguyên âm: aa -> â, ee -> ê, oo -> ô
	if lastLower == 'a' || lastLower == 'e' || lastLower == 'o' {
		for i := len(prefix) - 1; i >= 0; i-- {
			rowIdx, toneIdx := findVowelRowAndTone(prefix[i])
			if rowIdx < 0 {
				continue
			}
			base := vietVowelTable[rowIdx][0]
			if lastLower == 'a' && (base == 'a' || base == 'A') {
				targetRow := 2
				if base == 'A' {
					targetRow = 14
				}
				prefix[i] = vietVowelTable[targetRow][toneIdx]
				return prefix, true
			}
			if lastLower == 'e' && (base == 'e' || base == 'E') {
				targetRow := 4
				if base == 'E' {
					targetRow = 16
				}
				prefix[i] = vietVowelTable[targetRow][toneIdx]
				return prefix, true
			}
			if lastLower == 'o' && (base == 'o' || base == 'O') {
				targetRow := 7
				if base == 'O' {
					targetRow = 19
				}
				prefix[i] = vietVowelTable[targetRow][toneIdx]
				return prefix, true
			}
		}
	}

	// 3. Xử lý dấu móc / trăng với phím 'w': aw -> ă, ow -> ơ, uw -> ư, uow -> ươ
	if lastLower == 'w' {
		// Kiểm tra cụm uo -> ươ
		for i := len(prefix) - 1; i >= 1; i-- {
			r2, t2 := findVowelRowAndTone(prefix[i])
			r1, t1 := findVowelRowAndTone(prefix[i-1])
			if r1 >= 0 && r2 >= 0 {
				b1 := unicode.ToLower(vietVowelTable[r1][0])
				b2 := unicode.ToLower(vietVowelTable[r2][0])
				if (b1 == 'u' || b1 == 'ư') && (b2 == 'o' || b2 == 'ơ') {
					uRow := 10
					if unicode.IsUpper(vietVowelTable[r1][0]) {
						uRow = 22
					}
					oRow := 8
					if unicode.IsUpper(vietVowelTable[r2][0]) {
						oRow = 20
					}
					prefix[i-1] = vietVowelTable[uRow][t1]
					prefix[i] = vietVowelTable[oRow][t2]
					return prefix, true
				}
			}
		}
		for i := len(prefix) - 1; i >= 0; i-- {
			rowIdx, toneIdx := findVowelRowAndTone(prefix[i])
			if rowIdx < 0 {
				continue
			}
			base := vietVowelTable[rowIdx][0]
			switch base {
			case 'a':
				prefix[i] = vietVowelTable[1][toneIdx]
				return prefix, true
			case 'A':
				prefix[i] = vietVowelTable[13][toneIdx]
				return prefix, true
			case 'o':
				prefix[i] = vietVowelTable[8][toneIdx]
				return prefix, true
			case 'O':
				prefix[i] = vietVowelTable[20][toneIdx]
				return prefix, true
			case 'u':
				prefix[i] = vietVowelTable[10][toneIdx]
				return prefix, true
			case 'U':
				prefix[i] = vietVowelTable[22][toneIdx]
				return prefix, true
			}
		}
	}

	// 4. Xử lý 5 phím thanh điệu Telex: s (sắc=1), f (huyền=2), r (hỏi=3), x (ngã=4), j (nặng=5), z (xóa dấu=0)
	toneMap := map[rune]int{
		's': 1,
		'f': 2,
		'r': 3,
		'x': 4,
		'j': 5,
		'z': 0,
	}
	targetTone, isToneKey := toneMap[lastLower]
	if !isToneKey {
		return word, false
	}

	// Tìm các nguyên âm trong từ (bỏ qua 'u' trong 'qu' và 'i' trong 'gi' nếu sau đó còn nguyên âm khác)
	var vowelIndices []int
	for i, r := range prefix {
		if rowIdx, _ := findVowelRowAndTone(r); rowIdx >= 0 {
			vowelIndices = append(vowelIndices, i)
		}
	}
	if len(vowelIndices) == 0 {
		return word, false
	}

	lowerPrefix := strings.ToLower(string(prefix))
	if len(vowelIndices) > 1 && (strings.HasPrefix(lowerPrefix, "qu") || strings.HasPrefix(lowerPrefix, "gi")) {
		vowelIndices = vowelIndices[1:]
	}

	// Chọn vị trí nguyên âm đặt dấu thanh chuẩn chính tả tiếng Việt
	targetIdx := vowelIndices[0]
	if len(vowelIndices) >= 2 {
		// Nếu có nguyên âm mang dấu phụ (ă, â, ê, ô, ơ, ư), ưu tiên đặt dấu thanh vào nguyên âm đó
		foundAccentVowel := false
		for _, idx := range vowelIndices {
			rIdx, _ := findVowelRowAndTone(prefix[idx])
			base := unicode.ToLower(vietVowelTable[rIdx][0])
			if base == 'ă' || base == 'â' || base == 'ê' || base == 'ô' || base == 'ơ' || base == 'ư' {
				targetIdx = idx
				foundAccentVowel = true
				break
			}
		}
		if !foundAccentVowel {
			// Nếu có phụ âm cuối hoặc 3 nguyên âm, đặt ở nguyên âm thứ 2; ngược lại đặt ở nguyên âm đầu
			lastVowelIdx := vowelIndices[len(vowelIndices)-1]
			hasTrailingConsonant := lastVowelIdx < len(prefix)-1
			if hasTrailingConsonant || len(vowelIndices) == 3 {
				targetIdx = vowelIndices[1]
			} else {
				targetIdx = vowelIndices[0]
			}
		}
	}

	rowIdx, currentTone := findVowelRowAndTone(prefix[targetIdx])
	if rowIdx < 0 {
		return word, false
	}
	// Nếu gõ phím 'z' khi chưa có dấu thanh nào thì giữ nguyên chữ 'z'
	if targetTone == 0 && currentTone == 0 {
		return word, false
	}
	// Nếu gõ lặp lại cùng một phím dấu (VD: 'ss'), trả về nguyên gốc và giữ chữ 's'
	if currentTone == targetTone && targetTone != 0 {
		prefix[targetIdx] = vietVowelTable[rowIdx][0]
		return append(prefix, last), true
	}

	// Xóa dấu thanh cũ trên các nguyên âm khác (nếu có) và đặt dấu mới vào targetIdx
	for _, idx := range vowelIndices {
		rI, _ := findVowelRowAndTone(prefix[idx])
		if rI >= 0 {
			prefix[idx] = vietVowelTable[rI][0]
		}
	}
	prefix[targetIdx] = vietVowelTable[rowIdx][targetTone]
	return prefix, true
}
