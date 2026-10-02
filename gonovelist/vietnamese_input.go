package main

import (
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// GlobalTelexEnabled mặc định TẮT (false) để nhường quyền hoàn toàn cho bộ gõ hệ thống (Fcitx5 / IBus / Unikey),
// tránh xung đột 2 bộ gõ cùng lúc gây lỗi biến dạng chữ ("lại phải" -> "lẫi phi", "phaỉ").
// Người dùng chỉ bật tùy chọn này trên thanh công cụ nếu máy hoàn toàn không có Fcitx5/IBus.
var GlobalTelexEnabled = false

// ConfigureVietnameseFont tự động cấu hình biến môi trường IME cho Linux/Wayland (Fcitx5)
// và thiết lập phông chữ hệ thống hỗ trợ đầy đủ Unicode Tiếng Việt UTF-8 trước khi khởi tạo Fyne.
func ConfigureVietnameseFont() {
	if runtime.GOOS == "linux" {
		if os.Getenv("XMODIFIERS") == "" {
			_ = os.Setenv("XMODIFIERS", "@im=fcitx")
		}
		if os.Getenv("GTK_IM_MODULE") == "" {
			_ = os.Setenv("GTK_IM_MODULE", "fcitx")
		}
		if os.Getenv("QT_IM_MODULE") == "" {
			_ = os.Setenv("QT_IM_MODULE", "fcitx")
		}
		lc := os.Getenv("LC_CTYPE")
		if lc == "" || lc == "C" || lc == "POSIX" {
			lang := os.Getenv("LANG")
			if strings.Contains(strings.ToUpper(lang), "UTF-8") || strings.Contains(strings.ToUpper(lang), "UTF8") {
				_ = os.Setenv("LC_CTYPE", lang)
			} else {
				_ = os.Setenv("LC_CTYPE", "en_US.UTF-8")
			}
		}
	}

	if os.Getenv("FYNE_FONT") != "" {
		return
	}
	candidateFonts := []string{
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
		"/usr/share/fonts/noto/NotoSans-Regular.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
		"/usr/share/fonts/liberation/LiberationSans-Regular.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
		"/usr/share/fonts/gnu-free/FreeSans.ttf",
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
		"/System/Library/Fonts/Supplemental/Arial.ttf",
		"/Library/Fonts/Arial.ttf",
		`C:\Windows\Fonts\segoeui.ttf`,
		`C:\Windows\Fonts\arial.ttf`,
		`C:\Windows\Fonts\tahoma.ttf`,
	}

	for _, path := range candidateFonts {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			_ = os.Setenv("FYNE_FONT", path)
			return
		}
	}
}

// DecodeUTF8Runes giải mã chuỗi UTF-8 thành mảng rune chuẩn xác thông qua gói unicode/utf8.
// Tuyệt đối không dùng len(str) hay cắt chuỗi theo byte index để tránh làm hỏng ký tự Tiếng Việt đa byte (2-4 bytes).
func DecodeUTF8Runes(s string) []rune {
	if s == "" {
		return nil
	}
	runeCount := utf8.RuneCountInString(s)
	runes := make([]rune, 0, runeCount)
	for len(s) > 0 {
		r, width := utf8.DecodeRuneInString(s)
		if r == utf8.RuneError && width == 1 {
			s = s[1:]
			continue
		}
		runes = append(runes, r)
		s = s[width:]
	}
	return runes
}

// UTF8RuneLength trả về số lượng ký tự Unicode (rune) thực sự thay vì số byte của chuỗi.
func UTF8RuneLength(s string) int {
	return utf8.RuneCountInString(s)
}

// NormalizeCombiningMarksNFC hợp nhất các dấu tổ hợp Unicode NFD (U+0300..U+036F) bị rò rỉ
// từ luồng Fcitx5 IME vào đúng nguyên âm đứng ngay trước nó theo chuẩn Unicode dựng sẵn (NFC).
func NormalizeCombiningMarksNFC(s string) (string, bool) {
	if s == "" {
		return s, false
	}
	var out []rune
	changed := false
	rest := s
	for len(rest) > 0 {
		r, width := utf8.DecodeRuneInString(rest)
		rest = rest[width:]

		if act, isCombining := mapCombiningDiacritic(r); isCombining {
			if len(out) > 0 {
				lastRune := out[len(out)-1]
				if composedRune, ok := applyCombiningToSingleRune(lastRune, act); ok {
					out[len(out)-1] = composedRune
					changed = true
					continue
				}
			}
			// Nếu dấu tổ hợp bị rò rỉ mà không có nguyên âm hợp lệ đứng trước, loại bỏ để tránh hỏng chuỗi
			changed = true
			continue
		}
		out = append(out, r)
	}
	if !changed {
		return s, false
	}
	return string(out), true
}

// VietnameseEntry kế thừa widget.Entry của Fyne v2, đảm bảo 2 mục tiêu cốt lõi:
//  1. Bảo toàn tuyệt đối luồng UTF-8 đa byte từ Fcitx5/IBus: không can thiệp sửa đổi văn bản chồng lấn
//     lên bộ đệm Preedit/Commit của Fcitx5, sử dụng unicode/utf8 cho mọi tính toán độ dài và vị trí.
//  2. Khóa cứng dòng con trỏ (lockedRow, lockedCol) khi nhấp chuột xuống dòng mới (MouseDown/MouseUp/Tapped)
//     và khi Auto-Save chạy ngầm, ngăn con trỏ nhảy ngược về cuối dòng trên.
type VietnameseEntry struct {
	widget.Entry
	mu                sync.Mutex
	lockedRow         int
	lockedCol         int
	hasLockedCursor   bool
	composingInternal bool
	systemIMEDetected bool
	lastTypedAt       time.Time
	userOnChanged     func(string)
}

// NewVietnameseEntry khởi tạo ô nhập văn bản 1 dòng chuẩn UTF-8.
func NewVietnameseEntry() *VietnameseEntry {
	e := &VietnameseEntry{}
	e.ExtendBaseWidget(e)
	e.installInternalOnChanged()
	return e
}

// NewVietnameseMultiLineEntry khởi tạo khung soạn thảo nhiều dòng chuẩn UTF-8, chống nhảy con trỏ.
func NewVietnameseMultiLineEntry() *VietnameseEntry {
	e := &VietnameseEntry{}
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapWord
	e.ExtendBaseWidget(e)
	e.installInternalOnChanged()
	return e
}

// NewVietEntry là bí danh tương thích cho NewVietnameseEntry.
func NewVietEntry() *VietnameseEntry {
	return NewVietnameseEntry()
}

// NewVietMultiLineEntry là bí danh tương thích cho NewVietnameseMultiLineEntry.
func NewVietMultiLineEntry() *VietnameseEntry {
	return NewVietnameseMultiLineEntry()
}

// SetEntryOnChanged đăng ký callback OnChanged an toàn cho VietnameseEntry.
func SetEntryOnChanged(entry *VietnameseEntry, onChanged func(string)) {
	if entry == nil {
		return
	}
	entry.SetOnChangedCallback(onChanged)
}

// AttachVietnameseTelex gắn callback OnChanged an toàn cho VietnameseEntry.
func AttachVietnameseTelex(entry *VietnameseEntry, onChanged func(string)) {
	if entry == nil {
		return
	}
	entry.SetOnChangedCallback(onChanged)
}

func (e *VietnameseEntry) installInternalOnChanged() {
	e.Entry.OnChanged = func(text string) {
		e.mu.Lock()
		if e.composingInternal {
			e.mu.Unlock()
			return
		}
		cb := e.userOnChanged
		e.mu.Unlock()

		if cb != nil {
			cb(text)
		}
	}
}

// SetOnChangedCallback thiết lập hàm lắng nghe thay đổi văn bản.
func (e *VietnameseEntry) SetOnChangedCallback(cb func(string)) {
	e.mu.Lock()
	e.userOnChanged = cb
	e.mu.Unlock()
	e.installInternalOnChanged()
}

// IsActivelyTyping kiểm tra xem người dùng có đang gõ phím/nhấp chuột trong khoảng windowDuration hay không.
func (e *VietnameseEntry) IsActivelyTyping(windowDuration time.Duration) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.lastTypedAt.IsZero() {
		return false
	}
	return time.Since(e.lastTypedAt) < windowDuration
}

// GetLockedCursor trả về tọa độ (CursorRow, CursorColumn) hiện tại của người dùng.
func (e *VietnameseEntry) GetLockedCursor() (int, int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.hasLockedCursor {
		return e.lockedRow, e.lockedCol
	}
	return e.Entry.CursorRow, e.Entry.CursorColumn
}

// RestoreLockedCursor khôi phục lại tọa độ con trỏ nếu một tác vụ nền (như Auto-Save) làm trôi con trỏ.
func (e *VietnameseEntry) RestoreLockedCursor(row, col int) {
	e.mu.Lock()
	if e.Entry.CursorRow != row || e.Entry.CursorColumn != col {
		e.Entry.CursorRow = row
		e.Entry.CursorColumn = col
	}
	e.lockedRow = row
	e.lockedCol = col
	e.hasLockedCursor = true
	e.mu.Unlock()
}

// SetText ghi đè widget.Entry.SetText và đồng bộ tọa độ con trỏ theo số lượng rune UTF-8.
func (e *VietnameseEntry) SetText(text string) {
	cleanText, _ := NormalizeCombiningMarksNFC(text)

	e.mu.Lock()
	e.composingInternal = true
	e.mu.Unlock()

	e.Entry.SetText(cleanText)

	e.mu.Lock()
	e.lockedRow = e.Entry.CursorRow
	e.lockedCol = e.Entry.CursorColumn
	e.hasLockedCursor = true
	e.composingInternal = false
	e.mu.Unlock()
}

// syncCursorFromUserClick ghi nhận chính xác dòng và cột mà người dùng vừa nhấp chuột chọn.
func (e *VietnameseEntry) syncCursorFromUserClick() {
	e.mu.Lock()
	e.lockedRow = e.Entry.CursorRow
	e.lockedCol = e.Entry.CursorColumn
	e.hasLockedCursor = true
	e.lastTypedAt = time.Now()
	e.mu.Unlock()
}

// MouseDown bắt sự kiện nhấn chuột xuống và khóa tọa độ dòng mới.
func (e *VietnameseEntry) MouseDown(ev *desktop.MouseEvent) {
	e.Entry.MouseDown(ev)
	e.syncCursorFromUserClick()
}

// MouseUp bắt sự kiện nhả chuột để đồng bộ chính xác CursorRow/CursorColumn sau khi nhấp chuột.
func (e *VietnameseEntry) MouseUp(ev *desktop.MouseEvent) {
	e.Entry.MouseUp(ev)
	e.syncCursorFromUserClick()
}

// Tapped bắt sự kiện nhấp chuột hoàn chỉnh.
func (e *VietnameseEntry) Tapped(ev *fyne.PointEvent) {
	e.Entry.Tapped(ev)
	e.syncCursorFromUserClick()
}

// TappedSecondary đồng bộ vị trí con trỏ khi nhấp chuột phải.
func (e *VietnameseEntry) TappedSecondary(ev *fyne.PointEvent) {
	e.Entry.TappedSecondary(ev)
	e.syncCursorFromUserClick()
}

// DragEnd đồng bộ vị trí con trỏ sau khi kéo chuột chọn vùng văn bản.
func (e *VietnameseEntry) DragEnd() {
	e.Entry.DragEnd()
	e.syncCursorFromUserClick()
}

// FocusGained bảo vệ vị trí con trỏ đã chọn khi khung soạn thảo nhận lại tiêu điểm.
func (e *VietnameseEntry) FocusGained() {
	e.mu.Lock()
	hadLock := e.hasLockedCursor
	savedRow := e.lockedRow
	savedCol := e.lockedCol
	e.mu.Unlock()

	e.Entry.FocusGained()

	if hadLock {
		e.mu.Lock()
		if e.Entry.CursorRow == 0 && e.Entry.CursorColumn == 0 && (savedRow > 0 || savedCol > 0) {
			e.Entry.CursorRow = savedRow
			e.Entry.CursorColumn = savedCol
		}
		e.mu.Unlock()
	}
}

// TypedKey chuyển tiếp nguyên vẹn sự kiện phím cho Fyne (bao gồm cả xung Backspace của Fcitx5),
// chỉ khóa tọa độ dòng để không bị nhảy về (0, 0) khi chuyển dòng.
func (e *VietnameseEntry) TypedKey(ev *fyne.KeyEvent) {
	if e.Disabled() {
		return
	}

	e.mu.Lock()
	e.lastTypedAt = time.Now()
	prevRow := e.Entry.CursorRow
	e.mu.Unlock()

	e.Entry.TypedKey(ev)
	PlayTypingSound(0, ev)

	e.mu.Lock()
	if (ev.Name == fyne.KeyReturn || ev.Name == fyne.KeyEnter) && e.MultiLine {
		if e.Entry.CursorRow <= prevRow {
			e.Entry.CursorRow = prevRow + 1
			e.Entry.CursorColumn = 0
		}
	}
	e.lockedRow = e.Entry.CursorRow
	e.lockedCol = e.Entry.CursorColumn
	e.hasLockedCursor = true
	e.mu.Unlock()
}

// TypedRune xử lý ký tự Unicode chuẩn UTF-8 từ Fcitx5/IBus hoặc bàn phím:
//  1. Nếu Fcitx5 gửi dấu tổ hợp rời (Combining Diacritical Marks U+0300..U+036F), hợp nhất thành
//     rune NFC hoàn chỉnh với nguyên âm liền trước mà không làm lệch byte offset.
//  2. Nếu Fcitx5 gửi ký tự Tiếng Việt đã dựng sẵn (r > 127 như 'ạ', 'ả', 'ể'...), tự động nhận diện
//     System IME đang hoạt động và chèn trực tiếp 1 rune nguyên vẹn vào Fyne.
//  3. Bảo vệ dòng hiện hành (rowBefore) để con trỏ không bao giờ nhảy ngược lên dòng trên.
func (e *VietnameseEntry) TypedRune(r rune) {
	if e.Disabled() {
		return
	}
	if !utf8.ValidRune(r) {
		return
	}
	PlayTypingSound(r, nil)

	e.mu.Lock()
	e.lastTypedAt = time.Now()

	// Nếu Fyne bất ngờ làm mất tọa độ dòng (về 0,0) sau khi người dùng đã nhấp chuột ở dòng dưới,
	// khôi phục ngay tọa độ đã khóa trước khi chèn ký tự.
	if e.hasLockedCursor && e.Entry.CursorRow == 0 && e.Entry.CursorColumn == 0 && (e.lockedRow > 0 || e.lockedCol > 0) {
		e.Entry.CursorRow = e.lockedRow
		e.Entry.CursorColumn = e.lockedCol
	}

	// Nếu nhận được ký tự Unicode Tiếng Việt dựng sẵn (r > 127) từ Fcitx5/IBus,
	// đánh dấu hệ thống đã có IME để không chạy bộ gõ Telex nội bộ chồng lên Fcitx5.
	if r > 127 && !isCombiningMarkRune(r) {
		e.systemIMEDetected = true
	}

	// 1. Trường hợp luồng Fcitx5 gửi dấu tổ hợp Unicode NFD rời (U+0300..U+036F):
	if combAct, isCombining := mapCombiningDiacritic(r); isCombining {
		lastRune, hasLast := e.decodeRuneBeforeCaretLocked()
		if hasLast {
			if composedRune, ok := applyCombiningToSingleRune(lastRune, combAct); ok {
				rowBefore := e.Entry.CursorRow
				colBefore := e.Entry.CursorColumn
				e.composingInternal = true
				e.mu.Unlock()

				e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
				e.Entry.TypedRune(composedRune)

				e.mu.Lock()
				e.composingInternal = false
				if e.Entry.CursorRow < rowBefore {
					e.Entry.CursorRow = rowBefore
					e.Entry.CursorColumn = colBefore
				}
				e.lockedRow = e.Entry.CursorRow
				e.lockedCol = e.Entry.CursorColumn
				e.hasLockedCursor = true
				cb := e.userOnChanged
				currentText := e.Entry.Text
				e.mu.Unlock()

				e.Entry.Refresh()
				if cb != nil {
					cb(currentText)
				}
				return
			}
		}
		e.mu.Unlock()
		return
	}

	// 2. Chỉ chạy bộ gõ Telex nội bộ khi người dùng BẬT thủ công VÀ không dùng Fcitx5
	if GlobalTelexEnabled && !e.systemIMEDetected {
		if wordRunes := e.extractWordBeforeCaretUTF8Locked(); len(wordRunes) > 0 && len(wordRunes) <= 15 {
			candidate := make([]rune, len(wordRunes)+1)
			copy(candidate, wordRunes)
			candidate[len(wordRunes)] = r
			if composedWord, changed := composeTelexWord(candidate); changed {
				e.applyTelexDiffLocked(wordRunes, composedWord)
				cb := e.userOnChanged
				currentText := e.Entry.Text
				e.mu.Unlock()

				e.Entry.Refresh()
				if cb != nil {
					cb(currentText)
				}
				return
			}
		}
	}

	// 3. Chèn trực tiếp 1 rune UTF-8 chuẩn xác từ Fcitx5 / bàn phím và giữ nguyên dòng hiện tại
	rowBefore := e.Entry.CursorRow
	colBefore := e.Entry.CursorColumn
	e.mu.Unlock()

	e.Entry.TypedRune(r)

	e.mu.Lock()
	// Ngăn lỗi nhảy ngược con trỏ lên dòng phía trên khi đang gõ trên dòng mới
	if e.Entry.CursorRow < rowBefore {
		e.Entry.CursorRow = rowBefore
		e.Entry.CursorColumn = colBefore + 1
	}
	e.lockedRow = e.Entry.CursorRow
	e.lockedCol = e.Entry.CursorColumn
	e.hasLockedCursor = true
	e.mu.Unlock()
}

// decodeRuneBeforeCaretLocked giải mã ký tự Unicode (rune) cuối cùng trước con trỏ bằng unicode/utf8.
func (e *VietnameseEntry) decodeRuneBeforeCaretLocked() (rune, bool) {
	lines := strings.Split(e.Entry.Text, "\n")
	if len(lines) == 0 {
		return 0, false
	}
	row := e.Entry.CursorRow
	if row < 0 || row >= len(lines) {
		// Khi văn bản có ngắt dòng mềm (TextWrapWord), lấy rune cuối cùng của văn bản nếu đang gõ ở cuối
		r, width := utf8.DecodeLastRuneInString(e.Entry.Text)
		if r == utf8.RuneError && width <= 1 {
			return 0, false
		}
		return r, width > 0
	}
	lineRunes := DecodeUTF8Runes(lines[row])
	col := e.Entry.CursorColumn
	if col <= 0 || col > len(lineRunes) {
		if len(lineRunes) > 0 {
			return lineRunes[len(lineRunes)-1], true
		}
		return 0, false
	}
	return lineRunes[col-1], true
}

// extractWordBeforeCaretUTF8Locked trích xuất từ Tiếng Việt ngay trước con trỏ theo đơn vị rune UTF-8.
func (e *VietnameseEntry) extractWordBeforeCaretUTF8Locked() []rune {
	lines := strings.Split(e.Entry.Text, "\n")
	if len(lines) == 0 {
		return nil
	}
	row := e.Entry.CursorRow
	if row < 0 || row >= len(lines) {
		allRunes := DecodeUTF8Runes(e.Entry.Text)
		end := len(allRunes)
		start := end
		for start > 0 && isVietnameseWordRune(allRunes[start-1]) {
			start--
		}
		return allRunes[start:end]
	}

	lineRunes := DecodeUTF8Runes(lines[row])
	col := e.Entry.CursorColumn
	if col < 0 || col > len(lineRunes) {
		col = len(lineRunes)
	}
	start := col
	for start > 0 && isVietnameseWordRune(lineRunes[start-1]) {
		start--
	}
	return lineRunes[start:col]
}

func (e *VietnameseEntry) applyTelexDiffLocked(oldWord, newWord []rune) {
	commonPrefix := 0
	maxLen := len(oldWord)
	if len(newWord) < maxLen {
		maxLen = len(newWord)
	}
	for commonPrefix < maxLen && oldWord[commonPrefix] == newWord[commonPrefix] {
		commonPrefix++
	}

	deleteCount := len(oldWord) - commonPrefix
	insertSlice := newWord[commonPrefix:]

	expectedRow := e.Entry.CursorRow
	expectedCol := e.Entry.CursorColumn - deleteCount + len(insertSlice)
	if expectedCol < 0 {
		expectedCol = len(newWord)
	}

	e.composingInternal = true
	e.mu.Unlock()

	for i := 0; i < deleteCount; i++ {
		e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	}
	for _, ch := range insertSlice {
		e.Entry.TypedRune(ch)
	}

	e.mu.Lock()
	e.composingInternal = false
	if e.Entry.CursorRow < expectedRow {
		e.Entry.CursorRow = expectedRow
		e.Entry.CursorColumn = expectedCol
	}
	e.lockedRow = e.Entry.CursorRow
	e.lockedCol = e.Entry.CursorColumn
	e.hasLockedCursor = true
}

// ---------------------------------------------------------------------------
// Bảng tra cứu Unicode Tiếng Việt NFC & Chuyển đổi Dấu Tổ Hợp NFD -> NFC
// ---------------------------------------------------------------------------

type composeActionKind int

const (
	actionNone composeActionKind = iota
	actionTone
	actionCircumflex // ^ (â, ê, ô)
	actionBreveOnly  // ă
	actionHornOnly   // ư, ơ
)

type composeAction struct {
	kind composeActionKind
	tone int // 0: Ngang, 1: Sắc, 2: Huyền, 3: Hỏi, 4: Ngã, 5: Nặng
}

func isCombiningMarkRune(r rune) bool {
	return r >= '\u0300' && r <= '\u036F'
}

func mapCombiningDiacritic(r rune) (composeAction, bool) {
	switch r {
	case '\u0301', '\u0341':
		return composeAction{kind: actionTone, tone: 1}, true
	case '\u0300', '\u0340':
		return composeAction{kind: actionTone, tone: 2}, true
	case '\u0309':
		return composeAction{kind: actionTone, tone: 3}, true
	case '\u0303', '\u0342':
		return composeAction{kind: actionTone, tone: 4}, true
	case '\u0323':
		return composeAction{kind: actionTone, tone: 5}, true
	case '\u0302':
		return composeAction{kind: actionCircumflex}, true
	case '\u0306':
		return composeAction{kind: actionBreveOnly}, true
	case '\u031B':
		return composeAction{kind: actionHornOnly}, true
	default:
		return composeAction{}, false
	}
}

func applyCombiningToSingleRune(target rune, act composeAction) (rune, bool) {
	meta, ok := runeToVowelMeta[target]
	if !ok {
		return target, false
	}
	switch act.kind {
	case actionTone:
		return makeVowelWithTone(meta.base, act.tone), true
	case actionCircumflex:
		switch meta.base {
		case 'a':
			return makeVowelWithTone('â', meta.tone), true
		case 'A':
			return makeVowelWithTone('Â', meta.tone), true
		case 'e':
			return makeVowelWithTone('ê', meta.tone), true
		case 'E':
			return makeVowelWithTone('Ê', meta.tone), true
		case 'o':
			return makeVowelWithTone('ô', meta.tone), true
		case 'O':
			return makeVowelWithTone('Ô', meta.tone), true
		}
	case actionBreveOnly:
		switch meta.base {
		case 'a':
			return makeVowelWithTone('ă', meta.tone), true
		case 'A':
			return makeVowelWithTone('Ă', meta.tone), true
		}
	case actionHornOnly:
		switch meta.base {
		case 'o':
			return makeVowelWithTone('ơ', meta.tone), true
		case 'O':
			return makeVowelWithTone('Ơ', meta.tone), true
		case 'u':
			return makeVowelWithTone('ư', meta.tone), true
		case 'U':
			return makeVowelWithTone('Ư', meta.tone), true
		}
	}
	return target, false
}

var vietVowelTable = map[rune][6]rune{
	'a': {'a', 'á', 'à', 'ả', 'ã', 'ạ'},
	'ă': {'ă', 'ắ', 'ằ', 'ẳ', 'ẵ', 'ặ'},
	'â': {'â', 'ấ', 'ầ', 'ẩ', 'ẫ', 'ậ'},
	'e': {'e', 'é', 'è', 'ẻ', 'ẽ', 'ẹ'},
	'ê': {'ê', 'ế', 'ề', 'ể', 'ễ', 'ệ'},
	'i': {'i', 'í', 'ì', 'ỉ', 'ĩ', 'ị'},
	'o': {'o', 'ó', 'ò', 'ỏ', 'õ', 'ọ'},
	'ô': {'ô', 'ố', 'ồ', 'ổ', 'ỗ', 'ộ'},
	'ơ': {'ơ', 'ớ', 'ờ', 'ở', 'ỡ', 'ợ'},
	'u': {'u', 'ú', 'ù', 'ủ', 'ũ', 'ụ'},
	'ư': {'ư', 'ứ', 'ừ', 'ử', 'ữ', 'ự'},
	'y': {'y', 'ý', 'ỳ', 'ỷ', 'ỹ', 'ỵ'},
	'A': {'A', 'Á', 'À', 'Ả', 'Ã', 'Ạ'},
	'Ă': {'Ă', 'Ắ', 'Ằ', 'Ẳ', 'Ẵ', 'Ặ'},
	'Â': {'Â', 'Ấ', 'Ầ', 'Ẩ', 'Ẫ', 'Ậ'},
	'E': {'E', 'É', 'È', 'Ẻ', 'Ẽ', 'Ẹ'},
	'Ê': {'Ê', 'Ế', 'Ề', 'Ể', 'Ễ', 'Ệ'},
	'I': {'I', 'Í', 'Ì', 'Ỉ', 'Ĩ', 'Ị'},
	'O': {'O', 'Ó', 'Ò', 'Ỏ', 'Õ', 'Ọ'},
	'Ô': {'Ô', 'Ố', 'Ồ', 'Ổ', 'Ỗ', 'Ộ'},
	'Ơ': {'Ơ', 'Ớ', 'Ờ', 'Ở', 'Ỡ', 'Ợ'},
	'U': {'U', 'Ú', 'Ù', 'Ủ', 'Ũ', 'Ụ'},
	'Ư': {'Ư', 'Ứ', 'Ừ', 'Ử', 'Ữ', 'Ự'},
	'Y': {'Y', 'Ý', 'Ỳ', 'Ỷ', 'Ỹ', 'Ỵ'},
}

type vowelMeta struct {
	base rune
	tone int
}

var runeToVowelMeta map[rune]vowelMeta

func init() {
	runeToVowelMeta = make(map[rune]vowelMeta, 144)
	for base, tones := range vietVowelTable {
		for tIdx, r := range tones {
			runeToVowelMeta[r] = vowelMeta{base: base, tone: tIdx}
		}
	}
}

func makeVowelWithTone(base rune, tone int) rune {
	if tones, ok := vietVowelTable[base]; ok && tone >= 0 && tone < 6 {
		return tones[tone]
	}
	return base
}

func isVietnameseWordRune(r rune) bool {
	return unicode.IsLetter(r)
}

func composeTelexWord(word []rune) ([]rune, bool) {
	n := len(word)
	if n < 2 {
		return word, false
	}

	lastKey := unicode.ToLower(word[n-1])
	stem := make([]rune, n-1)
	copy(stem, word[:n-1])

	if lastKey == 'd' && len(stem) == 1 {
		switch stem[0] {
		case 'd':
			stem[0] = 'đ'
			return stem, true
		case 'D':
			stem[0] = 'Đ'
			return stem, true
		}
	}

	switch lastKey {
	case 'a', 'e', 'o':
		for i := len(stem) - 1; i >= 0; i-- {
			meta, ok := runeToVowelMeta[stem[i]]
			if !ok {
				continue
			}
			if unicode.ToLower(meta.base) == lastKey {
				var newBase rune
				switch meta.base {
				case 'a':
					newBase = 'â'
				case 'A':
					newBase = 'Â'
				case 'e':
					newBase = 'ê'
				case 'E':
					newBase = 'Ê'
				case 'o':
					newBase = 'ô'
				case 'O':
					newBase = 'Ô'
				}
				if newBase != 0 {
					stem[i] = makeVowelWithTone(newBase, meta.tone)
					return repositionToneInWord(stem), true
				}
			}
		}

	case 'w':
		for i := len(stem) - 1; i >= 1; i-- {
			m2, ok2 := runeToVowelMeta[stem[i]]
			m1, ok1 := runeToVowelMeta[stem[i-1]]
			if ok1 && ok2 {
				b1 := unicode.ToLower(m1.base)
				b2 := unicode.ToLower(m2.base)
				if (b1 == 'u' || b1 == 'ư') && (b2 == 'o' || b2 == 'ơ') {
					uBase := 'ư'
					if unicode.IsUpper(m1.base) {
						uBase = 'Ư'
					}
					oBase := 'ơ'
					if unicode.IsUpper(m2.base) {
						oBase = 'Ơ'
					}
					combinedTone := m2.tone
					if m1.tone != 0 {
						combinedTone = m1.tone
					}
					stem[i-1] = makeVowelWithTone(uBase, 0)
					stem[i] = makeVowelWithTone(oBase, combinedTone)
					return repositionToneInWord(stem), true
				}
			}
		}

		for i := len(stem) - 1; i >= 0; i-- {
			meta, ok := runeToVowelMeta[stem[i]]
			if !ok {
				continue
			}
			var newBase rune
			switch meta.base {
			case 'u':
				if i > 0 && unicode.ToLower(stem[i-1]) == 'q' {
					continue
				}
				newBase = 'ư'
			case 'U':
				if i > 0 && unicode.ToLower(stem[i-1]) == 'q' {
					continue
				}
				newBase = 'Ư'
			case 'o':
				newBase = 'ơ'
			case 'O':
				newBase = 'Ơ'
			case 'a':
				newBase = 'ă'
			case 'A':
				newBase = 'Ă'
			}
			if newBase != 0 {
				stem[i] = makeVowelWithTone(newBase, meta.tone)
				return repositionToneInWord(stem), true
			}
		}
	}

	toneMap := map[rune]int{
		's': 1,
		'f': 2,
		'r': 3,
		'x': 4,
		'j': 5,
		'z': 0,
	}

	if targetTone, isToneKey := toneMap[lastKey]; isToneKey {
		vowelIndices := findTargetVowelIndices(stem)
		if len(vowelIndices) == 0 {
			return word, false
		}

		currentTone := 0
		for _, idx := range vowelIndices {
			if m, ok := runeToVowelMeta[stem[idx]]; ok && m.tone != 0 {
				currentTone = m.tone
				break
			}
		}

		if lastKey == 'z' && currentTone == 0 {
			return word, false
		}

		if targetTone != 0 && currentTone == targetTone {
			for _, idx := range vowelIndices {
				if m, ok := runeToVowelMeta[stem[idx]]; ok {
					stem[idx] = makeVowelWithTone(m.base, 0)
				}
			}
			stem = append(stem, word[n-1])
			return stem, true
		}

		for _, idx := range vowelIndices {
			if m, ok := runeToVowelMeta[stem[idx]]; ok {
				stem[idx] = makeVowelWithTone(m.base, 0)
			}
		}

		primaryIdx := selectPrimaryToneIndex(stem, vowelIndices)
		if m, ok := runeToVowelMeta[stem[primaryIdx]]; ok {
			stem[primaryIdx] = makeVowelWithTone(m.base, targetTone)
			return stem, true
		}
	}

	return word, false
}

func findTargetVowelIndices(runes []rune) []int {
	var indices []int
	for i, r := range runes {
		if _, ok := runeToVowelMeta[r]; ok {
			indices = append(indices, i)
		}
	}
	if len(indices) <= 1 {
		return indices
	}

	firstIdx := indices[0]
	if firstIdx > 0 &&
		unicode.ToLower(runes[firstIdx-1]) == 'q' &&
		unicode.ToLower(runeToVowelMeta[runes[firstIdx]].base) == 'u' {
		indices = indices[1:]
	}

	if len(indices) > 1 {
		firstIdx = indices[0]
		if firstIdx > 0 &&
			unicode.ToLower(runes[firstIdx-1]) == 'g' &&
			unicode.ToLower(runeToVowelMeta[runes[firstIdx]].base) == 'i' {
			indices = indices[1:]
		}
	}

	return indices
}

// selectPrimaryToneIndex xác định đúng nguyên âm đặt dấu theo chuẩn chính tả Tiếng Việt:
// Với nguyên âm đôi mở như "ai", "ao", "au", "ay", "ia", "ua", "ưa" (như trong "lại", "phải"),
// dấu thanh LUÔN đặt ở nguyên âm đầu tiên ('a' -> "lại", "phải", tuyệt đối không nhảy sang 'i' thành "phaỉ").
func selectPrimaryToneIndex(runes []rune, vowelIndices []int) int {
	if len(vowelIndices) == 1 {
		return vowelIndices[0]
	}

	// 1. Nếu có nguyên âm mang dấu mũ hoặc móc (ă, â, ê, ô, ơ, ư) -> ưu tiên đặt dấu lên nguyên âm đó
	for i := len(vowelIndices) - 1; i >= 0; i-- {
		idx := vowelIndices[i]
		base := unicode.ToLower(runeToVowelMeta[runes[idx]].base)
		if base == 'ă' || base == 'â' || base == 'ê' || base == 'ô' || base == 'ơ' || base == 'ư' {
			return idx
		}
	}

	// 2. Chỉ xem là có phụ âm cuối nếu ký tự sau nguyên âm cuối cùng thực sự là PHỤ ÂM (không phải nguyên âm như 'i', 'y', 'o', 'u')
	lastVowelIdx := vowelIndices[len(vowelIndices)-1]
	hasEndingConsonant := false
	for i := lastVowelIdx + 1; i < len(runes); i++ {
		if unicode.IsLetter(runes[i]) {
			if _, isVowel := runeToVowelMeta[runes[i]]; !isVowel {
				hasEndingConsonant = true
				break
			}
		}
	}

	if hasEndingConsonant || len(vowelIndices) >= 3 {
		return vowelIndices[1]
	}

	// 3. Nguyên âm đôi mở: chỉ "oa", "oe", "uy" đặt ở nguyên âm thứ 2; còn lại ("ai", "ao", "au", "ay", "ia", "ua"...) đặt ở nguyên âm thứ 1
	firstBase := unicode.ToLower(runeToVowelMeta[runes[vowelIndices[0]]].base)
	secondBase := unicode.ToLower(runeToVowelMeta[runes[vowelIndices[1]]].base)
	if (firstBase == 'o' && (secondBase == 'a' || secondBase == 'e')) ||
		(firstBase == 'u' && secondBase == 'y') {
		return vowelIndices[1]
	}

	return vowelIndices[0]
}

func repositionToneInWord(runes []rune) []rune {
	vowelIndices := findTargetVowelIndices(runes)
	if len(vowelIndices) <= 1 {
		return runes
	}

	existingTone := 0
	for _, idx := range vowelIndices {
		if m, ok := runeToVowelMeta[runes[idx]]; ok && m.tone != 0 {
			existingTone = m.tone
			break
		}
	}
	if existingTone == 0 {
		return runes
	}

	out := make([]rune, len(runes))
	copy(out, runes)
	for _, idx := range vowelIndices {
		if m, ok := runeToVowelMeta[out[idx]]; ok {
			out[idx] = makeVowelWithTone(m.base, 0)
		}
	}

	targetIdx := selectPrimaryToneIndex(out, vowelIndices)
	if m, ok := runeToVowelMeta[out[targetIdx]]; ok {
		out[targetIdx] = makeVowelWithTone(m.base, existingTone)
	}
	return out
}
