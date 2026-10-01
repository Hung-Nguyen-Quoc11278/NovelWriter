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

// EditorPanel manages the prose editor, debounced auto-save engine, metadata sidebar,
// side-notes scratchpad, and real-time Scene/Chapter word count progress bars.
type EditorPanel struct {
	store          *Store
	window         fyne.Window
	onSaved        func()
	currentProjID  int64
	currentScene   *Scene
	suppressEvents bool

	// Debounced auto-save synchronization
	saveMu    sync.Mutex
	saveTimer *time.Timer

	// Lookup caches for Character & Location widgets
	characters []Character
	locations  []Location

	// UI Widgets
	rootSplit       *container.Split
	inspectorBox    *fyne.Container
	sceneTitleEntry *widget.Entry
	saveStateLabel  *widget.Label
	proseEntry      *widget.Entry
	richPreview     *widget.RichText
	sideNotesEntry  *widget.Entry

	// Metadata & Context controls
	statusSelect    *widget.Select
	povSelect       *widget.Select
	locationSelect  *widget.Select
	charCheckGroup  *widget.CheckGroup
	targetWordEntry *widget.Entry

	// Progress & Word Count widgets
	sceneWordLabel   *widget.Label
	sceneProgress    *widget.ProgressBar
	chapterWordLabel *widget.Label
	chapterProgress  *widget.ProgressBar
}

// NewEditorPanel initializes the center editor and right context inspector.
func NewEditorPanel(store *Store, w fyne.Window, onSaved func()) *EditorPanel {
	ep := &EditorPanel{
		store:   store,
		window:  w,
		onSaved: onSaved,
	}
	ep.buildUI()
	return ep
}

// Container returns the top-level Fyne CanvasObject for embedding in the main split view.
func (ep *EditorPanel) Container() fyne.CanvasObject {
	return ep.rootSplit
}

func (ep *EditorPanel) buildUI() {
	ep.sceneTitleEntry = widget.NewEntry()
	ep.sceneTitleEntry.SetPlaceHolder("Scene Title")
	ep.sceneTitleEntry.OnChanged = func(val string) {
		if ep.suppressEvents || ep.currentScene == nil {
			return
		}
		ep.currentScene.Title = val
		ep.scheduleAutoSave()
	}

	ep.saveStateLabel = widget.NewLabel("Saved to SQLite")
	ep.saveStateLabel.TextStyle = fyne.TextStyle{Monospace: true}

	// Formatting insert helpers for Markdown-aware RichText
	boldBtn := widget.NewButton("B", func() { ep.insertSnippet("**bold**") })
	italicBtn := widget.NewButton("I", func() { ep.insertSnippet("*italic*") })
	h2Btn := widget.NewButton("H2", func() { ep.insertSnippet("\n## Subheading\n") })
	quoteBtn := widget.NewButton("Quote", func() { ep.insertSnippet("\n> ") })
	breakBtn := widget.NewButton("* * *", func() { ep.insertSnippet("\n\n* * *\n\n") })

	headerBar := container.NewBorder(
		nil, nil,
		container.NewHBox(boldBtn, italicBtn, h2Btn, quoteBtn, breakBtn),
		ep.saveStateLabel,
		ep.sceneTitleEntry,
	)

	// Multi-line prose editor
	ep.proseEntry = widget.NewMultiLineEntry()
	ep.proseEntry.Wrapping = fyne.TextWrapWord
	ep.proseEntry.SetPlaceHolder("Begin writing your scene prose...")
	ep.proseEntry.OnChanged = func(text string) {
		if ep.suppressEvents || ep.currentScene == nil {
			return
		}
		ep.currentScene.Content = text
		ep.richPreview.ParseMarkdown(text)
		ep.updateLiveWordCounts()
		ep.scheduleAutoSave()
	}

	ep.richPreview = widget.NewRichTextFromMarkdown("")
	ep.richPreview.Wrapping = fyne.TextWrapWord

	editorTabs := container.NewAppTabs(
		container.NewTabItem("Write Prose", ep.proseEntry),
		container.NewTabItem("Typeset Preview", container.NewVScroll(ep.richPreview)),
	)

	// Bottom Word Count & Progress Status Bar
	ep.sceneWordLabel = widget.NewLabel("Scene: 0 / 1200 words")
	ep.sceneWordLabel.TextStyle = fyne.TextStyle{Monospace: true}
	ep.sceneProgress = widget.NewProgressBar()

	ep.chapterWordLabel = widget.NewLabel("Chapter: 0 / 3000 words")
	ep.chapterWordLabel.TextStyle = fyne.TextStyle{Monospace: true}
	ep.chapterProgress = widget.NewProgressBar()

	progressFooter := container.NewGridWithColumns(2,
		container.NewVBox(ep.sceneWordLabel, ep.sceneProgress),
		container.NewVBox(ep.chapterWordLabel, ep.chapterProgress),
	)

	centerPane := container.NewBorder(
		container.NewVBox(headerBar, widget.NewSeparator()),
		container.NewVBox(widget.NewSeparator(), progressFooter),
		nil, nil,
		editorTabs,
	)

	// Right-Hand Context & Metadata Mapping + Side Notes Panel
	ep.statusSelect = widget.NewSelect(AllStatuses(), func(val string) {
		if ep.suppressEvents || ep.currentScene == nil {
			return
		}
		ep.currentScene.Status = SceneStatus(val)
		ep.scheduleAutoSave()
	})

	ep.povSelect = widget.NewSelect([]string{"(None)"}, func(val string) {
		if ep.suppressEvents || ep.currentScene == nil {
			return
		}
		ep.currentScene.POVCharacterID = ep.findCharacterIDByName(val)
		ep.scheduleAutoSave()
	})

	ep.locationSelect = widget.NewSelect([]string{"(None)"}, func(val string) {
		if ep.suppressEvents || ep.currentScene == nil {
			return
		}
		ep.currentScene.LocationID = ep.findLocationIDByName(val)
		ep.scheduleAutoSave()
	})

	ep.charCheckGroup = widget.NewCheckGroup([]string{}, func(selected []string) {
		if ep.suppressEvents || ep.currentScene == nil {
			return
		}
		var ids []int64
		for _, name := range selected {
			if idPtr := ep.findCharacterIDByName(name); idPtr != nil {
				ids = append(ids, *idPtr)
			}
		}
		ep.currentScene.CharacterIDs = ids
		ep.scheduleAutoSave()
	})

	ep.targetWordEntry = widget.NewEntry()
	ep.targetWordEntry.SetPlaceHolder("1200")
	ep.targetWordEntry.OnChanged = func(val string) {
		if ep.suppressEvents || ep.currentScene == nil {
			return
		}
		var target int
		if _, err := fmt.Sscanf(val, "%d", &target); err == nil && target > 0 {
			ep.currentScene.TargetWords = target
			ep.updateLiveWordCounts()
			ep.scheduleAutoSave()
		}
	}

	metaForm := container.NewVBox(
		widget.NewLabelWithStyle("Scene Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.statusSelect,
		widget.NewLabelWithStyle("Point of View (POV)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.povSelect,
		widget.NewLabelWithStyle("Setting / Location", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.locationSelect,
		widget.NewLabelWithStyle("Scene Word Target", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.targetWordEntry,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Characters in Scene", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ep.charCheckGroup,
		layout.NewSpacer(),
	)

	ep.sideNotesEntry = widget.NewMultiLineEntry()
	ep.sideNotesEntry.Wrapping = fyne.TextWrapWord
	ep.sideNotesEntry.SetPlaceHolder("Scratchpad for scene continuity notes, research fragments, and dialogue ideas...")
	ep.sideNotesEntry.OnChanged = func(notes string) {
		if ep.suppressEvents || ep.currentScene == nil {
			return
		}
		ep.currentScene.SideNotes = notes
		ep.scheduleAutoSave()
	}

	inspectorTabs := container.NewAppTabs(
		container.NewTabItem("Context & Cast", container.NewVScroll(metaForm)),
		container.NewTabItem("Side Notes", ep.sideNotesEntry),
	)

	ep.inspectorBox = container.NewMax(inspectorTabs)
	ep.rootSplit = container.NewHSplit(centerPane, ep.inspectorBox)
	ep.rootSplit.Offset = 0.70
}

func (ep *EditorPanel) insertSnippet(snippet string) {
	if ep.currentScene == nil {
		return
	}
	ep.proseEntry.SetText(ep.proseEntry.Text + snippet)
}

// SetDistractionFree hides or reveals the right-hand Context & Side Notes inspector.
func (ep *EditorPanel) SetDistractionFree(enabled bool) {
	if enabled {
		ep.inspectorBox.Hide()
		ep.rootSplit.Offset = 1.0
	} else {
		ep.inspectorBox.Show()
		ep.rootSplit.Offset = 0.70
	}
	ep.rootSplit.Refresh()
}

// ReloadMetadataOptions refreshes the available Characters and Locations for the active Project.
func (ep *EditorPanel) ReloadMetadataOptions(projectID int64) {
	ep.currentProjID = projectID
	chars, _ := ep.store.ListCharacters(projectID)
	locs, _ := ep.store.ListLocations(projectID)
	ep.characters = chars
	ep.locations = locs

	povOpts := []string{"(None)"}
	charNames := make([]string, 0, len(chars))
	for _, c := range chars {
		povOpts = append(povOpts, c.Name)
		charNames = append(charNames, c.Name)
	}

	locOpts := []string{"(None)"}
	for _, l := range locs {
		locOpts = append(locOpts, l.Name)
	}

	ep.suppressEvents = true
	ep.povSelect.Options = povOpts
	ep.povSelect.Refresh()
	ep.locationSelect.Options = locOpts
	ep.locationSelect.Refresh()
	ep.charCheckGroup.Options = charNames
	ep.charCheckGroup.Refresh()
	ep.suppressEvents = false
}

// LoadScene populates the editor and metadata widgets with the selected Scene from SQLite.
func (ep *EditorPanel) LoadScene(projectID int64, sceneID int64) {
	ep.FlushPendingSave()
	ep.ReloadMetadataOptions(projectID)

	sc, err := ep.store.GetScene(sceneID)
	if err != nil {
		return
	}
	ep.currentScene = sc

	ep.suppressEvents = true
	ep.sceneTitleEntry.SetText(sc.Title)
	ep.proseEntry.SetText(sc.Content)
	ep.richPreview.ParseMarkdown(sc.Content)
	ep.sideNotesEntry.SetText(sc.SideNotes)
	ep.statusSelect.SetSelected(string(sc.Status))
	ep.targetWordEntry.SetText(fmt.Sprintf("%d", sc.TargetWords))

	if sc.POVCharacterID != nil {
		ep.povSelect.SetSelected(ep.findCharacterNameByID(*sc.POVCharacterID))
	} else {
		ep.povSelect.SetSelected("(None)")
	}

	if sc.LocationID != nil {
		ep.locationSelect.SetSelected(ep.findLocationNameByID(*sc.LocationID))
	} else {
		ep.locationSelect.SetSelected("(None)")
	}

	var selectedChars []string
	for _, cid := range sc.CharacterIDs {
		name := ep.findCharacterNameByID(cid)
		if name != "(None)" {
			selectedChars = append(selectedChars, name)
		}
	}
	ep.charCheckGroup.SetSelected(selectedChars)
	ep.suppressEvents = false

	ep.updateLiveWordCounts()
	ep.saveStateLabel.SetText("Saved to SQLite")
}

// scheduleAutoSave resets the 750ms debounce timer and persists changes when typing pauses.
func (ep *EditorPanel) scheduleAutoSave() {
	ep.saveMu.Lock()
	defer ep.saveMu.Unlock()

	ep.saveStateLabel.SetText("Unsaved edits...")
	if ep.saveTimer != nil {
		ep.saveTimer.Stop()
	}

	ep.saveTimer = time.AfterFunc(750*time.Millisecond, func() {
		ep.FlushPendingSave()
	})
}

// FlushPendingSave immediately writes the active Scene to SQLite if a timer is active.
func (ep *EditorPanel) FlushPendingSave() {
	ep.saveMu.Lock()
	if ep.saveTimer != nil {
		ep.saveTimer.Stop()
		ep.saveTimer = nil
	}
	sc := ep.currentScene
	ep.saveMu.Unlock()

	if sc == nil {
		return
	}

	if err := ep.store.UpdateScene(sc); err == nil {
		ep.saveStateLabel.SetText(fmt.Sprintf("Auto-saved %s", time.Now().Format("15:04:05")))
		ep.updateLiveWordCounts()
		if ep.onSaved != nil {
			ep.onSaved()
		}
	}
}

func (ep *EditorPanel) updateLiveWordCounts() {
	if ep.currentScene == nil {
		return
	}
	words := CountWords(ep.currentScene.Content)
	ep.currentScene.WordCount = words

	target := ep.currentScene.TargetWords
	if target <= 0 {
		target = 1200
	}
	ratio := float64(words) / float64(target)
	if ratio > 1.0 {
		ratio = 1.0
	}
	ep.sceneWordLabel.SetText(fmt.Sprintf("Scene: %d / %d words (%.0f%%)", words, target, ratio*100))
	ep.sceneProgress.SetValue(ratio)

	ch, chWords, err := ep.store.GetChapter(ep.currentScene.ChapterID)
	if err == nil && ch != nil {
		chTarget := ch.TargetWords
		if chTarget <= 0 {
			chTarget = 3000
		}
		chRatio := float64(chWords) / float64(chTarget)
		if chRatio > 1.0 {
			chRatio = 1.0
		}
		ep.chapterWordLabel.SetText(fmt.Sprintf("Chapter: %d / %d words (%.0f%%)", chWords, chTarget, chRatio*100))
		ep.chapterProgress.SetValue(chRatio)
	}
}

func (ep *EditorPanel) findCharacterIDByName(name string) *int64 {
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
	return "(None)"
}

func (ep *EditorPanel) findLocationIDByName(name string) *int64 {
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
	return "(None)"
}
