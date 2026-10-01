package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// NovelistUI coordinates the main Fyne window, hierarchical tree sidebar, and scene editor.
type NovelistUI struct {
	app             fyne.App
	window          fyne.Window
	store           *Store
	activeProject   Project
	acts            []Act
	childrenMap     map[string][]string
	nodeLookup      map[string]TreeNode
	selectedUID     string
	distractionFree bool

	// Widgets
	projectSelect *widget.Select
	tree          *widget.Tree
	sidebarBox    *fyne.Container
	mainSplit     *container.Split
	editorPanel   *EditorPanel
	statusFooter  *widget.Label
}

// NewNovelistUI constructs the complete desktop UI and binds it to the SQLite Store.
func NewNovelistUI(a fyne.App, w fyne.Window, store *Store) (*NovelistUI, error) {
	ui := &NovelistUI{
		app:         a,
		window:      w,
		store:       store,
		childrenMap: make(map[string][]string),
		nodeLookup:  make(map[string]TreeNode),
	}

	projects, err := store.ListProjects()
	if err != nil || len(projects) == 0 {
		return nil, fmt.Errorf("load projects: %w", err)
	}
	ui.activeProject = projects[0]

	ui.editorPanel = NewEditorPanel(store, w, func() {
		ui.RefreshTreeData()
	})

	ui.buildLayout(projects)
	ui.RefreshTreeData()
	ui.selectFirstAvailableScene()

	return ui, nil
}

func (ui *NovelistUI) buildLayout(projects []Project) {
	// Top Toolbar & Project Switcher
	projectNames := make([]string, len(projects))
	for i, p := range projects {
		projectNames[i] = p.Title
	}

	ui.projectSelect = widget.NewSelect(projectNames, func(selected string) {
		all, _ := ui.store.ListProjects()
		for _, p := range all {
			if p.Title == selected {
				ui.activeProject = p
				ui.RefreshTreeData()
				ui.selectFirstAvailableScene()
				break
			}
		}
	})
	ui.projectSelect.SetSelected(ui.activeProject.Title)

	newProjBtn := widget.NewButtonWithIcon("New Book", theme.FolderNewIcon(), ui.showNewProjectDialog)
	castBtn := widget.NewButtonWithIcon("Cast & Locations", theme.AccountIcon(), ui.showWorldbuildingDialog)
	exportMDBtn := widget.NewButtonWithIcon("Export MD", theme.DocumentSaveIcon(), func() {
		ui.exportManuscript("md")
	})
	exportHTMLBtn := widget.NewButtonWithIcon("Export HTML", theme.DownloadIcon(), func() {
		ui.exportManuscript("html")
	})
	focusBtn := widget.NewButtonWithIcon("Distraction-Free", theme.VisibilityIcon(), func() {
		ui.ToggleDistractionFree()
	})

	topBar := container.NewHBox(
		widget.NewLabelWithStyle("GoNovelist", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		ui.projectSelect,
		newProjBtn,
		castBtn,
		layout.NewSpacer(),
		exportMDBtn,
		exportHTMLBtn,
		focusBtn,
	)

	// Hierarchical widget.Tree for Act -> Chapter -> Scene
	ui.tree = widget.NewTree(
		func(uid widget.TreeNodeUID) []widget.TreeNodeUID {
			return ui.childrenMap[uid]
		},
		func(uid widget.TreeNodeUID) bool {
			if uid == "" {
				return true
			}
			node, ok := ui.nodeLookup[uid]
			if !ok {
				return false
			}
			return node.Kind == NodeAct || node.Kind == NodeChapter
		},
		func(branch bool) fyne.CanvasObject {
			title := widget.NewLabel("Node Title")
			meta := widget.NewLabel("0w")
			meta.TextStyle = fyne.TextStyle{Monospace: true}
			return container.NewBorder(nil, nil, nil, meta, title)
		},
		func(uid widget.TreeNodeUID, branch bool, obj fyne.CanvasObject) {
			c := obj.(*fyne.Container)
			titleLbl := c.Objects[0].(*widget.Label)
			metaLbl := c.Objects[1].(*widget.Label)

			node, ok := ui.nodeLookup[uid]
			if !ok {
				return
			}
			titleLbl.SetText(node.Title)
			switch node.Kind {
			case NodeAct:
				titleLbl.TextStyle = fyne.TextStyle{Bold: true}
				metaLbl.SetText("ACT")
			case NodeChapter:
				titleLbl.TextStyle = fyne.TextStyle{Italic: true}
				metaLbl.SetText(fmt.Sprintf("%dw", node.WordCount))
			case NodeScene:
				titleLbl.TextStyle = fyne.TextStyle{}
				metaLbl.SetText(fmt.Sprintf("[%s] %dw", node.Status, node.WordCount))
			}
		},
	)

	ui.tree.OnSelected = func(uid widget.TreeNodeUID) {
		ui.selectedUID = uid
		node, ok := ui.nodeLookup[uid]
		if !ok {
			return
		}
		if node.Kind == NodeScene {
			ui.editorPanel.LoadScene(ui.activeProject.ID, node.ID)
		}
	}

	// Sidebar Node CRUD + Reorder Action Bar
	addActBtn := widget.NewButton("+ Act", ui.promptAddAct)
	addChapBtn := widget.NewButton("+ Ch", ui.promptAddChapter)
	addSceneBtn := widget.NewButton("+ Scene", ui.promptAddScene)
	renameBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), ui.promptRenameNode)
	upBtn := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() { ui.moveSelectedNode(-1) })
	downBtn := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() { ui.moveSelectedNode(1) })
	delBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), ui.promptDeleteNode)

	treeToolbar := container.NewVBox(
		container.NewGridWithColumns(3, addActBtn, addChapBtn, addSceneBtn),
		container.NewGridWithColumns(4, renameBtn, upBtn, downBtn, delBtn),
		widget.NewSeparator(),
	)

	ui.statusFooter = widget.NewLabel("Manuscript Ready")
	ui.sidebarBox = container.NewBorder(treeToolbar, ui.statusFooter, nil, nil, ui.tree)

	ui.mainSplit = container.NewHSplit(ui.sidebarBox, ui.editorPanel.Container())
	ui.mainSplit.Offset = 0.24

	root := container.NewBorder(
		container.NewVBox(topBar, widget.NewSeparator()),
		nil, nil, nil,
		ui.mainSplit,
	)
	ui.window.SetContent(root)
}

// RefreshTreeData reloads the SQLite hierarchy into Fyne's Tree lookup maps.
func (ui *NovelistUI) RefreshTreeData() {
	acts, err := ui.store.LoadHierarchy(ui.activeProject.ID)
	if err != nil {
		return
	}
	ui.acts = acts
	ui.childrenMap = make(map[string][]string)
	ui.nodeLookup = make(map[string]TreeNode)

	totalManuscriptWords := 0
	for _, a := range acts {
		actUID := MakeUID(NodeAct, a.ID)
		ui.childrenMap[""] = append(ui.childrenMap[""], actUID)
		actWords := 0

		for _, c := range a.Chapters {
			chUID := MakeUID(NodeChapter, c.ID)
			ui.childrenMap[actUID] = append(ui.childrenMap[actUID], chUID)
			chWords := 0

			for _, s := range c.Scenes {
				scUID := MakeUID(NodeScene, s.ID)
				ui.childrenMap[chUID] = append(ui.childrenMap[chUID], scUID)
				ui.nodeLookup[scUID] = TreeNode{
					UID:       scUID,
					Kind:      NodeScene,
					ID:        s.ID,
					ParentID:  c.ID,
					Title:     s.Title,
					Status:    s.Status,
					WordCount: s.WordCount,
					Target:    s.TargetWords,
				}
				chWords += s.WordCount
			}

			ui.nodeLookup[chUID] = TreeNode{
				UID:       chUID,
				Kind:      NodeChapter,
				ID:        c.ID,
				ParentID:  a.ID,
				Title:     c.Title,
				WordCount: chWords,
				Target:    c.TargetWords,
			}
			actWords += chWords
		}

		ui.nodeLookup[actUID] = TreeNode{
			UID:       actUID,
			Kind:      NodeAct,
			ID:        a.ID,
			ParentID:  ui.activeProject.ID,
			Title:     a.Title,
			WordCount: actWords,
		}
		totalManuscriptWords += actWords
	}

	ui.tree.Refresh()
	ui.tree.OpenAllBranches()
	ui.statusFooter.SetText(fmt.Sprintf("Total: %d / %d words", totalManuscriptWords, ui.activeProject.TargetWords))
}

func (ui *NovelistUI) selectFirstAvailableScene() {
	for _, a := range ui.acts {
		for _, c := range a.Chapters {
			if len(c.Scenes) > 0 {
				uid := MakeUID(NodeScene, c.Scenes[0].ID)
				ui.tree.Select(uid)
				return
			}
		}
	}
}

// ToggleDistractionFree collapses or restores the left hierarchy tree and right metadata drawer.
func (ui *NovelistUI) ToggleDistractionFree() {
	ui.distractionFree = !ui.distractionFree
	if ui.distractionFree {
		ui.sidebarBox.Hide()
		ui.mainSplit.Offset = 0.0
	} else {
		ui.sidebarBox.Show()
		ui.mainSplit.Offset = 0.24
	}
	ui.editorPanel.SetDistractionFree(ui.distractionFree)
	ui.mainSplit.Refresh()
}

func (ui *NovelistUI) promptAddAct() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("e.g., Act III: The Meridian Breach")
	dialog.ShowForm("Create New Act", "Create", "Cancel",
		[]*widget.FormItem{widget.NewFormItem("Act Title", entry)},
		func(ok bool) {
			if !ok || entry.Text == "" {
				return
			}
			act, err := ui.store.CreateAct(ui.activeProject.ID, entry.Text)
			if err == nil {
				ui.RefreshTreeData()
				ui.tree.Select(MakeUID(NodeAct, act.ID))
			}
		}, ui.window)
}

func (ui *NovelistUI) promptAddChapter() {
	var targetActID int64
	if node, ok := ui.nodeLookup[ui.selectedUID]; ok {
		switch node.Kind {
		case NodeAct:
			targetActID = node.ID
		case NodeChapter:
			targetActID = node.ParentID
		case NodeScene:
			if chNode, ok := ui.nodeLookup[MakeUID(NodeChapter, node.ParentID)]; ok {
				targetActID = chNode.ParentID
			}
		}
	}
	if targetActID == 0 && len(ui.acts) > 0 {
		targetActID = ui.acts[len(ui.acts)-1].ID
	}
	if targetActID == 0 {
		dialog.ShowInformation("No Act Available", "Please create an Act first.", ui.window)
		return
	}

	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("e.g., Chapter 4: The Sounding Line")
	targetEntry := widget.NewEntry()
	targetEntry.SetText("3000")

	dialog.ShowForm("Create New Chapter", "Create", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Chapter Title", titleEntry),
			widget.NewFormItem("Target Words", targetEntry),
		},
		func(ok bool) {
			if !ok || titleEntry.Text == "" {
				return
			}
			var target int
			_, _ = fmt.Sscanf(targetEntry.Text, "%d", &target)
			ch, err := ui.store.CreateChapter(targetActID, titleEntry.Text, target)
			if err == nil {
				ui.RefreshTreeData()
				ui.tree.Select(MakeUID(NodeChapter, ch.ID))
			}
		}, ui.window)
}

func (ui *NovelistUI) promptAddScene() {
	var targetChapterID int64
	if node, ok := ui.nodeLookup[ui.selectedUID]; ok {
		switch node.Kind {
		case NodeChapter:
			targetChapterID = node.ID
		case NodeScene:
			targetChapterID = node.ParentID
		case NodeAct:
			for _, a := range ui.acts {
				if a.ID == node.ID && len(a.Chapters) > 0 {
					targetChapterID = a.Chapters[len(a.Chapters)-1].ID
				}
			}
		}
	}
	if targetChapterID == 0 {
		dialog.ShowInformation("Select a Chapter", "Please select a Chapter or Scene first.", ui.window)
		return
	}

	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("e.g., Scene 2: Crossing the Causeway")
	targetEntry := widget.NewEntry()
	targetEntry.SetText("1200")

	dialog.ShowForm("Create New Scene", "Create", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Scene Title", titleEntry),
			widget.NewFormItem("Target Words", targetEntry),
		},
		func(ok bool) {
			if !ok || titleEntry.Text == "" {
				return
			}
			var target int
			_, _ = fmt.Sscanf(targetEntry.Text, "%d", &target)
			sc, err := ui.store.CreateScene(targetChapterID, titleEntry.Text, target)
			if err == nil {
				ui.RefreshTreeData()
				uid := MakeUID(NodeScene, sc.ID)
				ui.tree.Select(uid)
				ui.editorPanel.LoadScene(ui.activeProject.ID, sc.ID)
			}
		}, ui.window)
}

func (ui *NovelistUI) promptRenameNode() {
	node, ok := ui.nodeLookup[ui.selectedUID]
	if !ok {
		return
	}
	entry := widget.NewEntry()
	entry.SetText(node.Title)

	dialog.ShowForm("Rename Item", "Save", "Cancel",
		[]*widget.FormItem{widget.NewFormItem("Title", entry)},
		func(confirmed bool) {
			if !confirmed || entry.Text == "" {
				return
			}
			switch node.Kind {
			case NodeAct:
				_ = ui.store.RenameAct(node.ID, entry.Text)
			case NodeChapter:
				_ = ui.store.UpdateChapter(node.ID, entry.Text, node.Target)
			case NodeScene:
				sc, err := ui.store.GetScene(node.ID)
				if err == nil {
					sc.Title = entry.Text
					_ = ui.store.UpdateScene(sc)
					ui.editorPanel.LoadScene(ui.activeProject.ID, sc.ID)
				}
			}
			ui.RefreshTreeData()
		}, ui.window)
}

func (ui *NovelistUI) moveSelectedNode(direction int) {
	node, ok := ui.nodeLookup[ui.selectedUID]
	if !ok {
		return
	}
	switch node.Kind {
	case NodeAct:
		_ = ui.store.MoveAct(node.ID, direction)
	case NodeChapter:
		_ = ui.store.MoveChapter(node.ID, direction)
	case NodeScene:
		_ = ui.store.MoveScene(node.ID, direction)
	}
	ui.RefreshTreeData()
}

func (ui *NovelistUI) promptDeleteNode() {
	node, ok := ui.nodeLookup[ui.selectedUID]
	if !ok {
		return
	}
	dialog.ShowConfirm("Confirm Deletion",
		fmt.Sprintf("Delete '%s' and any nested items permanently?", node.Title),
		func(confirmed bool) {
			if !confirmed {
				return
			}
			switch node.Kind {
			case NodeAct:
				_ = ui.store.DeleteAct(node.ID)
			case NodeChapter:
				_ = ui.store.DeleteChapter(node.ID)
			case NodeScene:
				_ = ui.store.DeleteScene(node.ID)
			}
			ui.selectedUID = ""
			ui.RefreshTreeData()
			ui.selectFirstAvailableScene()
		}, ui.window)
}

func (ui *NovelistUI) showNewProjectDialog() {
	titleEntry := widget.NewEntry()
	authorEntry := widget.NewEntry()
	genreEntry := widget.NewEntry()
	targetEntry := widget.NewEntry()
	targetEntry.SetText("80000")

	dialog.ShowForm("New Novel Project", "Create", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Book Title", titleEntry),
			widget.NewFormItem("Author", authorEntry),
			widget.NewFormItem("Genre", genreEntry),
			widget.NewFormItem("Target Word Count", targetEntry),
		},
		func(ok bool) {
			if !ok || titleEntry.Text == "" {
				return
			}
			var target int
			_, _ = fmt.Sscanf(targetEntry.Text, "%d", &target)
			proj, err := ui.store.CreateProject(titleEntry.Text, authorEntry.Text, genreEntry.Text, target)
			if err != nil {
				return
			}
			act, _ := ui.store.CreateAct(proj.ID, "Act I: Opening")
			ch, _ := ui.store.CreateChapter(act.ID, "Chapter 1", 3000)
			_, _ = ui.store.CreateScene(ch.ID, "Scene 1", 1200)

			projects, _ := ui.store.ListProjects()
			names := make([]string, len(projects))
			for i, p := range projects {
				names[i] = p.Title
			}
			ui.projectSelect.Options = names
			ui.activeProject = *proj
			ui.projectSelect.SetSelected(proj.Title)
			ui.RefreshTreeData()
			ui.selectFirstAvailableScene()
		}, ui.window)
}

func (ui *NovelistUI) showWorldbuildingDialog() {
	charName := widget.NewEntry()
	charName.SetPlaceHolder("Character Name")
	charRole := widget.NewSelect([]string{"Protagonist", "Deuteragonist", "Antagonist", "Supporting"}, nil)
	charRole.SetSelected("Supporting")
	charBio := widget.NewEntry()
	charBio.SetPlaceHolder("Brief character notes")

	locName := widget.NewEntry()
	locName.SetPlaceHolder("Location / Setting Name")
	locDesc := widget.NewEntry()
	locDesc.SetPlaceHolder("Atmospheric details")

	addCharBtn := widget.NewButton("Add Character", func() {
		if charName.Text == "" {
			return
		}
		_, _ = ui.store.CreateCharacter(ui.activeProject.ID, charName.Text, charRole.Selected, charBio.Text)
		charName.SetText("")
		charBio.SetText("")
		ui.editorPanel.ReloadMetadataOptions(ui.activeProject.ID)
	})

	addLocBtn := widget.NewButton("Add Location", func() {
		if locName.Text == "" {
			return
		}
		_, _ = ui.store.CreateLocation(ui.activeProject.ID, locName.Text, locDesc.Text)
		locName.SetText("")
		locDesc.SetText("")
		ui.editorPanel.ReloadMetadataOptions(ui.activeProject.ID)
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("Add Cast Character", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		charName, charRole, charBio, addCharBtn,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Add World Location", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		locName, locDesc, addLocBtn,
	)

	d := dialog.NewCustom("Cast & Worldbuilding Registry", "Done", content, ui.window)
	d.Resize(fyne.NewSize(440, 420))
	d.Show()
}

func (ui *NovelistUI) exportManuscript(format string) {
	var compiled string
	var err error
	ext := ".md"
	if format == "html" {
		compiled, err = ui.store.ExportManuscriptHTML(ui.activeProject)
		ext = ".html"
	} else {
		compiled, err = ui.store.ExportManuscriptMarkdown(ui.activeProject)
	}
	if err != nil {
		dialog.ShowError(err, ui.window)
		return
	}

	outName := filepath.Clean(ui.activeProject.Title) + "_manuscript" + ext
	if err := os.WriteFile(outName, []byte(compiled), 0644); err != nil {
		dialog.ShowError(err, ui.window)
		return
	}
	dialog.ShowInformation("Manuscript Exported",
		fmt.Sprintf("Compiled sequential Acts, Chapters, and Scenes to:\n%s", outName),
		ui.window)
}
