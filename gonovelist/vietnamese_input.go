package main

import (
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// GlobalTelexEnabled bật/tắt bộ xử lý gõ Tiếng Việt (Telex / VNI / Chặn Dead-Key Fcitx5 Wayland).
var GlobalTelexEnabled = true

// GlobalVNIModeEnabled cho phép gõ cả kiểu VNI (1-9) lẫn chặn Dead-Key (' ^ ~ `) bị rò rỉ từ Fcitx5 trên Wayland.
var GlobalVNIModeEnabled = true

// ConfigureVietnameseFont tự động cấu hình biến môi trường IME cho Linux/Wayland (Fcitx5)
// và thiết lập phông chữ hệ thống hỗ trợ đầy đủ Unicode Tiếng Việt trước khi khởi tạo Fyne.
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
		// Arch Linux / Ubuntu / Fedora / Debian
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
		"/usr/share/fonts/noto/NotoSans-Regular.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
		"/usr/share/fonts/liberation/LiberationSans-Regular.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
		"/usr/share/fonts/gnu-free/FreeSans.ttf",
		// macOS
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
		"/System/Library/Fonts/Supplemental/Arial.ttf",
		"/Library/Fonts/Arial.ttf",
		// Windows
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

// VietnameseEntry là custom widget kế thừa widget.Entry của Fyne v2, khắc phục triệt để lỗi
// con trỏ (caret) bị nhảy ngược lên cuối dòng trên (ví dụ nhảy về sau chữ "gặp" ở dòng 1)
// khi người dùng nhấp chuột xuống dòng mới (dòng 2) và gõ 2-3 ký tự Tiếng Việt:
//
// Nguyên nhân gốc rễ trong Fyne v2:
//  1. Khi bật Wrapping = fyne.TextWrapWord, CursorRow và CursorColumn của Fyne là tọa độ DÒNG HIỂN THỊ
//     (visual wrapped row), KHÔNG phải số lượng ký tự xuống dòng '\n'. Mọi hàm tự tính (row, col)
//     bằng cách đếm '\n' hoặc strings.Split(text, "\n") rồi gán vào CursorRow sẽ làm con trỏ nhảy
//     ngược về cuối dòng hiển thị phía trên ngay khi gõ đến ký tự thứ 2-3 (lúc Telex kích hoạt).
//  2. Việc gọi Entry.SetText() bên trong OnChanged hoặc TypedRune làm Fyne tính toán lại toàn bộ
//     RichText rowBounds và ghi đè lại vị trí con trỏ cũ.
//
// Giải pháp kiến trúc trong VietnameseEntry:
//  - Lưu trữ bộ đệm từ đang gõ tại con trỏ (wordBuf []rune) độc lập với tọa độ '\n', xóa sạch wordBuf
//    ngay khi người dùng nhấp chuột (MouseDown, MouseUp, Tapped) để tách biệt hoàn toàn với dòng cũ.
//  - Tuyệt đối KHÔNG gọi SetText() khi đang gõ; chỉ thay thế phần hậu tố thay đổi của từ hiện tại bằng
//    sự kiện gốc của Fyne (Entry.TypedKey(KeyBackspace) + Entry.TypedRune) ngay tại con trỏ thực tế.
//  - Khóa cứng tọa độ dòng do người dùng chọn (lockedRow, lockedCol, mouseFloorRow) để ngăn chặn
//    mọi hiện tượng nhảy lùi dòng do WordWrap hoặc xung Backspace từ Fcitx5/Wayland.
type VietnameseEntry struct {
	widget.Entry
	mu              sync.Mutex
	wordBuf         []rune
	lockedRow       int
	lockedCol       int
	mouseFloorRow   int
	hasLockedCursor bool
	composing       bool
	lastTypedAt     time.Time
	userOnChanged   func(string)
}

// NewVietnameseEntry khởi tạo ô nhập văn bản 1 dòng với cơ chế khóa con trỏ & hỗ trợ Tiếng Việt.
func NewVietnameseEntry() *VietnameseEntry {
	e := &VietnameseEntry{}
	e.ExtendBaseWidget(e)
	e.installInternalOnChanged()
	return e
}

// NewVietnameseMultiLineEntry khởi tạo khung soạn thảo nhiều dòng chống nhảy con trỏ khi nhấp chuột xuống dòng.
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

// SetEntryOnChanged đăng ký callback lắng nghe thay đổi văn bản mà không phá vỡ bộ khóa con trỏ.
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
		suppress := e.composing
		cb := e.userOnChanged
		e.mu.Unlock()

		if suppress {
			return
		}
		if cb != nil {
			cb(text)
		}
	}
}

// SetOnChangedCallback thiết lập hàm callback chỉ được gọi sau khi chu kỳ ghép chữ Tiếng Việt hoàn tất.
func (e *VietnameseEntry) SetOnChangedCallback(cb func(string)) {
	e.mu.Lock()
	e.userOnChanged = cb
	e.mu.Unlock()
	e.installInternalOnChanged()
}

func (e *VietnameseEntry) fireUserOnChanged() {
	e.mu.Lock()
	cb := e.userOnChanged
	currentText := e.Entry.Text
	e.mu.Unlock()

	if cb != nil {
		cb(currentText)
	}
}

// IsActivelyTyping kiểm tra xem người dùng có vừa gõ phím hoặc nhấp chuột trong khoảng windowDuration hay không.
func (e *VietnameseEntry) IsActivelyTyping(windowDuration time.Duration) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.lastTypedAt.IsZero() {
		return false
	}
	return time.Since(e.lastTypedAt) < windowDuration
}

// GetLockedCursor trả về tọa độ (CursorRow, CursorColumn) đang được khóa bởi người dùng.
func (e *VietnameseEntry) GetLockedCursor() (int, int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.hasLockedCursor {
		return e.lockedRow, e.lockedCol
	}
	return e.Entry.CursorRow, e.Entry.CursorColumn
}

// RestoreLockedCursor khôi phục lại đúng tọa độ con trỏ nếu một tác vụ nền (như Auto-Save / Refresh) làm trôi con trỏ.
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

// SetText ghi đè widget.Entry.SetText khi nạp Cảnh mới từ SQLite, đồng thời xóa sạch bộ đệm từ cũ.
func (e *VietnameseEntry) SetText(text string) {
	e.mu.Lock()
	e.composing = true
	e.wordBuf = e.wordBuf[:0]
	e.mu.Unlock()

	e.Entry.SetText(text)

	e.mu.Lock()
	e.lockedRow = e.Entry.CursorRow
	e.lockedCol = e.Entry.CursorColumn
	e.mouseFloorRow = e.Entry.CursorRow
	e.hasLockedCursor = true
	e.composing = false
	e.mu.Unlock()
}

// syncCursorFromUserClick khóa chặt vị trí dòng và cột ngay tại nơi người dùng nhấp chuột
// và xóa bộ đệm từ của dòng cũ (ngăn tuyệt đối việc nhảy ngược về từ cuối của dòng trên).
func (e *VietnameseEntry) syncCursorFromUserClick() {
	e.mu.Lock()
	e.wordBuf = e.wordBuf[:0]
	e.lockedRow = e.Entry.CursorRow
	e.lockedCol = e.Entry.CursorColumn
	e.mouseFloorRow = e.Entry.CursorRow
	e.hasLockedCursor = true
	e.lastTypedAt = time.Now()
	e.mu.Unlock()
}

// MouseDown bắt sự kiện nhấn chuột xuống, để Fyne định vị dòng mới rồi khóa tọa độ ngay lập tức.
func (e *VietnameseEntry) MouseDown(ev *desktop.MouseEvent) {
	e.Entry.MouseDown(ev)
	e.syncCursorFromUserClick()
}

// MouseUp bắt sự kiện nhả chuột (quan trọng trong Fyne v2 vì Entry.MouseUp có thể cập nhật lại CursorRow/CursorColumn).
func (e *VietnameseEntry) MouseUp(ev *desktop.MouseEvent) {
	e.Entry.MouseUp(ev)
	e.syncCursorFromUserClick()
}

// Tapped bắt sự kiện chạm/nhấp chuột hoàn chỉnh để đảm bảo tọa độ dòng mới luôn được khóa cứng.
func (e *VietnameseEntry) Tapped(ev *fyne.PointEvent) {
	e.Entry.Tapped(ev)
	e.syncCursorFromUserClick()
}

// TappedSecondary đồng bộ vị trí con trỏ khi nhấp chuột phải.
func (e *VietnameseEntry) TappedSecondary(ev *fyne.PointEvent) {
	e.Entry.TappedSecondary(ev)
	e.syncCursorFromUserClick()
}

// DragEnd đồng bộ vị trí con trỏ sau khi người dùng kéo chuột bôi đen văn bản.
func (e *VietnameseEntry) DragEnd() {
	e.Entry.DragEnd()
	e.syncCursorFromUserClick()
}

// FocusGained giữ nguyên vị trí con trỏ đã khóa khi khung soạn thảo nhận lại tiêu điểm (focus).
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

// enforceLockedCursorBeforeTyping kiểm tra trước mỗi phím gõ: nếu Fyne hoặc bộ gõ hệ thống tự ý
// làm tụt CursorRow về dòng phía trên so với dòng người dùng đã nhấp chuột, khôi phục ngay lập tức.
func (e *VietnameseEntry) enforceLockedCursorBeforeTypingLocked() {
	if !e.hasLockedCursor {
		e.lockedRow = e.Entry.CursorRow
		e.lockedCol = e.Entry.CursorColumn
		e.hasLockedCursor = true
		return
	}
	if e.Entry.CursorRow < e.lockedRow ||
		(e.Entry.CursorRow == 0 && e.Entry.CursorColumn == 0 && (e.lockedRow > 0 || e.lockedCol > 0)) {
		e.Entry.CursorRow = e.lockedRow
		e.Entry.CursorColumn = e.lockedCol
	}
}

// TypedKey xử lý các phím điều hướng, Enter và Backspace mà KHÔNG bao giờ gọi SetText().
func (e *VietnameseEntry) TypedKey(ev *fyne.KeyEvent) {
	if e.Disabled() {
		return
	}

	e.mu.Lock()
	now := time.Now()
	timeSinceLastRune := now.Sub(e.lastTypedAt)
	e.lastTypedAt = now

	switch ev.Name {
	case fyne.KeyBackspace:
		// Khôi phục con trỏ nếu bị trôi trước khi xóa
		e.enforceLockedCursorBeforeTypingLocked()

		// Chặn xung Backspace giả mạo từ Fcitx5/IBus (gửi kèm trong vòng < 35ms khi đang gõ)
		// cố tình xóa ký tự xuống dòng khi con trỏ đang ở đầu dòng mới (CursorColumn == 0).
		if e.Entry.SelectedText() == "" &&
			e.Entry.CursorColumn == 0 &&
			e.Entry.CursorRow <= e.mouseFloorRow &&
			len(e.wordBuf) == 0 &&
			timeSinceLastRune < 35*time.Millisecond {
			e.mu.Unlock()
			return
		}

		if e.Entry.SelectedText() != "" {
			e.wordBuf = e.wordBuf[:0]
		} else if len(e.wordBuf) > 0 {
			e.wordBuf = e.wordBuf[:len(e.wordBuf)-1]
		}
		e.mu.Unlock()

		e.Entry.TypedKey(ev)

		e.mu.Lock()
		e.lockedRow = e.Entry.CursorRow
		e.lockedCol = e.Entry.CursorColumn
		if e.Entry.CursorRow < e.mouseFloorRow {
			e.mouseFloorRow = e.Entry.CursorRow
		}
		e.hasLockedCursor = true
		e.mu.Unlock()
		return

	case fyne.KeyReturn, fyne.KeyEnter:
		e.enforceLockedCursorBeforeTypingLocked()
		prevRow := e.Entry.CursorRow
		e.wordBuf = e.wordBuf[:0]
		e.mu.Unlock()

		// Sử dụng xử lý xuống dòng nguyên bản của Fyne để giữ tương thích 100% với TextWrapWord
		e.Entry.TypedKey(ev)

		e.mu.Lock()
		if e.MultiLine && e.Entry.CursorRow <= prevRow {
			e.Entry.CursorRow = prevRow + 1
			e.Entry.CursorColumn = 0
		}
		e.lockedRow = e.Entry.CursorRow
		e.lockedCol = e.Entry.CursorColumn
		e.mouseFloorRow = e.Entry.CursorRow
		e.hasLockedCursor = true
		e.mu.Unlock()
		return

	case fyne.KeyUp, fyne.KeyDown, fyne.KeyLeft, fyne.KeyRight,
		fyne.KeyHome, fyne.KeyEnd, fyne.KeyPageUp, fyne.KeyPageDown, fyne.KeyDelete:
		e.wordBuf = e.wordBuf[:0]
		e.mu.Unlock()

		e.Entry.TypedKey(ev)

		e.mu.Lock()
		e.lockedRow = e.Entry.CursorRow
		e.lockedCol = e.Entry.CursorColumn
		e.mouseFloorRow = e.Entry.CursorRow
		e.hasLockedCursor = true
		e.mu.Unlock()
		return
	}

	e.mu.Unlock()
	e.Entry.TypedKey(ev)

	e.mu.Lock()
	e.lockedRow = e.Entry.CursorRow
	e.lockedCol = e.Entry.CursorColumn
	e.hasLockedCursor = true
	e.mu.Unlock()
}

// TypedRune xử lý từng ký tự được gõ vào ngay tại vị trí con trỏ hiện hành,
// ghép dấu Tiếng Việt tại chỗ (in-place suffix replacement) và khóa chặt dòng đang chọn.
func (e *VietnameseEntry) TypedRune(r rune) {
	if e.Disabled() {
		return
	}

	// Nếu đang bôi đen văn bản, xóa vùng chọn trước
	if e.Entry.SelectedText() != "" {
		e.mu.Lock()
		e.wordBuf = e.wordBuf[:0]
		e.mu.Unlock()
		e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	}

	e.mu.Lock()
	e.lastTypedAt = time.Now()
	e.enforceLockedCursorBeforeTypingLocked()

	// 1. Kiểm tra nếu r là dấu tổ hợp Unicode NFD (Combining Diacritical Marks U+0300..U+036F)
	if combAct, isCombining := mapCombiningDiacritic(r); isCombining {
		if len(e.wordBuf) > 0 {
			if composedWord, ok := applyActionToStem(e.wordBuf, combAct, r); ok {
				e.replaceWordSuffixInPlaceLocked(composedWord)
				e.mu.Unlock()
				e.Entry.Refresh()
				e.fireUserOnChanged()
				return
			}
		}
		e.mu.Unlock()
		return
	}

	// 2. Xử lý ghép chữ Tiếng Việt (Telex / VNI / Dead-Key) trên bộ đệm từ hiện tại (e.wordBuf)
	if GlobalTelexEnabled && len(e.wordBuf) > 0 && len(e.wordBuf) <= 15 {
		if composedWord, changed := tryComposeOnWordBuf(e.wordBuf, r); changed {
			e.replaceWordSuffixInPlaceLocked(composedWord)
			e.mu.Unlock()
			e.Entry.Refresh()
			e.fireUserOnChanged()
			return
		}
	}

	// 3. Nếu r là chữ cái tiếp theo nối vào từ đã có dấu (ví dụ đã có "hoá", gõ thêm 'n' -> "hoán"),
	// kiểm tra xem có cần dịch chuyển vị trí dấu thanh trên từ hay không.
	if isVietnameseWordRune(r) && len(e.wordBuf) >= 1 && len(e.wordBuf) <= 15 {
		candidate := make([]rune, len(e.wordBuf)+1)
		copy(candidate, e.wordBuf)
		candidate[len(e.wordBuf)] = r
		rebalanced := repositionToneInWord(candidate)
		if string(rebalanced) != string(candidate) {
			e.replaceWordSuffixInPlaceLocked(rebalanced)
			e.mu.Unlock()
			e.Entry.Refresh()
			e.fireUserOnChanged()
			return
		}
	}

	// 4. Chèn ký tự thông thường bằng hàm gốc của Fyne và khóa cứng dòng hiện hành
	rowBefore := e.Entry.CursorRow
	colBefore := e.Entry.CursorColumn
	e.composing = true
	e.mu.Unlock()

	e.Entry.TypedRune(r)

	e.mu.Lock()
	e.composing = false

	// Ngăn chặn tuyệt đối lỗi Fyne tự ý nhảy lùi CursorRow về dòng trên sau khi chèn ký tự
	if e.Entry.CursorRow < rowBefore {
		e.Entry.CursorRow = rowBefore
		e.Entry.CursorColumn = colBefore + 1
	}

	if isVietnameseWordRune(r) {
		e.wordBuf = append(e.wordBuf, r)
	} else {
		e.wordBuf = e.wordBuf[:0]
	}

	e.lockedRow = e.Entry.CursorRow
	e.lockedCol = e.Entry.CursorColumn
	if e.lockedRow > e.mouseFloorRow {
		// Nếu dòng tự động ngắt xuống dòng mới (soft-wrap), nâng sàn dòng theo dòng mới
		e.mouseFloorRow = e.lockedRow
	}
	e.hasLockedCursor = true
	e.mu.Unlock()

	e.fireUserOnChanged()
}

// replaceWordSuffixInPlaceLocked chỉ xóa phần hậu tố thay đổi của từ vừa gõ (thường 1-2 ký tự)
// bằng KeyBackspace nội bộ và chèn lại các ký tự đã ghép dấu, đồng thời khóa cứng tọa độ
// (expectedRow, expectedCol) để con trỏ KHÔNG BAO GIỜ bị nhảy ngược lên dòng 1.
func (e *VietnameseEntry) replaceWordSuffixInPlaceLocked(composedWord []rune) {
	oldWord := e.wordBuf

	// Tìm tiền tố chung dài nhất giữa từ cũ và từ mới để giảm tối đa số lần xóa
	commonPrefix := 0
	maxLen := len(oldWord)
	if len(composedWord) < maxLen {
		maxLen = len(composedWord)
	}
	for commonPrefix < maxLen && oldWord[commonPrefix] == composedWord[commonPrefix] {
		commonPrefix++
	}

	deleteCount := len(oldWord) - commonPrefix
	insertSlice := composedWord[commonPrefix:]

	// Ghi nhận tọa độ dòng/cột kỳ vọng ngay trên dòng hiện tại của người dùng
	expectedRow := e.Entry.CursorRow
	if e.hasLockedCursor && expectedRow < e.lockedRow {
		expectedRow = e.lockedRow
	}
	expectedCol := e.Entry.CursorColumn - deleteCount + len(insertSlice)
	if expectedCol < 0 {
		expectedCol = len(composedWord)
	}

	e.composing = true
	e.mu.Unlock()

	for i := 0; i < deleteCount; i++ {
		e.Entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	}
	for _, ch := range insertSlice {
		e.Entry.TypedRune(ch)
	}

	e.mu.Lock()
	e.composing = false

	// KHÓA CHẶT DÒNG HIỆN HÀNH:
	// Nếu trong tích tắc xóa ký tự đầu của từ trên dòng 2, bộ ngắt dòng (TextWrapWord) hoặc
	// trạng thái nội bộ của Fyne làm tụt CursorRow về cuối dòng 1 (expectedRow - 1),
	// lập tức đặt con trỏ trở lại đúng dòng 2 (expectedRow, expectedCol).
	if e.Entry.CursorRow < expectedRow {
		e.Entry.CursorRow = expectedRow
		e.Entry.CursorColumn = expectedCol
	}

	e.wordBuf = make([]rune, len(composedWord))
	copy(e.wordBuf, composedWord)

	e.lockedRow = e.Entry.CursorRow
	e.lockedCol = e.Entry.CursorColumn
	if e.lockedRow > e.mouseFloorRow {
		e.mouseFloorRow = e.lockedRow
	}
	e.hasLockedCursor = true
}

// ---------------------------------------------------------------------------
// Bộ máy Hợp nhất Dấu Tiếng Việt (Dead-Keys, Combining Marks, Telex & VNI)
// ---------------------------------------------------------------------------

type composeActionKind int

const (
	actionNone composeActionKind = iota
	actionTone
	actionCircumflex // ^ (â, ê, ô)
	actionHornBreve  // w / 7 (ư, ơ, ă)
	actionBreveOnly  // 8 (ă)
	actionHornOnly   // 7 hoặc U+031B (ư, ơ)
	actionStrokeD    // d / 9 (đ)
)

type composeAction struct {
	kind composeActionKind
	tone int // 0: Ngang (xóa dấu), 1: Sắc, 2: Huyền, 3: Hỏi, 4: Ngã, 5: Nặng
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

func tryComposeOnWordBuf(wordBuf []rune, keyRune rune) ([]rune, bool) {
	if len(wordBuf) == 0 || len(wordBuf) > 15 {
		return nil, false
	}

	// 1. Xử lý Dead-Key (' ` ^ ~) hoặc phím số VNI (1..9, 0)
	if GlobalVNIModeEnabled {
		var act composeAction
		switch keyRune {
		case '\'', '´':
			act = composeAction{kind: actionTone, tone: 1}
		case '`':
			act = composeAction{kind: actionTone, tone: 2}
		case '~':
			act = composeAction{kind: actionTone, tone: 4}
		case '^':
			act = composeAction{kind: actionCircumflex}
		case '1':
			act = composeAction{kind: actionTone, tone: 1}
		case '2':
			act = composeAction{kind: actionTone, tone: 2}
		case '3':
			act = composeAction{kind: actionTone, tone: 3}
		case '4':
			act = composeAction{kind: actionTone, tone: 4}
		case '5':
			act = composeAction{kind: actionTone, tone: 5}
		case '0':
			act = composeAction{kind: actionTone, tone: 0}
		case '6':
			act = composeAction{kind: actionCircumflex}
		case '7':
			act = composeAction{kind: actionHornOnly}
		case '8':
			act = composeAction{kind: actionBreveOnly}
		case '9':
			act = composeAction{kind: actionStrokeD}
		}

		if act.kind != actionNone {
			if composed, ok := applyActionToStem(wordBuf, act, keyRune); ok {
				return composed, true
			}
		}
	}

	// 2. Xử lý quy tắc gõ Telex
	candidateWord := make([]rune, len(wordBuf)+1)
	copy(candidateWord, wordBuf)
	candidateWord[len(wordBuf)] = keyRune

	transformedWord, changed := composeTelexWord(candidateWord)
	if !changed {
		return nil, false
	}
	return transformedWord, true
}

func applyActionToStem(wordBuf []rune, act composeAction, rawKey rune) ([]rune, bool) {
	if len(wordBuf) == 0 {
		return nil, false
	}
	stem := make([]rune, len(wordBuf))
	copy(stem, wordBuf)

	switch act.kind {
	case actionStrokeD:
		if stem[0] == 'd' {
			stem[0] = 'đ'
			return stem, true
		}
		if stem[0] == 'D' {
			stem[0] = 'Đ'
			return stem, true
		}
		if stem[0] == 'đ' && unicode.IsPrint(rawKey) {
			stem[0] = 'd'
			stem = append(stem, rawKey)
			return stem, true
		}
		if stem[0] == 'Đ' && unicode.IsPrint(rawKey) {
			stem[0] = 'D'
			stem = append(stem, rawKey)
			return stem, true
		}

	case actionCircumflex:
		for i := len(stem) - 1; i >= 0; i-- {
			meta, ok := runeToVowelMeta[stem[i]]
			if !ok {
				continue
			}
			switch meta.base {
			case 'a':
				stem[i] = makeVowelWithTone('â', meta.tone)
				return repositionToneInWord(stem), true
			case 'A':
				stem[i] = makeVowelWithTone('Â', meta.tone)
				return repositionToneInWord(stem), true
			case 'e':
				stem[i] = makeVowelWithTone('ê', meta.tone)
				return repositionToneInWord(stem), true
			case 'E':
				stem[i] = makeVowelWithTone('Ê', meta.tone)
				return repositionToneInWord(stem), true
			case 'o':
				stem[i] = makeVowelWithTone('ô', meta.tone)
				return repositionToneInWord(stem), true
			case 'O':
				stem[i] = makeVowelWithTone('Ô', meta.tone)
				return repositionToneInWord(stem), true
			}
		}

	case actionBreveOnly:
		for i := len(stem) - 1; i >= 0; i-- {
			meta, ok := runeToVowelMeta[stem[i]]
			if !ok {
				continue
			}
			if meta.base == 'a' {
				stem[i] = makeVowelWithTone('ă', meta.tone)
				return repositionToneInWord(stem), true
			}
			if meta.base == 'A' {
				stem[i] = makeVowelWithTone('Ă', meta.tone)
				return repositionToneInWord(stem), true
			}
		}

	case actionHornOnly, actionHornBreve:
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
			switch meta.base {
			case 'u':
				if i > 0 && unicode.ToLower(stem[i-1]) == 'q' {
					continue
				}
				stem[i] = makeVowelWithTone('ư', meta.tone)
				return repositionToneInWord(stem), true
			case 'U':
				if i > 0 && unicode.ToLower(stem[i-1]) == 'q' {
					continue
				}
				stem[i] = makeVowelWithTone('Ư', meta.tone)
				return repositionToneInWord(stem), true
			case 'o':
				stem[i] = makeVowelWithTone('ơ', meta.tone)
				return repositionToneInWord(stem), true
			case 'O':
				stem[i] = makeVowelWithTone('Ơ', meta.tone)
				return repositionToneInWord(stem), true
			}
		}

	case actionTone:
		vowelIndices := findTargetVowelIndices(stem)
		if len(vowelIndices) == 0 {
			return nil, false
		}
		currentTone := 0
		for _, idx := range vowelIndices {
			if m, ok := runeToVowelMeta[stem[idx]]; ok && m.tone != 0 {
				currentTone = m.tone
				break
			}
		}
		if act.tone == 0 && currentTone == 0 {
			return nil, false
		}
		if act.tone != 0 && currentTone == act.tone && unicode.IsPrint(rawKey) && rawKey < 128 {
			for _, idx := range vowelIndices {
				if m, ok := runeToVowelMeta[stem[idx]]; ok {
					stem[idx] = makeVowelWithTone(m.base, 0)
				}
			}
			stem = append(stem, rawKey)
			return stem, true
		}

		for _, idx := range vowelIndices {
			if m, ok := runeToVowelMeta[stem[idx]]; ok {
				stem[idx] = makeVowelWithTone(m.base, 0)
			}
		}
		primaryIdx := selectPrimaryToneIndex(stem, vowelIndices)
		if m, ok := runeToVowelMeta[stem[primaryIdx]]; ok {
			stem[primaryIdx] = makeVowelWithTone(m.base, act.tone)
			return stem, true
		}
	}

	return nil, false
}

// Bảng tra cứu nguyên âm Tiếng Việt với 6 thanh điệu:
// Index: 0: Ngang, 1: Sắc (s/1), 2: Huyền (f/2), 3: Hỏi (r/3), 4: Ngã (x/4), 5: Nặng (j/5)
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

	// 1. Xử lý dd / DD -> đ / Đ ở đầu từ
	if lastKey == 'd' && len(stem) >= 1 {
		if stem[0] == 'd' && len(stem) == 1 {
			stem[0] = 'đ'
			return stem, true
		}
		if stem[0] == 'D' && len(stem) == 1 {
			stem[0] = 'Đ'
			return stem, true
		}
		if stem[0] == 'đ' && len(stem) == 1 {
			stem[0] = 'd'
			stem = append(stem, word[n-1])
			return stem, true
		}
		if stem[0] == 'Đ' && len(stem) == 1 {
			stem[0] = 'D'
			stem = append(stem, word[n-1])
			return stem, true
		}
	}

	// 2. Xử lý mũ và móc nguyên âm (aa -> â, aw -> ă, ee -> ê, oo -> ô, ow -> ơ, uw -> ư)
	switch lastKey {
	case 'a', 'e', 'o':
		for i := len(stem) - 1; i >= 0; i-- {
			meta, ok := runeToVowelMeta[stem[i]]
			if !ok {
				continue
			}
			lowerBase := unicode.ToLower(meta.base)
			if lowerBase == lastKey {
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
		// Trường hợp đặc biệt "uo" + "w" -> "ươ"
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

		// Trường hợp đơn: u -> ư, o -> ơ, a -> ă
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

	// 3. Xử lý phím dấu thanh: s (sắc), f (huyền), r (hỏi), x (ngã), j (nặng), z (xóa dấu)
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

	// 4. Tự động cân chỉnh vị trí dấu khi gõ thêm phụ âm cuối
	repositioned := repositionToneInWord(word)
	if string(repositioned) != string(word) {
		return repositioned, true
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

func selectPrimaryToneIndex(runes []rune, vowelIndices []int) int {
	if len(vowelIndices) == 1 {
		return vowelIndices[0]
	}

	for i := len(vowelIndices) - 1; i >= 0; i-- {
		idx := vowelIndices[i]
		base := unicode.ToLower(runeToVowelMeta[runes[idx]].base)
		if base == 'ă' || base == 'â' || base == 'ê' || base == 'ô' || base == 'ơ' || base == 'ư' {
			return idx
		}
	}

	lastVowelIdx := vowelIndices[len(vowelIndices)-1]
	hasEndingConsonant := lastVowelIdx < len(runes)-1
	if hasEndingConsonant || len(vowelIndices) >= 3 {
		return vowelIndices[1]
	}

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
