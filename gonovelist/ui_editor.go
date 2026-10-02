package main

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

const noneOptionLabel = "(Chưa chọn)"

// EditorPanel quản lý trình soạn thảo văn xuôi, bộ tự động lưu (auto-save),
// bảng Ngữ cảnh Cảnh (Nhân vật, Địa điểm, Vật phẩm, Sự kiện, POV, Trạng thái) và Ghi chú bên lề.
type EditorPanel struct {
	store     *Store
	window    fyne.Window
	projectID int64
	onSaved   func()

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
	headerForm      *fyne.Container
	footerStats     *fyne.Container
	centerEditor    *fyne.Container
	contextVBox     *fyne.Container
	contextScroll   *container.Scroll
	notesTabContent *fyne.Container

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
	ep.proseEntry.SetPlaceHolder("Bắt đầu viết nội dung cảnh bằng tiếng Việt tại đây... Mọi thay đổi sẽ được tự động lưu vào SQLite.")
	ep.proseEntry.SetOnChangedCallback(func(_ string) {
		ep.updateLiveWordCounts()
		ep.scheduleAutoSave()
	})

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

	zoomControlsBox := container.NewHBox(
		widget.NewSeparator(),
		zoomOutBtn,
		ep.fontSizeLabel,
		zoomInBtn,
		zoomResetBtn,
	)

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

	// Bố cục khu vực soạn thảo trung tâm (tách dòng tiêu đề và thanh công cụ Zoom để giảm MinSize ngang,
	// giúp thanh Ngữ cảnh Cảnh bên phải không bao giờ bị đẩy tràn khỏi màn hình khi phóng to chữ hoặc cửa sổ)
	titleRow := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("Cảnh:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewHBox(widget.NewLabel("Mục tiêu từ:"), ep.targetWordsEntry),
		ep.titleEntry,
	)

	summaryAndZoomRow := container.NewBorder(
		nil, nil,
		nil,
		zoomControlsBox,
		ep.summaryEntry,
	)

	ep.headerForm = container.NewVBox(
		titleRow,
		summaryAndZoomRow,
		widget.NewSeparator(),
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

	ep.centerEditor = container.NewBorder(ep.headerForm, ep.footerStats, nil, nil, ep.proseEntry)

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

// UpdateFontSizeIndicator cập nhật nhãn hiển thị cỡ chữ hiện tại (px & %).
func (ep *EditorPanel) UpdateFontSizeIndicator(sizePx float32, zoomPercent int) {
	if ep.fontSizeLabel != nil {
		ep.fontSizeLabel.SetText(fmt.Sprintf("Cỡ chữ: %.0fpx (%d%%)", sizePx, zoomPercent))
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
