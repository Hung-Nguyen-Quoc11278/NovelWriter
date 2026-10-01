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

// EditorPanel quản lý trình soạn thảo văn xuôi, bộ tự động lưu (auto-save), siêu dữ liệu và ghi chú bên lề.
type EditorPanel struct {
	store     *Store
	window    fyne.Window
	projectID int64
	onSaved   func()

	mu          sync.Mutex
	activeScene *Scene
	loading     bool
	saveTimer   *time.Timer

	// Các thành phần trình soạn thảo trung tâm
	titleEntry       *widget.Entry
	summaryEntry     *widget.Entry
	proseEntry       *widget.Entry
	targetWordsEntry *widget.Entry

	// Thanh tiến độ và đếm từ thời gian thực
	sceneWordLabel   *widget.Label
	chapterWordLabel *widget.Label
	sceneProgress    *widget.ProgressBar
	chapterProgress  *widget.ProgressBar
	saveStateLabel   *widget.Label

	// Bảng ngữ cảnh & siêu dữ liệu bên phải (Nhân vật, Bối cảnh, Góc nhìn POV, Trạng thái, Ghi chú)
	statusSelect      *widget.Select
	povSelect         *widget.Select
	locationSelect    *widget.Select
	charactersCheck   *widget.CheckGroup
	sideNotesEntry    *widget.Entry
	inspectorTabs     *container.AppTabs
	splitContainer    *container.Split

	// Bộ nhớ đệm danh sách Nhân vật & Bối cảnh
	characters []Character
	locations  []Location
}

// NewEditorPanel khởi tạo khung soạn thảo văn xuôi và bảng ngữ cảnh bên phải bằng tiếng Việt.
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
	ep.titleEntry = widget.NewEntry()
	ep.titleEntry.SetPlaceHolder("Tiêu đề cảnh...")
	ep.titleEntry.OnChanged = func(_ string) { ep.scheduleAutoSave() }

	ep.summaryEntry = widget.NewEntry()
	ep.summaryEntry.SetPlaceHolder("Tóm tắt ngắn gọn nội dung hoặc mục đích kịch tính của cảnh...")
	ep.summaryEntry.OnChanged = func(_ string) { ep.scheduleAutoSave() }

	ep.targetWordsEntry = widget.NewEntry()
	ep.targetWordsEntry.SetPlaceHolder("1200")
	ep.targetWordsEntry.OnChanged = func(_ string) { ep.scheduleAutoSave() }

	ep.proseEntry = widget.NewMultiLineEntry()
	ep.proseEntry.Wrapping = fyne.TextWrapWord
	ep.proseEntry.SetPlaceHolder("Bắt đầu viết nội dung cảnh tại đây... Mọi thay đổi sẽ được tự động lưu vào SQLite.")
	ep.proseEntry.OnChanged = func(_ string) {
		ep.updateLiveWordCounts()
		ep.scheduleAutoSave()
	}

	ep.sceneWordLabel = widget.NewLabel("Cảnh: 0 / 1200 từ")
	ep.chapterWordLabel = widget.NewLabel("Chương: 0 / 3000 từ")
	ep.saveStateLabel = widget.NewLabel("Đã lưu")

	ep.sceneProgress = widget.NewProgressBar()
	ep.chapterProgress = widget.NewProgressBar()

	// Các điều khiển Siêu dữ liệu (Metadata)
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

	ep.sideNotesEntry = widget.NewMultiLineEntry()
	ep.sideNotesEntry.Wrapping = fyne.TextWrapWord
	ep.sideNotesEntry.SetPlaceHolder("Ghi chú bên lề, ý tưởng đột xuất, câu thoại nháp hoặc tư liệu lịch sử cho cảnh này...")
	ep.sideNotesEntry.OnChanged = func(_ string) {
		ep.scheduleAutoSave()
	}

	// Bố cục khu vực soạn thảo trung tâm
	headerForm := container.NewVBox(
		container.NewBorder(
			nil, nil,
			widget.NewLabelWithStyle("Cảnh:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			container.NewHBox(widget.NewLabel("Mục tiêu từ:"), ep.targetWordsEntry),
			ep.titleEntry,
		),
		ep.summaryEntry,
		widget.NewSeparator(),
	)

	footerStats := container.NewVBox(
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

	centerEditor := container.NewBorder(headerForm, footerStats, nil, nil, ep.proseEntry)

	// Bố cục thanh thanh tra Ngữ cảnh & Ghi chú bên phải
	contextTabContent := container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("TRẠNG THÁI BIÊN TẬP", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.statusSelect,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("GÓC NHÌN TRẦN THUẬT (POV)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.povSelect,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("BỐI CẢNH / ĐỊA ĐIỂM", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.locationSelect,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("NHÂN VẬT XUẤT HIỆN TRONG CẢNH", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.charactersCheck,
	))

	notesTabContent := container.NewBorder(
		widget.NewLabelWithStyle("GHI CHÚ BÊN LỀ & NHÁP Ý TƯỞNG", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		ep.sideNotesEntry,
	)

	ep.inspectorTabs = container.NewAppTabs(
		container.NewTabItem("Ngữ cảnh & Nhân vật", contextTabContent),
		container.NewTabItem("Ghi chú bên lề", notesTabContent),
	)

	ep.splitContainer = container.NewHSplit(centerEditor, ep.inspectorTabs)
	ep.splitContainer.Offset = 0.70
}

// Container trả về đối tượng CanvasObject gốc của khung soạn thảo.
func (ep *EditorPanel) Container() fyne.CanvasObject {
	return ep.splitContainer
}

// SetDistractionFree ẩn hoặc hiện thanh Ngữ cảnh & Ghi chú bên phải.
func (ep *EditorPanel) SetDistractionFree(enabled bool) {
	if enabled {
		ep.inspectorTabs.Hide()
		ep.splitContainer.Offset = 1.0
	} else {
		ep.inspectorTabs.Show()
		ep.splitContainer.Offset = 0.70
	}
	ep.splitContainer.Refresh()
}

// ReloadMetadataOptions tải lại danh sách Nhân vật và Địa điểm từ SQLite.
func (ep *EditorPanel) ReloadMetadataOptions(projectID int64) {
	ep.projectID = projectID
	chars, _ := ep.store.ListCharacters(projectID)
	locs, _ := ep.store.ListLocations(projectID)

	ep.characters = chars
	ep.locations = locs

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

	ep.loading = true
	ep.povSelect.Options = povOpts
	ep.povSelect.Refresh()

	ep.locationSelect.Options = locOpts
	ep.locationSelect.Refresh()

	ep.charactersCheck.Options = charNames
	ep.charactersCheck.Refresh()
	ep.loading = false
}

// LoadScene nạp dữ liệu một Cảnh từ SQLite vào trình soạn thảo.
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

	// Gán bối cảnh
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
	ep.saveTimer = time.AfterFunc(750*time.Millisecond, func() {
		ep.FlushPendingSave()
	})
}

// FlushPendingSave ghi ngay lập tức mọi thay đổi của Cảnh hiện tại xuống SQLite.
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
	ep.mu.Unlock()

	if err := ep.store.UpdateScene(&sc); err == nil {
		ep.mu.Lock()
		if ep.activeScene != nil && ep.activeScene.ID == sc.ID {
			ep.activeScene.WordCount = sc.WordCount
			ep.activeScene.TargetWords = sc.TargetWords
		}
		ep.mu.Unlock()

		ep.saveStateLabel.SetText(fmt.Sprintf("Đã tự động lưu lúc %s", time.Now().Format("15:04:05")))
		ep.updateLiveWordCounts()
		if ep.onSaved != nil {
			ep.onSaved()
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
