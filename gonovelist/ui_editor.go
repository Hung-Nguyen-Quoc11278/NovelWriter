package main

import (
	"fmt"
	"math"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const noneOptionLabel = "(Chưa chọn)"

// MinWidthLayout đảm bảo một widget con (như ô nhập Mục tiêu từ) luôn có chiều rộng tối thiểu cố định
// khi đặt bên trong HBox hoặc thanh điều khiển ngang, tránh bị ép nhỏ hoặc tràn chữ.
type MinWidthLayout struct {
	MinWidth float32
}

func NewMinWidthContainer(minWidth float32, obj fyne.CanvasObject) *fyne.Container {
	return container.New(&MinWidthLayout{MinWidth: minWidth}, obj)
}

func (l *MinWidthLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, obj := range objects {
		if !obj.Visible() {
			continue
		}
		obj.Move(fyne.NewPos(0, 0))
		obj.Resize(size)
	}
}

func (l *MinWidthLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w := l.MinWidth
	h := float32(0)
	for _, obj := range objects {
		if !obj.Visible() {
			continue
		}
		ms := obj.MinSize()
		if ms.Width > w {
			w = ms.Width
		}
		if ms.Height > h {
			h = ms.Height
		}
	}
	return fyne.NewSize(w, h)
}

// ResponsiveHeaderRowLayout bố trí Dòng 1 của khung soạn thảo:
//   - Nhóm 0 (Trái/Giữa - Giãn rộng): Nhãn "Cảnh:" + Ô nhập tiêu đề cảnh (titleEntry)
//   - Nhóm 1 (Phải): Nhãn "Mục tiêu từ:" + Ô nhập số từ mục tiêu (targetWordsEntry)
//   - Nhóm 2 (Phải): Cụm điều khiển cỡ chữ (A-, Cỡ chữ, A+, Mặc định)
//
// Khi cửa sổ đủ rộng, cả 3 nhóm nằm gọn trên cùng 1 hàng ngang và ô Tiêu đề cảnh tự động giãn kín phần trống.
// Khi thu hẹp cửa sổ hoặc phóng to cỡ chữ (A+), các cụm bên phải tự động xuống hàng mượt mà, tuyệt đối không đè lên ô Tiêu đề.
type ResponsiveHeaderRowLayout struct {
	TitleMinWidth      float32
	GapX               float32
	GapY               float32
	currentWidth       float32
	lastNotifiedHeight float32
	onHeightChanged    func()
}

func NewResponsiveHeaderRowLayout(titleMinWidth, gapX, gapY float32, onHeightChanged func()) *ResponsiveHeaderRowLayout {
	return &ResponsiveHeaderRowLayout{
		TitleMinWidth:   titleMinWidth,
		GapX:            gapX,
		GapY:            gapY,
		currentWidth:    720,
		onHeightChanged: onHeightChanged,
	}
}

func (l *ResponsiveHeaderRowLayout) SetCurrentWidth(w float32) {
	if w > 80 {
		l.currentWidth = w
	}
}

func (l *ResponsiveHeaderRowLayout) computeLayout(objects []fyne.CanvasObject, availWidth float32, apply bool) float32 {
	var visible []fyne.CanvasObject
	for _, obj := range objects {
		if obj != nil && obj.Visible() {
			visible = append(visible, obj)
		}
	}
	if len(visible) == 0 {
		return 0
	}
	if availWidth <= 80 {
		availWidth = l.currentWidth
		if availWidth <= 80 {
			availWidth = 720
		}
	}

	titleMinW := l.TitleMinWidth
	if ms := visible[0].MinSize().Width; ms > titleMinW {
		titleMinW = ms
	}

	// Nếu chỉ có 1 phần tử
	if len(visible) == 1 {
		h := visible[0].MinSize().Height
		if apply {
			visible[0].Move(fyne.NewPos(0, 0))
			visible[0].Resize(fyne.NewSize(availWidth, h))
		}
		return h
	}

	// Tính tổng chiều rộng của các cụm điều khiển bên phải (Mục tiêu từ + Thu phóng cỡ chữ)
	rightTotalW := float32(0)
	maxH := visible[0].MinSize().Height
	for i := 1; i < len(visible); i++ {
		ms := visible[i].MinSize()
		if i > 1 {
			rightTotalW += l.GapX
		}
		rightTotalW += ms.Width
		if ms.Height > maxH {
			maxH = ms.Height
		}
	}

	// Trường hợp 1: Đủ chiều rộng để đặt Tiêu đề cảnh + Mục tiêu từ + Cụm cỡ chữ trên cùng 1 hàng ngang
	if availWidth >= titleMinW+l.GapX+rightTotalW {
		titleW := availWidth - rightTotalW - l.GapX
		if apply {
			visible[0].Move(fyne.NewPos(0, 0))
			visible[0].Resize(fyne.NewSize(titleW, maxH))

			x := titleW + l.GapX
			for i := 1; i < len(visible); i++ {
				ms := visible[i].MinSize()
				visible[i].Move(fyne.NewPos(x, (maxH-ms.Height)/2))
				visible[i].Resize(fyne.NewSize(ms.Width, maxH))
				x += ms.Width + l.GapX
			}
		}
		return maxH
	}

	// Trường hợp 2: Không đủ chỗ cho cả 3 nhóm trên 1 dòng -> Ô Tiêu đề cảnh ưu tiên dòng đầu,
	// các cụm điều khiển bên phải tự động gói (wrap) xuống dòng kế tiếp.
	y := float32(0)
	firstRowH := visible[0].MinSize().Height
	if len(visible) >= 2 {
		secondMin := visible[1].MinSize()
		if availWidth >= titleMinW+l.GapX+secondMin.Width {
			if secondMin.Height > firstRowH {
				firstRowH = secondMin.Height
			}
			titleW := availWidth - secondMin.Width - l.GapX
			if apply {
				visible[0].Move(fyne.NewPos(0, 0))
				visible[0].Resize(fyne.NewSize(titleW, firstRowH))
				visible[1].Move(fyne.NewPos(titleW+l.GapX, 0))
				visible[1].Resize(fyne.NewSize(secondMin.Width, firstRowH))
			}
			y = firstRowH + l.GapY
			for i := 2; i < len(visible); i++ {
				ms := visible[i].MinSize()
				if apply {
					visible[i].Move(fyne.NewPos(0, y))
					visible[i].Resize(fyne.NewSize(ms.Width, ms.Height))
				}
				y += ms.Height + l.GapY
			}
			return y - l.GapY
		}
	}

	// Trường hợp 3: Cửa sổ rất hẹp -> Tiêu đề chiếm trọn dòng 1, các cụm điều khiển xếp dòng dưới
	if apply {
		visible[0].Move(fyne.NewPos(0, 0))
		visible[0].Resize(fyne.NewSize(availWidth, firstRowH))
	}
	y = firstRowH + l.GapY
	x := float32(0)
	rowH := float32(0)
	for i := 1; i < len(visible); i++ {
		ms := visible[i].MinSize()
		if x > 0 && x+ms.Width > availWidth {
			x = 0
			y += rowH + l.GapY
			rowH = 0
		}
		if apply {
			visible[i].Move(fyne.NewPos(x, y))
			visible[i].Resize(fyne.NewSize(ms.Width, ms.Height))
		}
		x += ms.Width + l.GapX
		if ms.Height > rowH {
			rowH = ms.Height
		}
	}
	return y + rowH
}

func (l *ResponsiveHeaderRowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if size.Width > 80 {
		l.currentWidth = size.Width
	}
	reqH := l.computeLayout(objects, size.Width, true)
	if size.Width > 80 && math.Abs(float64(reqH-size.Height)) > 1.0 && math.Abs(float64(reqH-l.lastNotifiedHeight)) > 1.0 {
		l.lastNotifiedHeight = reqH
		if l.onHeightChanged != nil {
			l.onHeightChanged()
		}
	} else if math.Abs(float64(reqH-size.Height)) <= 1.0 {
		l.lastNotifiedHeight = reqH
	}
}

func (l *ResponsiveHeaderRowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	reqH := l.computeLayout(objects, l.currentWidth, false)
	return fyne.NewSize(l.TitleMinWidth, reqH)
}

// ResponsiveToolbarWrapLayout bố trí Dòng 2 (Thanh công cụ Định dạng Văn bản Phong phú):
// Tự động xếp các nút (In đậm, In nghiêng, Gạch chân, Trích dẫn, Tiêu đề phụ, Ngắt cảnh, Xem trước)
// theo hàng ngang với khoảng cách đều đặn và tự động rớt dòng (Flow Wrap) khi chiều ngang hẹp
// hoặc khi người dùng phóng to cỡ chữ. Tuyệt đối không dùng Spacer âm gây đè nút.
type ResponsiveToolbarWrapLayout struct {
	GapX               float32
	GapY               float32
	PushLastRight      bool
	currentWidth       float32
	lastNotifiedHeight float32
	onHeightChanged    func()
}

func NewResponsiveToolbarWrapLayout(gapX, gapY float32, pushLastRight bool, onHeightChanged func()) *ResponsiveToolbarWrapLayout {
	return &ResponsiveToolbarWrapLayout{
		GapX:            gapX,
		GapY:            gapY,
		PushLastRight:   pushLastRight,
		currentWidth:    720,
		onHeightChanged: onHeightChanged,
	}
}

func (l *ResponsiveToolbarWrapLayout) SetCurrentWidth(w float32) {
	if w > 80 {
		l.currentWidth = w
	}
}

func (l *ResponsiveToolbarWrapLayout) computeWrap(objects []fyne.CanvasObject, availWidth float32, apply bool) (float32, float32) {
	var visible []fyne.CanvasObject
	for _, obj := range objects {
		if obj != nil && obj.Visible() {
			visible = append(visible, obj)
		}
	}
	if len(visible) == 0 {
		return 0, 0
	}
	if availWidth <= 80 {
		availWidth = l.currentWidth
		if availWidth <= 80 {
			availWidth = 720
		}
	}

	type itemPlacement struct {
		obj    fyne.CanvasObject
		rowIdx int
		x      float32
		w      float32
		h      float32
	}

	placements := make([]itemPlacement, 0, len(visible))
	var rowHeights []float32

	x := float32(0)
	currentRow := 0
	rowMaxH := float32(0)
	minSingleWidth := float32(120)

	for i, obj := range visible {
		ms := obj.MinSize()
		if ms.Width > minSingleWidth {
			minSingleWidth = ms.Width
		}

		// Nếu phần tử không vừa dòng hiện tại thì xuống dòng mới
		if x > 0 && x+ms.Width > availWidth {
			rowHeights = append(rowHeights, rowMaxH)
			currentRow++
			x = 0
			rowMaxH = 0
		}

		posX := x
		// Nút cuối cùng ("Xem trước Định dạng") được đẩy gọn về mép phải nếu dòng hiện tại còn đủ chỗ trống,
		// nhưng luôn đảm bảo posX >= x nên không bao giờ đè lên các nút định dạng bên trái.
		if i == len(visible)-1 && l.PushLastRight && availWidth-ms.Width > x {
			posX = availWidth - ms.Width
		}

		placements = append(placements, itemPlacement{
			obj:    obj,
			rowIdx: currentRow,
			x:      posX,
			w:      ms.Width,
			h:      ms.Height,
		})

		x += ms.Width + l.GapX
		if ms.Height > rowMaxH {
			rowMaxH = ms.Height
		}
	}
	rowHeights = append(rowHeights, rowMaxH)

	// Tính tọa độ Y cho từng dòng
	rowY := make([]float32, len(rowHeights))
	totalH := float32(0)
	for r, h := range rowHeights {
		rowY[r] = totalH
		totalH += h
		if r < len(rowHeights)-1 {
			totalH += l.GapY
		}
	}

	if apply {
		for _, p := range placements {
			rh := rowHeights[p.rowIdx]
			offsetY := rowY[p.rowIdx] + (rh-p.h)/2
			p.obj.Move(fyne.NewPos(p.x, offsetY))
			p.obj.Resize(fyne.NewSize(p.w, rh))
		}
	}

	return minSingleWidth, totalH
}

func (l *ResponsiveToolbarWrapLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if size.Width > 80 {
		l.currentWidth = size.Width
	}
	_, reqH := l.computeWrap(objects, size.Width, true)
	if size.Width > 80 && math.Abs(float64(reqH-size.Height)) > 1.0 && math.Abs(float64(reqH-l.lastNotifiedHeight)) > 1.0 {
		l.lastNotifiedHeight = reqH
		if l.onHeightChanged != nil {
			l.onHeightChanged()
		}
	} else if math.Abs(float64(reqH-size.Height)) <= 1.0 {
		l.lastNotifiedHeight = reqH
	}
}

func (l *ResponsiveToolbarWrapLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minW, reqH := l.computeWrap(objects, l.currentWidth, false)
	return fyne.NewSize(minW, reqH)
}

// EditorPanel quản lý trình soạn thảo văn xuôi, bộ tự động lưu (auto-save),
// bảng Ngữ cảnh Cảnh (Nhân vật, Địa điểm, Vật phẩm, Sự kiện, POV, Trạng thái) và Ghi chú bên lề.
type EditorPanel struct {
	store     *Store
	window    fyne.Window
	projectID int64
	onSaved   func()
	onAudioExport func()

	mu          sync.Mutex
	activeScene *Scene
	loading     bool
	saveTimer   *time.Timer

	// Các khung nhập văn bản trung tâm (Hỗ trợ gõ Tiếng Việt Telex & Khóa cứng con trỏ dòng)
	titleEntry       *VietnameseEntry
	summaryEntry     *VietnameseEntry
	proseEntry       *VietnameseEntry
	targetWordsEntry *VietnameseEntry

	// Thanh tiến độ, đếm từ thời gian thực và điều khiển thu phóng cỡ chữ (Zoom)
	sceneWordLabel   *widget.Label
	chapterWordLabel *widget.Label
	sceneProgress    *widget.ProgressBar
	chapterProgress  *widget.ProgressBar
	saveStateLabel   *widget.Label
	fontSizeLabel    *widget.Label
	onZoomDelta      func(delta float32)
	onZoomReset      func()

	// Bảng Ngữ cảnh Cảnh bên phải (Trạng thái, POV, Địa điểm, Nhân vật, Vật phẩm, Sự kiện, Ghi chú)
	statusSelect    *widget.Select
	povSelect       *widget.Select
	locationSelect  *widget.Select
	charactersCheck *widget.CheckGroup
	propsCheck      *widget.CheckGroup
	eventsCheck     *widget.CheckGroup
	sideNotesEntry  *VietnameseEntry
	inspectorTabs   *container.AppTabs
	splitContainer  *container.Split

	// Các vùng chứa bố cục để đồng bộ kích thước tức thì khi phóng to cửa sổ (Maximize / Scale)
	headerTopRow       *fyne.Container
	headerTopRowLayout *ResponsiveHeaderRowLayout
	formattingToolbar  *fyne.Container
	toolbarWrapLayout  *ResponsiveToolbarWrapLayout
	summaryRow         *fyne.Container
	headerForm         *fyne.Container
	footerStats        *fyne.Container
	centerEditor       *fyne.Container
	editorBodyStack    *fyne.Container
	richPreview        *widget.RichText
	richPreviewScroll  *container.Scroll
	isPreviewMode      bool
	contextVBox        *fyne.Container
	contextScroll      *container.Scroll
	notesTabContent    *fyne.Container

	// Bộ nhớ đệm danh sách thực thể của tác phẩm hiện tại
	characters []Character
	locations  []Location
	props      []Prop
	events     []Event
}

// NewEditorPanel khởi tạo khung soạn thảo văn xuôi và bảng Ngữ cảnh Cảnh bằng tiếng Việt.
func NewEditorPanel(store *Store, window fyne.Window, projectID int64, onSaved func()) *EditorPanel {
	ep := &EditorPanel{
		store:     store,
		window:    window,
		projectID: projectID,
		onSaved:   onSaved,
	}
	ep.buildUI()
	ep.ReloadMetadataOptions(projectID)
	return ep
}

func (ep *EditorPanel) buildUI() {
	ep.titleEntry = NewVietnameseEntry()
	ep.titleEntry.SetPlaceHolder("Nhập tiêu đề cảnh (hỗ trợ gõ Tiếng Việt)...")
	ep.titleEntry.SetOnChangedCallback(func(_ string) {
		ep.scheduleAutoSave()
	})

	ep.summaryEntry = NewVietnameseEntry()
	ep.summaryEntry.SetPlaceHolder("Tóm tắt ngắn gọn nội dung hoặc mục đích kịch tính của cảnh...")
	ep.summaryEntry.SetOnChangedCallback(func(_ string) {
		ep.scheduleAutoSave()
	})

	ep.targetWordsEntry = NewVietnameseEntry()
	ep.targetWordsEntry.SetPlaceHolder("1200")
	ep.targetWordsEntry.SetOnChangedCallback(func(_ string) {
		ep.scheduleAutoSave()
	})

	ep.proseEntry = NewVietnameseMultiLineEntry()
	ep.proseEntry.SetPlaceHolder("Bắt đầu viết nội dung cảnh bằng tiếng Việt tại đây... Hỗ trợ định dạng In đậm (**chữ**), In nghiêng (*chữ*), Gạch chân (<u>chữ</u>) và Trích dẫn (> dòng).")
	ep.proseEntry.SetOnChangedCallback(func(_ string) {
		if ep.isPreviewMode && ep.richPreview != nil {
			ep.richPreview.ParseMarkdown(ConvertRichProseToFyneMarkdown(ep.proseEntry.Text))
		}
		ep.updateLiveWordCounts()
		ep.scheduleAutoSave()
	})

	ep.richPreview = widget.NewRichTextFromMarkdown("")
	ep.richPreview.Wrapping = fyne.TextWrapWord
	ep.richPreviewScroll = container.NewVScroll(container.NewPadded(ep.richPreview))
	ep.richPreviewScroll.Hide()

	ep.sceneWordLabel = widget.NewLabel("Cảnh: 0 / 1200 từ")
	ep.chapterWordLabel = widget.NewLabel("Chương: 0 / 3000 từ")
	ep.saveStateLabel = widget.NewLabel("Đã lưu")
	ep.fontSizeLabel = widget.NewLabelWithStyle("Cỡ chữ: 18px (129%)", fyne.TextAlignCenter, fyne.TextStyle{Monospace: true})

	zoomOutBtn := widget.NewButton("A-", func() {
		if ep.onZoomDelta != nil {
			ep.onZoomDelta(-EditorFontSizeStep)
		}
	})
	zoomOutBtn.Importance = widget.LowImportance

	zoomInBtn := widget.NewButton("A+", func() {
		if ep.onZoomDelta != nil {
			ep.onZoomDelta(EditorFontSizeStep)
		}
	})
	zoomInBtn.Importance = widget.LowImportance

	zoomResetBtn := widget.NewButton("Mặc định", func() {
		if ep.onZoomReset != nil {
			ep.onZoomReset()
		}
	})
	zoomResetBtn.Importance = widget.LowImportance

	ep.sceneProgress = widget.NewProgressBar()
	ep.chapterProgress = widget.NewProgressBar()

	// Các điều khiển trong bảng Ngữ cảnh Cảnh
	ep.statusSelect = widget.NewSelect(AllSceneStatuses(), func(_ string) {
		ep.scheduleAutoSave()
	})

	ep.povSelect = widget.NewSelect([]string{noneOptionLabel}, func(_ string) {
		ep.scheduleAutoSave()
	})

	ep.locationSelect = widget.NewSelect([]string{noneOptionLabel}, func(_ string) {
		ep.scheduleAutoSave()
	})

	ep.charactersCheck = widget.NewCheckGroup([]string{}, func(_ []string) {
		ep.scheduleAutoSave()
	})

	ep.propsCheck = widget.NewCheckGroup([]string{}, func(_ []string) {
		ep.scheduleAutoSave()
	})

	ep.eventsCheck = widget.NewCheckGroup([]string{}, func(_ []string) {
		ep.scheduleAutoSave()
	})

	ep.sideNotesEntry = NewVietnameseMultiLineEntry()
	ep.sideNotesEntry.SetPlaceHolder("Ghi chú bên lề, ý tưởng đột xuất, câu thoại nháp hoặc tư liệu lịch sử cho cảnh này...")
	ep.sideNotesEntry.SetOnChangedCallback(func(_ string) {
		ep.scheduleAutoSave()
	})

	refreshHeaderHeight := func() {
		if ep.headerForm != nil {
			ep.headerForm.Refresh()
		}
		if ep.centerEditor != nil {
			ep.centerEditor.Refresh()
		}
	}

	// =========================================================================
	// DÒNG 1 (ROW 1): Tiêu đề cảnh + Mục tiêu số từ + Điều khiển cỡ chữ (A-, Cỡ chữ, A+, Mặc định)
	// =========================================================================
	titleGroup := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("Cảnh:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil,
		ep.titleEntry,
	)

	targetWordsGroup := container.NewHBox(
		widget.NewLabel("Mục tiêu từ:"),
		NewMinWidthContainer(84, ep.targetWordsEntry),
	)

	zoomControlsGroup := container.NewHBox(
		widget.NewSeparator(),
		zoomOutBtn,
		ep.fontSizeLabel,
		zoomInBtn,
		zoomResetBtn,
	)

	ep.headerTopRowLayout = NewResponsiveHeaderRowLayout(220, 8, 6, refreshHeaderHeight)
	ep.headerTopRow = container.New(
		ep.headerTopRowLayout,
		titleGroup,
		targetWordsGroup,
		zoomControlsGroup,
	)

	// =========================================================================
	// DÒNG 2 (ROW 2): Thanh công cụ Định dạng Văn bản Phong phú (Rich Text Toolbar)
	// Sử dụng ResponsiveToolbarWrapLayout để các nút định dạng tự động xuống dòng
	// khi thu hẹp cửa sổ hoặc tăng cỡ chữ, tuyệt đối không chồng lấp hay đè lên nhau.
	// =========================================================================
	boldBtn := widget.NewButton("B In đậm", func() {
		ep.FormatBold()
	})
	boldBtn.Importance = widget.LowImportance

	italicBtn := widget.NewButton("I In nghiêng", func() {
		ep.FormatItalic()
	})
	italicBtn.Importance = widget.LowImportance

	underlineBtn := widget.NewButton("U Gạch chân", func() {
		ep.FormatUnderline()
	})
	underlineBtn.Importance = widget.LowImportance

	quoteBtn := widget.NewButton("❝ Trích dẫn", func() {
		ep.FormatBlockquote()
	})
	quoteBtn.Importance = widget.LowImportance

	headingBtn := widget.NewButton("H Tiêu đề phụ", func() {
		ep.FormatSubHeading()
	})
	headingBtn.Importance = widget.LowImportance

	dividerBtn := widget.NewButton("― Ngắt cảnh", func() {
		ep.InsertSceneDivider()
	})
	dividerBtn.Importance = widget.LowImportance

	var previewToggleBtn *widget.Button
	previewToggleBtn = widget.NewButton("👁️ Xem trước Định dạng", func() {
		PlayUIClickSound()
		ep.isPreviewMode = !ep.isPreviewMode
		if ep.isPreviewMode {
			ep.richPreview.ParseMarkdown(ConvertRichProseToFyneMarkdown(ep.proseEntry.Text))
			ep.proseEntry.Hide()
			ep.richPreviewScroll.Show()
			previewToggleBtn.SetText("✏️ Quay lại Soạn thảo")
			previewToggleBtn.Importance = widget.HighImportance
		} else {
			ep.richPreviewScroll.Hide()
			ep.proseEntry.Show()
			previewToggleBtn.SetText("👁️ Xem trước Định dạng")
			previewToggleBtn.Importance = widget.LowImportance
			if ep.window != nil && ep.window.Canvas() != nil {
				ep.window.Canvas().Focus(ep.proseEntry)
			}
		}
		previewToggleBtn.Refresh()
		if ep.formattingToolbar != nil {
			ep.formattingToolbar.Refresh()
		}
		if ep.editorBodyStack != nil {
			ep.editorBodyStack.Refresh()
		}
	})
	previewToggleBtn.Importance = widget.LowImportance

	audioBtn := widget.NewButtonWithIcon("🎧 Xuất Audio", theme.MediaPlayIcon(), func() {
		PlayUIClickSound()
		if ep.onAudioExport != nil {
			ep.onAudioExport()
		}
	})
	audioBtn.Importance = widget.LowImportance

	ep.toolbarWrapLayout = NewResponsiveToolbarWrapLayout(6, 6, true, refreshHeaderHeight)
	ep.formattingToolbar = container.New(
		ep.toolbarWrapLayout,
		widget.NewLabelWithStyle("Định dạng:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		boldBtn,
		italicBtn,
		underlineBtn,
		quoteBtn,
		headingBtn,
		dividerBtn,
		previewToggleBtn,
		audioBtn,
	)

	// =========================================================================
	// DÒNG 3 (ROW 3): Tóm tắt cảnh (chiếm trọn chiều ngang độc lập)
	// =========================================================================
	ep.summaryRow = container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("Tóm tắt:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil,
		ep.summaryEntry,
	)

	// Xếp chồng dọc (VBox) toàn bộ các dòng điều khiển đầu trình soạn thảo với khoảng đệm (Padded)
	// và đường kẻ phân cách rõ ràng để không bao giờ xảy ra va chạm bố cục khi phóng to / thu nhỏ.
	ep.headerForm = container.NewPadded(
		container.NewVBox(
			ep.headerTopRow,
			widget.NewSeparator(),
			ep.formattingToolbar,
			widget.NewSeparator(),
			ep.summaryRow,
			widget.NewSeparator(),
		),
	)

	ep.footerStats = container.NewVBox(
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			container.NewBorder(nil, nil, ep.sceneWordLabel, nil, ep.sceneProgress),
			container.NewBorder(nil, nil, ep.chapterWordLabel, nil, ep.chapterProgress),
		),
		container.NewHBox(
			widget.NewLabelWithStyle("Trạng thái lưu SQLite:", fyne.TextAlignLeading, fyne.TextStyle{Italic: true}),
			ep.saveStateLabel,
			layout.NewSpacer(),
		),
	)

	ep.editorBodyStack = container.NewStack(ep.proseEntry, ep.richPreviewScroll)
	ep.centerEditor = container.NewBorder(ep.headerForm, ep.footerStats, nil, nil, ep.editorBodyStack)

	// Bố cục thanh bên phải: Tab "Ngữ cảnh Cảnh" (Mở rộng với Nhân vật, Vật phẩm & Sự kiện)
	openHubBtn := widget.NewButton("⚙️ Mở Trung Tâm Thế Giới & Thẻ...", func() {
		ShowWorldBuildingHub(ep.store, ep.window, ep.projectID, func() {
			ep.ReloadMetadataOptions(ep.projectID)
			if ep.activeScene != nil {
				_ = ep.LoadScene(ep.activeScene.ID, ep.projectID)
			}
		})
	})

	ep.contextVBox = container.NewVBox(
		openHubBtn,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("TRẠNG THÁI BIÊN TẬP", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.statusSelect,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("GÓC NHÌN TRẦN THUẬT (POV)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.povSelect,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("ĐỊA ĐIỂM DIỄN RA CẢNH", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.locationSelect,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("👤 NHÂN VẬT TRONG CẢNH", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.charactersCheck,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("🧭 VẬT PHẨM TRONG CẢNH (PROPS)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.propsCheck,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("⏳ SỰ KIỆN TRONG CẢNH (EVENTS)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.eventsCheck,
	)
	ep.contextScroll = container.NewVScroll(ep.contextVBox)

	ep.notesTabContent = container.NewBorder(
		widget.NewLabelWithStyle("GHI CHÚ BÊN LỀ & NHÁP Ý TƯỞNG", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		ep.sideNotesEntry,
	)

	ep.inspectorTabs = container.NewAppTabs(
		container.NewTabItem("Ngữ cảnh Cảnh", ep.contextScroll),
		container.NewTabItem("Ghi chú bên lề", ep.notesTabContent),
	)

	ep.splitContainer = container.NewHSplit(ep.centerEditor, ep.inspectorTabs)
	ep.splitContainer.Offset = 0.68
}

// Container trả về đối tượng CanvasObject gốc của khung soạn thảo.
func (ep *EditorPanel) Container() fyne.CanvasObject {
	return ep.splitContainer
}

// ForceLayoutRefresh buộc thanh chia HSplit, trình soạn thảo trung tâm và thanh bên Ngữ cảnh Cảnh
// tính toán lại ranh giới (bounds) và làm mới bố cục ngay khi cửa sổ chính được phóng to (Maximize) hoặc đổi tỷ lệ.
func (ep *EditorPanel) ForceLayoutRefresh() {
	if ep.splitContainer == nil {
		return
	}

	savedRow, savedCol := 0, 0
	if ep.proseEntry != nil {
		savedRow, savedCol = ep.proseEntry.GetLockedCursor()
	}

	if ep.centerEditor != nil {
		availW := ep.centerEditor.Size().Width
		if availW > 80 {
			if ep.headerTopRowLayout != nil {
				ep.headerTopRowLayout.SetCurrentWidth(availW)
			}
			if ep.toolbarWrapLayout != nil {
				ep.toolbarWrapLayout.SetCurrentWidth(availW)
			}
		}
	}

	if ep.headerTopRow != nil {
		ep.headerTopRow.Refresh()
	}
	if ep.formattingToolbar != nil {
		ep.formattingToolbar.Refresh()
	}
	if ep.summaryRow != nil {
		ep.summaryRow.Refresh()
	}
	if ep.headerForm != nil {
		ep.headerForm.Refresh()
	}
	if ep.footerStats != nil {
		ep.footerStats.Refresh()
	}
	if ep.centerEditor != nil {
		ep.centerEditor.Refresh()
	}
	if ep.contextVBox != nil {
		ep.contextVBox.Refresh()
	}
	if ep.contextScroll != nil {
		ep.contextScroll.Refresh()
	}
	if ep.notesTabContent != nil {
		ep.notesTabContent.Refresh()
	}
	if ep.inspectorTabs != nil {
		ep.inspectorTabs.Refresh()
	}
	ep.splitContainer.Refresh()

	if ep.proseEntry != nil {
		ep.proseEntry.RestoreLockedCursor(savedRow, savedCol)
	}
}

// BindZoomHandlers kết nối các nút phóng to / thu nhỏ cỡ chữ trên thanh công cụ với bộ quản lý Theme.
func (ep *EditorPanel) BindZoomHandlers(onDelta func(delta float32), onReset func()) {
	ep.onZoomDelta = onDelta
	ep.onZoomReset = onReset
}

// SetOnAudioExport gán hàm xử lý sự kiện khi người dùng nhấn nút xuất Audio trên thanh công cụ.
func (ep *EditorPanel) SetOnAudioExport(fn func()) {
	ep.onAudioExport = fn
}

// UpdateFontSizeIndicator cập nhật nhãn hiển thị cỡ chữ hiện tại (px & %) và làm mới bố cục dòng tiêu đề.
func (ep *EditorPanel) UpdateFontSizeIndicator(sizePx float32, zoomPercent int) {
	if ep.fontSizeLabel != nil {
		ep.fontSizeLabel.SetText(fmt.Sprintf("Cỡ chữ: %.0fpx (%d%%)", sizePx, zoomPercent))
	}
	if ep.headerTopRow != nil {
		ep.headerTopRow.Refresh()
	}
	if ep.formattingToolbar != nil {
		ep.formattingToolbar.Refresh()
	}
	if ep.headerForm != nil {
		ep.headerForm.Refresh()
	}
}

// SetDistractionFree ẩn hoặc hiện thanh Ngữ cảnh Cảnh & Ghi chú bên phải.
func (ep *EditorPanel) SetDistractionFree(enabled bool) {
	if enabled {
		ep.inspectorTabs.Hide()
		ep.splitContainer.Offset = 1.0
	} else {
		ep.inspectorTabs.Show()
		ep.splitContainer.Offset = 0.68
	}
	ep.splitContainer.Refresh()
}

// ReloadMetadataOptions tải lại danh sách Nhân vật, Địa điểm, Vật phẩm và Sự kiện từ SQLite.
func (ep *EditorPanel) ReloadMetadataOptions(projectID int64) {
	ep.projectID = projectID
	chars, _ := ep.store.ListCharacters(projectID)
	locs, _ := ep.store.ListLocations(projectID)
	props, _ := ep.store.ListProps(projectID)
	events, _ := ep.store.ListEvents(projectID)

	ep.characters = chars
	ep.locations = locs
	ep.props = props
	ep.events = events

	povOpts := []string{noneOptionLabel}
	charNames := make([]string, 0, len(chars))
	for _, c := range chars {
		povOpts = append(povOpts, c.Name)
		charNames = append(charNames, c.Name)
	}

	locOpts := []string{noneOptionLabel}
	for _, l := range locs {
		locOpts = append(locOpts, l.Name)
	}

	propNames := make([]string, 0, len(props))
	for _, p := range props {
		propNames = append(propNames, p.Name)
	}

	eventTitles := make([]string, 0, len(events))
	for _, ev := range events {
		eventTitles = append(eventTitles, ev.Title)
	}

	ep.mu.Lock()
	wasLoading := ep.loading
	ep.loading = true
	ep.mu.Unlock()

	ep.povSelect.Options = povOpts
	ep.povSelect.Refresh()

	ep.locationSelect.Options = locOpts
	ep.locationSelect.Refresh()

	ep.charactersCheck.Options = charNames
	ep.charactersCheck.Refresh()

	ep.propsCheck.Options = propNames
	ep.propsCheck.Refresh()

	ep.eventsCheck.Options = eventTitles
	ep.eventsCheck.Refresh()

	ep.mu.Lock()
	ep.loading = wasLoading
	ep.mu.Unlock()
}

// LoadScene nạp dữ liệu một Cảnh (bao gồm Nhân vật, Vật phẩm, Sự kiện gắn kèm) từ SQLite vào trình soạn thảo.
func (ep *EditorPanel) LoadScene(sceneID int64, projectID int64) error {
	ep.FlushPendingSave()

	if ep.projectID != projectID {
		ep.ReloadMetadataOptions(projectID)
	}

	sc, err := ep.store.GetScene(sceneID)
	if err != nil {
		return err
	}

	ep.mu.Lock()
	ep.loading = true
	ep.activeScene = sc
	ep.mu.Unlock()

	ep.titleEntry.SetText(sc.Title)
	ep.summaryEntry.SetText(sc.Summary)
	ep.targetWordsEntry.SetText(fmt.Sprintf("%d", sc.TargetWords))
	ep.proseEntry.SetText(sc.Content)
	if ep.richPreview != nil {
		ep.richPreview.ParseMarkdown(ConvertRichProseToFyneMarkdown(sc.Content))
	}
	ep.sideNotesEntry.SetText(sc.SideNotes)
	ep.statusSelect.SetSelected(string(NormalizeStatus(sc.Status)))

	// Gán nhân vật POV
	povName := noneOptionLabel
	if sc.POVCharacterID != nil {
		povName = ep.findCharacterNameByID(*sc.POVCharacterID)
	}
	ep.povSelect.SetSelected(povName)

	// Gán địa điểm
	locName := noneOptionLabel
	if sc.LocationID != nil {
		locName = ep.findLocationNameByID(*sc.LocationID)
	}
	ep.locationSelect.SetSelected(locName)

	// Gán các nhân vật có mặt trong cảnh
	var selectedCharNames []string
	for _, id := range sc.CharacterIDs {
		name := ep.findCharacterNameByID(id)
		if name != noneOptionLabel {
			selectedCharNames = append(selectedCharNames, name)
		}
	}
	ep.charactersCheck.SetSelected(selectedCharNames)

	// Gán các Vật phẩm trong cảnh (Props in Scene)
	var selectedPropNames []string
	for _, id := range sc.PropIDs {
		name := ep.findPropNameByID(id)
		if name != "" {
			selectedPropNames = append(selectedPropNames, name)
		}
	}
	ep.propsCheck.SetSelected(selectedPropNames)

	// Gán các Sự kiện trong cảnh (Events in Scene)
	var selectedEventTitles []string
	for _, id := range sc.EventIDs {
		title := ep.findEventTitleByID(id)
		if title != "" {
			selectedEventTitles = append(selectedEventTitles, title)
		}
	}
	ep.eventsCheck.SetSelected(selectedEventTitles)

	ep.mu.Lock()
	ep.loading = false
	ep.mu.Unlock()

	ep.updateLiveWordCounts()
	ep.saveStateLabel.SetText("Đã đồng bộ với SQLite")
	return nil
}

func (ep *EditorPanel) updateLiveWordCounts() {
	ep.mu.Lock()
	sc := ep.activeScene
	ep.mu.Unlock()
	if sc == nil {
		return
	}

	liveWords := CountWords(ep.proseEntry.Text)
	target := sc.TargetWords
	if _, err := fmt.Sscanf(ep.targetWordsEntry.Text, "%d", &target); err != nil || target <= 0 {
		target = 1200
	}

	ep.sceneWordLabel.SetText(fmt.Sprintf("Cảnh: %d / %d từ", liveWords, target))
	ratio := float64(liveWords) / float64(target)
	if ratio > 1.0 {
		ratio = 1.0
	}
	ep.sceneProgress.SetValue(ratio)

	chWords, chTarget, err := ep.store.GetChapterWordProgress(sc.ChapterID)
	if err == nil {
		adjustedChWords := chWords - sc.WordCount + liveWords
		if adjustedChWords < 0 {
			adjustedChWords = liveWords
		}
		if chTarget <= 0 {
			chTarget = 3000
		}
		ep.chapterWordLabel.SetText(fmt.Sprintf("Chương: %d / %d từ", adjustedChWords, chTarget))
		chRatio := float64(adjustedChWords) / float64(chTarget)
		if chRatio > 1.0 {
			chRatio = 1.0
		}
		ep.chapterProgress.SetValue(chRatio)
	}
}

func (ep *EditorPanel) scheduleAutoSave() {
	ep.mu.Lock()
	defer ep.mu.Unlock()

	if ep.loading || ep.activeScene == nil {
		return
	}

	ep.saveStateLabel.SetText("Đang chờ tự động lưu...")
	if ep.saveTimer != nil {
		ep.saveTimer.Stop()
	}
	ep.saveTimer = time.AfterFunc(900*time.Millisecond, func() {
		// Nếu người dùng vẫn đang gõ liên tục, hoãn thêm một nhịp ngắn để tránh tranh chấp trạng thái con trỏ
		if ep.proseEntry != nil && ep.proseEntry.IsActivelyTyping(650*time.Millisecond) {
			ep.scheduleAutoSave()
			return
		}
		ep.FlushPendingSave()
	})
}

// FlushPendingSave ghi ngay lập tức mọi thay đổi của Cảnh (bao gồm Vật phẩm & Sự kiện trong cảnh) xuống SQLite
// mà vẫn bảo toàn tuyệt đối vị trí con trỏ (caret) trên dòng đang chọn của người dùng.
func (ep *EditorPanel) FlushPendingSave() {
	ep.mu.Lock()
	if ep.loading || ep.activeScene == nil {
		ep.mu.Unlock()
		return
	}
	if ep.saveTimer != nil {
		ep.saveTimer.Stop()
		ep.saveTimer = nil
	}

	savedRow, savedCol := 0, 0
	if ep.proseEntry != nil {
		savedRow, savedCol = ep.proseEntry.GetLockedCursor()
	}

	sc := *ep.activeScene
	sc.Title = ep.titleEntry.Text
	sc.Summary = ep.summaryEntry.Text
	sc.Content = ep.proseEntry.Text
	sc.SideNotes = ep.sideNotesEntry.Text
	if ep.statusSelect.Selected != "" {
		sc.Status = NormalizeStatus(SceneStatus(ep.statusSelect.Selected))
	}

	var target int
	if _, err := fmt.Sscanf(ep.targetWordsEntry.Text, "%d", &target); err == nil && target > 0 {
		sc.TargetWords = target
	}

	sc.POVCharacterID = ep.findCharacterIDByName(ep.povSelect.Selected)
	sc.LocationID = ep.findLocationIDByName(ep.locationSelect.Selected)

	var charIDs []int64
	for _, name := range ep.charactersCheck.Selected {
		if idPtr := ep.findCharacterIDByName(name); idPtr != nil {
			charIDs = append(charIDs, *idPtr)
		}
	}
	sc.CharacterIDs = charIDs

	var propIDs []int64
	for _, name := range ep.propsCheck.Selected {
		if idPtr := ep.findPropIDByName(name); idPtr != nil {
			propIDs = append(propIDs, *idPtr)
		}
	}
	sc.PropIDs = propIDs

	var eventIDs []int64
	for _, title := range ep.eventsCheck.Selected {
		if idPtr := ep.findEventIDByTitle(title); idPtr != nil {
			eventIDs = append(eventIDs, *idPtr)
		}
	}
	sc.EventIDs = eventIDs
	ep.mu.Unlock()

	if err := ep.store.UpdateScene(&sc); err == nil {
		ep.mu.Lock()
		if ep.activeScene != nil && ep.activeScene.ID == sc.ID {
			ep.activeScene.WordCount = sc.WordCount
			ep.activeScene.TargetWords = sc.TargetWords
			ep.activeScene.CharacterIDs = sc.CharacterIDs
			ep.activeScene.PropIDs = sc.PropIDs
			ep.activeScene.EventIDs = sc.EventIDs
		}
		ep.mu.Unlock()

		ep.saveStateLabel.SetText(fmt.Sprintf("Đã tự động lưu lúc %s", time.Now().Format("15:04:05")))
		ep.updateLiveWordCounts()
		if ep.onSaved != nil {
			ep.onSaved()
		}
		// Khôi phục lại tọa độ con trỏ đã khóa nếu bất kỳ lệnh Refresh nào làm trôi dòng đang chọn
		if ep.proseEntry != nil {
			ep.proseEntry.RestoreLockedCursor(savedRow, savedCol)
		}
	}
}

func (ep *EditorPanel) findCharacterIDByName(name string) *int64 {
	if name == "" || name == noneOptionLabel {
		return nil
	}
	for _, c := range ep.characters {
		if c.Name == name {
			id := c.ID
			return &id
		}
	}
	return nil
}

func (ep *EditorPanel) findCharacterNameByID(id int64) string {
	for _, c := range ep.characters {
		if c.ID == id {
			return c.Name
		}
	}
	return noneOptionLabel
}

func (ep *EditorPanel) findLocationIDByName(name string) *int64 {
	if name == "" || name == noneOptionLabel {
		return nil
	}
	for _, l := range ep.locations {
		if l.Name == name {
			id := l.ID
			return &id
		}
	}
	return nil
}

func (ep *EditorPanel) findLocationNameByID(id int64) string {
	for _, l := range ep.locations {
		if l.ID == id {
			return l.Name
		}
	}
	return noneOptionLabel
}

func (ep *EditorPanel) findPropIDByName(name string) *int64 {
	for _, p := range ep.props {
		if p.Name == name {
			id := p.ID
			return &id
		}
	}
	return nil
}

func (ep *EditorPanel) findPropNameByID(id int64) string {
	for _, p := range ep.props {
		if p.ID == id {
			return p.Name
		}
	}
	return ""
}

func (ep *EditorPanel) findEventIDByTitle(title string) *int64 {
	for _, ev := range ep.events {
		if ev.Title == title {
			id := ev.ID
			return &id
		}
	}
	return nil
}

func (ep *EditorPanel) findEventTitleByID(id int64) string {
	for _, ev := range ep.events {
		if ev.ID == id {
			return ev.Title
		}
	}
	return ""
}

// ==================== CÁC THAO TÁC ĐỊNH DẠNG VĂN BẢN (RICH TEXT ACTIONS) ====================

// FormatBold bọc văn bản đang chọn (hoặc từ tại con trỏ) bằng cặp thẻ In đậm (**văn bản**).
func (ep *EditorPanel) FormatBold() {
	if ep.proseEntry == nil {
		return
	}
	PlayUIClickSound()
	ep.proseEntry.WrapSelectionOrInsert("**", "**", "văn bản in đậm")
	if ep.window != nil && ep.window.Canvas() != nil && !ep.isPreviewMode {
		ep.window.Canvas().Focus(ep.proseEntry)
	}
}

// FormatItalic bọc văn bản đang chọn (hoặc từ tại con trỏ) bằng cặp thẻ In nghiêng (*văn bản*).
func (ep *EditorPanel) FormatItalic() {
	if ep.proseEntry == nil {
		return
	}
	PlayUIClickSound()
	ep.proseEntry.WrapSelectionOrInsert("*", "*", "văn bản in nghiêng")
	if ep.window != nil && ep.window.Canvas() != nil && !ep.isPreviewMode {
		ep.window.Canvas().Focus(ep.proseEntry)
	}
}

// FormatUnderline bọc văn bản đang chọn (hoặc từ tại con trỏ) bằng cặp thẻ Gạch chân / Ghi chú (<u>văn bản</u>).
func (ep *EditorPanel) FormatUnderline() {
	if ep.proseEntry == nil {
		return
	}
	PlayUIClickSound()
	ep.proseEntry.WrapSelectionOrInsert("<u>", "</u>", "ghi chú gạch chân")
	if ep.window != nil && ep.window.Canvas() != nil && !ep.isPreviewMode {
		ep.window.Canvas().Focus(ep.proseEntry)
	}
}

// FormatBlockquote bật/tắt định dạng Trích dẫn (> ) cho dòng văn bản hiện tại.
func (ep *EditorPanel) FormatBlockquote() {
	if ep.proseEntry == nil {
		return
	}
	PlayUIClickSound()
	ep.proseEntry.ToggleLinePrefix("> ", "Đoạn trích dẫn...")
	if ep.window != nil && ep.window.Canvas() != nil && !ep.isPreviewMode {
		ep.window.Canvas().Focus(ep.proseEntry)
	}
}

// FormatSubHeading bật/tắt định dạng Tiêu đề phụ (### ) cho dòng văn bản hiện tại.
func (ep *EditorPanel) FormatSubHeading() {
	if ep.proseEntry == nil {
		return
	}
	PlayUIClickSound()
	ep.proseEntry.ToggleLinePrefix("### ", "Tiêu đề phân đoạn")
	if ep.window != nil && ep.window.Canvas() != nil && !ep.isPreviewMode {
		ep.window.Canvas().Focus(ep.proseEntry)
	}
}

// InsertSceneDivider chèn dấu ngắt cảnh (* * *) vào vị trí dòng hiện tại.
func (ep *EditorPanel) InsertSceneDivider() {
	if ep.proseEntry == nil {
		return
	}
	PlayUIClickSound()
	ep.proseEntry.InsertBlockSnippet("* * *")
	if ep.window != nil && ep.window.Canvas() != nil && !ep.isPreviewMode {
		ep.window.Canvas().Focus(ep.proseEntry)
	}
}
