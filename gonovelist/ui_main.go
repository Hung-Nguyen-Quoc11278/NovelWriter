package main

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ResponsiveWindowLayout là bộ quản lý bố cục gốc (fyne.Layout) tự động bắt sự kiện thay đổi
// kích thước cửa sổ, phóng to toàn màn hình (Maximize), khôi phục (Restore) hoặc đổi tỷ lệ màn hình (Scale).
// Khi cửa sổ thay đổi kích thước đột ngột (như nhấn nút Maximize trên Linux X11/Wayland),
// nó buộc toàn bộ các container con và các thanh chia HSplit lồng nhau tính toán lại ranh giới ngay lập tức.
type ResponsiveWindowLayout struct {
	mu          sync.Mutex
	lastSize    fyne.Size
	onResized   func(newSize fyne.Size)
	settleTimer *time.Timer
}

// NewResponsiveWindowLayout khởi tạo bộ lắng nghe thay đổi kích thước cửa sổ thông qua cơ chế Layout của Fyne.
func NewResponsiveWindowLayout(onResized func(newSize fyne.Size)) *ResponsiveWindowLayout {
	return &ResponsiveWindowLayout{
		onResized: onResized,
	}
}

// Layout được Fyne tự động gọi mỗi khi khung vẽ (Canvas) của cửa sổ chính thay đổi kích thước hoặc phóng to.
func (l *ResponsiveWindowLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, obj := range objects {
		if obj == nil || !obj.Visible() {
			continue
		}
		obj.Move(fyne.NewPos(0, 0))
		obj.Resize(size)
	}

	if size.Width <= 0 || size.Height <= 0 {
		return
	}

	l.mu.Lock()
	changed := size.Width != l.lastSize.Width || size.Height != l.lastSize.Height
	if !changed {
		l.mu.Unlock()
		return
	}
	l.lastSize = size

	if l.settleTimer != nil {
		l.settleTimer.Stop()
	}
	// Trên Linux (X11/Wayland), sự kiện Maximize đôi khi gửi kích thước khung hình trước khi
	// viewport OpenGL ổn định hoàn toàn. Hẹn giờ đồng bộ lại sau 45ms giúp thanh bên phải
	// tự động giãn chuẩn xác mà không cần người dùng kéo tay viền cửa sổ.
	settledSize := size
	l.settleTimer = time.AfterFunc(45*time.Millisecond, func() {
		if l.onResized != nil {
			l.onResized(settledSize)
		}
	})
	l.mu.Unlock()

	if l.onResized != nil {
		l.onResized(size)
	}
}

// MinSize trả về kích thước tối thiểu hợp lý để không khóa cứng các thanh chia HSplit khi thu phóng.
func (l *ResponsiveWindowLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minW, minH := float32(320), float32(240)
	for _, obj := range objects {
		if obj == nil || !obj.Visible() {
			continue
		}
		ms := obj.MinSize()
		if ms.Width > minW {
			minW = ms.Width
		}
		if ms.Height > minH {
			minH = ms.Height
		}
	}
	return fyne.NewSize(minW, minH)
}

// NovelistUI điều phối bố cục cửa sổ chính, cây phân cấp Hồi/Chương/Cảnh và Trung tâm Thế giới.
type NovelistUI struct {
	app           fyne.App
	window        fyne.Window
	store         *Store
	activeProject Project

	// Dữ liệu ánh xạ cho Fyne Tree
	childrenMap map[string][]string
	nodeMeta    map[string]HierarchyNode
	selectedUID string

	// Các thành phần giao diện
	projectSelect   *widget.Select
	tree            *widget.Tree
	editorPanel     *EditorPanel
	mainSplit       *container.Split
	rootContainer   *fyne.Container
	sidebarBox      *fyne.Container
	distractionFree bool
	statusFooter    *widget.Label
	fontTheme       *DynamicFontTheme
}

// NewNovelistUI khởi tạo toàn bộ giao diện người dùng tiếng Việt cho GoNovelist.
func NewNovelistUI(app fyne.App, window fyne.Window, store *Store) (*NovelistUI, error) {
	projects, err := store.ListProjects()
	if err != nil || len(projects) == 0 {
		return nil, fmt.Errorf("không thể tải danh sách tác phẩm: %w", err)
	}

	// Nạp cỡ chữ ưa thích đã lưu từ Preferences (mặc định 18px chống mỏi mắt)
	savedFontSize := float32(app.Preferences().FloatWithFallback(PrefKeyEditorFontSize, float64(DefaultEditorFontSize)))
	fontTheme := NewDynamicFontTheme(savedFontSize)
	app.Settings().SetTheme(fontTheme)

	ui := &NovelistUI{
		app:           app,
		window:        window,
		store:         store,
		activeProject: projects[0],
		childrenMap:   make(map[string][]string),
		nodeMeta:      make(map[string]HierarchyNode),
		fontTheme:     fontTheme,
	}

	ui.editorPanel = NewEditorPanel(store, window, ui.activeProject.ID, func() {
		ui.RefreshTreeStatsOnly()
	})
	ui.editorPanel.BindZoomHandlers(
		func(delta float32) { ui.AdjustFontSize(delta) },
		func() { ui.ResetFontSize() },
	)
	ui.editorPanel.UpdateFontSizeIndicator(fontTheme.TextSize(), fontTheme.ZoomPercent())

	ui.buildMainMenu()
	content := ui.buildLayout(projects)
	ui.window.SetContent(content)
	ui.registerZoomShortcuts()

	ui.RefreshTreeData()
	ui.selectFirstAvailableScene()
	return ui, nil
}

// openWorldBuildingHub mở cửa sổ đa tab quản lý Nhân vật, Địa điểm, Vật phẩm, Sự kiện và Thẻ.
func (ui *NovelistUI) openWorldBuildingHub() {
	ShowWorldBuildingHub(ui.store, ui.window, ui.activeProject.ID, func() {
		ui.editorPanel.ReloadMetadataOptions(ui.activeProject.ID)
		if ui.editorPanel.activeScene != nil {
			_ = ui.editorPanel.LoadScene(ui.editorPanel.activeScene.ID, ui.activeProject.ID)
		}
	})
}

func (ui *NovelistUI) buildMainMenu() {
	fileMenu := fyne.NewMenu("Tệp",
		fyne.NewMenuItem("Tác phẩm mới...", func() {
			ui.showNewProjectDialog()
		}),
		fyne.NewMenuItem("Tạo tác phẩm mẫu Tiếng Việt", func() {
			proj, err := ui.store.SeedVietnameseSampleProject()
			if err != nil {
				dialog.ShowError(err, ui.window)
				return
			}
			ui.reloadProjectSelector(*proj)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Xuất bản ra Text (.txt)...", func() {
			ui.ShowExportDialog(ExportFormatTXT)
		}),
		fyne.NewMenuItem("Xuất bản ra ODT (.odt)...", func() {
			ui.ShowExportDialog(ExportFormatODT)
		}),
		fyne.NewMenuItem("Xuất bản ra PDF (.pdf)...", func() {
			ui.ShowExportDialog(ExportFormatPDF)
		}),
		fyne.NewMenuItem("Xuất bản ra EPUB (.epub)...", func() {
			ui.ShowExportDialog(ExportFormatEPUB)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Xuất bản ra Markdown (.md)...", func() {
			ui.ShowExportDialog(ExportFormatMarkdown)
		}),
		fyne.NewMenuItem("Xuất bản ra HTML (.html)...", func() {
			ui.ShowExportDialog(ExportFormatHTML)
		}),
	)

	hierarchyMenu := fyne.NewMenu("Cấu trúc",
		fyne.NewMenuItem("Thêm Hồi mới", func() { ui.showAddActDialog() }),
		fyne.NewMenuItem("Thêm Chương mới", func() { ui.showAddChapterDialog() }),
		fyne.NewMenuItem("Thêm Cảnh mới", func() { ui.showAddSceneDialog() }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Đổi tên mục đang chọn", func() { ui.showRenameNodeDialog() }),
		fyne.NewMenuItem("Di chuyển lên", func() { ui.moveSelectedNode(-1) }),
		fyne.NewMenuItem("Di chuyển xuống", func() { ui.moveSelectedNode(1) }),
		fyne.NewMenuItem("Xóa mục đang chọn", func() { ui.confirmDeleteNode() }),
	)

	worldMenu := fyne.NewMenu("Thế giới & Thẻ",
		fyne.NewMenuItem("Mở Trung Tâm Thế Giới (Nhân vật / Địa điểm / Vật phẩm / Sự kiện / Thẻ)...", func() {
			ui.openWorldBuildingHub()
		}),
	)

	viewMenu := fyne.NewMenu("Chế độ xem",
		fyne.NewMenuItem("Bật/Tắt Chế độ Tập trung (Ẩn thanh bên)", func() {
			ui.ToggleDistractionFree()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Phóng to cỡ chữ (Ctrl + +)", func() {
			ui.AdjustFontSize(EditorFontSizeStep)
		}),
		fyne.NewMenuItem("Thu nhỏ cỡ chữ (Ctrl + -)", func() {
			ui.AdjustFontSize(-EditorFontSizeStep)
		}),
		fyne.NewMenuItem("Đặt lại cỡ chữ mặc định 18px (Ctrl + 0)", func() {
			ui.ResetFontSize()
		}),
	)

	ui.window.SetMainMenu(fyne.NewMainMenu(fileMenu, hierarchyMenu, worldMenu, viewMenu))
}

func (ui *NovelistUI) buildLayout(projects []Project) fyne.CanvasObject {
	projectNames := make([]string, len(projects))
	for i, p := range projects {
		projectNames[i] = p.Title
	}

	ui.projectSelect = widget.NewSelect(projectNames, nil)
	ui.projectSelect.SetSelected(ui.activeProject.Title)
	ui.projectSelect.OnChanged = func(selected string) {
		all, _ := ui.store.ListProjects()
		for _, p := range all {
			if p.Title == selected {
				ui.activeProject = p
				ui.editorPanel.ReloadMetadataOptions(p.ID)
				ui.RefreshTreeData()
				ui.selectFirstAvailableScene()
				break
			}
		}
	}

	// Nút mở Trung tâm Xây dựng Thế giới & Quản lý Thẻ trên thanh công cụ
	worldHubBtn := widget.NewButtonWithIcon("Quản lý Thế giới & Thẻ", theme.GridIcon(), func() {
		ui.openWorldBuildingHub()
	})
	worldHubBtn.Importance = widget.HighImportance

	// Nút mở Hộp thoại Xuất Bản Thảo Đa Định Dạng (TXT, ODT, PDF, EPUB, MD, HTML)
	exportHubBtn := widget.NewButtonWithIcon("Xuất bản thảo...", theme.DocumentSaveIcon(), func() {
		ui.ShowExportDialog(ExportFormatPDF)
	})

	// Công tắc bật/tắt bộ gõ Tiếng Việt Telex nội bộ
	telexCheck := widget.NewCheck("Bộ gõ Tiếng Việt Telex tích hợp", func(checked bool) {
		GlobalTelexEnabled = checked
	})
	telexCheck.SetChecked(GlobalTelexEnabled)

	// Thanh công cụ thao tác nhanh cho Hồi / Chương / Cảnh
	addActBtn := widget.NewButtonWithIcon("Hồi", theme.ContentAddIcon(), func() {
		ui.showAddActDialog()
	})
	addChapBtn := widget.NewButtonWithIcon("Chương", theme.ContentAddIcon(), func() {
		ui.showAddChapterDialog()
	})
	addSceneBtn := widget.NewButtonWithIcon("Cảnh", theme.ContentAddIcon(), func() {
		ui.showAddSceneDialog()
	})

	renameBtn := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
		ui.showRenameNodeDialog()
	})
	renameBtn.Importance = widget.LowImportance

	upBtn := widget.NewButtonWithIcon("", theme.MoveUpIcon(), func() {
		ui.moveSelectedNode(-1)
	})
	upBtn.Importance = widget.LowImportance

	downBtn := widget.NewButtonWithIcon("", theme.MoveDownIcon(), func() {
		ui.moveSelectedNode(1)
	})
	downBtn.Importance = widget.LowImportance

	deleteBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		ui.confirmDeleteNode()
	})
	deleteBtn.Importance = widget.DangerImportance

	// Cây phân cấp widget.Tree cho Hồi -> Chương -> Cảnh
	ui.tree = widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return ui.childrenMap[uid]
		},
		func(uid widget.TreeNodeID) bool {
			if uid == "" {
				return true
			}
			node, ok := ui.nodeMeta[uid]
			if !ok {
				return false
			}
			return node.Kind == "act" || node.Kind == "chapter"
		},
		func(branch bool) fyne.CanvasObject {
			title := widget.NewLabel("Tiêu đề mục")
			title.Truncation = fyne.TextTruncateEllipsis
			badge := widget.NewLabel("0 từ")
			badge.TextStyle = fyne.TextStyle{Monospace: true}
			return container.NewBorder(nil, nil, nil, badge, title)
		},
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			node, ok := ui.nodeMeta[uid]
			if !ok {
				return
			}
			box := obj.(*fyne.Container)
			var titleLabel *widget.Label
			var badgeLabel *widget.Label
			for _, child := range box.Objects {
				if lbl, ok := child.(*widget.Label); ok {
					if lbl.TextStyle.Monospace {
						badgeLabel = lbl
					} else {
						titleLabel = lbl
					}
				}
			}
			if titleLabel != nil {
				switch node.Kind {
				case "act":
					titleLabel.SetText("📚 " + node.Title)
					titleLabel.TextStyle = fyne.TextStyle{Bold: true}
				case "chapter":
					titleLabel.SetText("📖 " + node.Title)
					titleLabel.TextStyle = fyne.TextStyle{Bold: false}
				case "scene":
					titleLabel.SetText("🎬 " + node.Title)
					titleLabel.TextStyle = fyne.TextStyle{Italic: false}
				}
				titleLabel.Refresh()
			}
			if badgeLabel != nil {
				if node.Kind == "scene" {
					badgeLabel.SetText(fmt.Sprintf("[%s] %dw", node.Status, node.WordCount))
				} else {
					badgeLabel.SetText(fmt.Sprintf("%dw", node.WordCount))
				}
			}
		},
	)

	ui.tree.OnSelected = func(uid widget.TreeNodeID) {
		ui.selectedUID = uid
		node, ok := ui.nodeMeta[uid]
		if !ok {
			return
		}
		if node.Kind == "scene" {
			_ = ui.editorPanel.LoadScene(node.DatabaseID, ui.activeProject.ID)
		}
	}

	ui.statusFooter = widget.NewLabel("Tổng số từ: 0")

	topSidebarControls := container.NewVBox(
		widget.NewLabelWithStyle("TÁC PHẨM ĐANG MỞ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		ui.projectSelect,
		worldHubBtn,
		exportHubBtn,
		telexCheck,
		widget.NewSeparator(),
		container.NewGridWithColumns(3, addActBtn, addChapBtn, addSceneBtn),
		container.NewHBox(
			widget.NewLabel("Thao tác:"),
			layout.NewSpacer(),
			renameBtn,
			upBtn,
			downBtn,
			deleteBtn,
		),
		widget.NewSeparator(),
	)

	ui.sidebarBox = container.NewBorder(
		topSidebarControls,
		container.NewVBox(widget.NewSeparator(), ui.statusFooter),
		nil,
		nil,
		ui.tree,
	)

	ui.mainSplit = container.NewHSplit(ui.sidebarBox, ui.editorPanel.Container())
	ui.mainSplit.Offset = 0.24

	// Bọc mainSplit trong container gốc sử dụng ResponsiveWindowLayout để tự động làm mới
	// toàn bộ HSplit lồng nhau và thanh bên Ngữ cảnh Cảnh khi cửa sổ Maximize / Restore / Scale.
	windowLayout := NewResponsiveWindowLayout(func(_ fyne.Size) {
		ui.ForceLayoutRefresh()
	})
	ui.rootContainer = container.New(windowLayout, ui.mainSplit)
	return ui.rootContainer
}

// ForceLayoutRefresh buộc toàn bộ cây giao diện (thanh bên trái, HSplit chính, HSplit soạn thảo
// và thanh bên Ngữ cảnh Cảnh bên phải) tính toán lại kích thước và vẽ lại ngay lập tức.
func (ui *NovelistUI) ForceLayoutRefresh() {
	if ui.sidebarBox != nil {
		ui.sidebarBox.Refresh()
	}
	if ui.editorPanel != nil {
		ui.editorPanel.ForceLayoutRefresh()
	}
	if ui.mainSplit != nil {
		ui.mainSplit.Refresh()
	}
	if ui.rootContainer != nil {
		ui.rootContainer.Refresh()
	}
}

func (ui *NovelistUI) reloadProjectSelector(active Project) {
	all, _ := ui.store.ListProjects()
	names := make([]string, len(all))
	for i, p := range all {
		names[i] = p.Title
	}
	ui.activeProject = active
	ui.projectSelect.Options = names
	ui.projectSelect.SetSelected(active.Title)
	ui.projectSelect.Refresh()
	ui.editorPanel.ReloadMetadataOptions(active.ID)
	ui.RefreshTreeData()
	ui.selectFirstAvailableScene()
}

// ToggleDistractionFree bật hoặc tắt chế độ viết tập trung (ẩn cây thư mục và bảng siêu dữ liệu).
func (ui *NovelistUI) ToggleDistractionFree() {
	ui.distractionFree = !ui.distractionFree
	if ui.distractionFree {
		ui.sidebarBox.Hide()
		ui.editorPanel.SetDistractionFree(true)
		ui.mainSplit.Offset = 0.0
	} else {
		ui.sidebarBox.Show()
		ui.editorPanel.SetDistractionFree(false)
		ui.mainSplit.Offset = 0.24
	}
	ui.mainSplit.Refresh()
}

// AdjustFontSize tăng hoặc giảm cỡ chữ động, lưu vào Preferences và làm mới bố cục ngay lập tức
// mà không làm mất vị trí con trỏ đang soạn thảo.
func (ui *NovelistUI) AdjustFontSize(delta float32) {
	if ui.fontTheme == nil {
		return
	}
	ui.ApplyFontSize(ui.fontTheme.TextSize() + delta)
}

// ResetFontSize đưa cỡ chữ về mức chuẩn dễ đọc (18px).
func (ui *NovelistUI) ResetFontSize() {
	ui.ApplyFontSize(DefaultEditorFontSize)
}

// ApplyFontSize áp dụng cỡ chữ mới cho toàn bộ trình soạn thảo và lưu trữ cục bộ.
func (ui *NovelistUI) ApplyFontSize(targetSize float32) {
	if ui.fontTheme == nil {
		return
	}

	savedRow, savedCol := 0, 0
	if ui.editorPanel != nil && ui.editorPanel.proseEntry != nil {
		savedRow, savedCol = ui.editorPanel.proseEntry.GetLockedCursor()
	}

	newSize := ui.fontTheme.SetTextSize(targetSize)
	ui.app.Preferences().SetFloat(PrefKeyEditorFontSize, float64(newSize))
	ui.app.Settings().SetTheme(ui.fontTheme)

	if ui.editorPanel != nil {
		ui.editorPanel.UpdateFontSizeIndicator(newSize, ui.fontTheme.ZoomPercent())
		if ui.editorPanel.proseEntry != nil {
			ui.editorPanel.proseEntry.Refresh()
			ui.editorPanel.proseEntry.RestoreLockedCursor(savedRow, savedCol)
		}
	}
	ui.ForceLayoutRefresh()
	if ui.window != nil && ui.window.Content() != nil {
		ui.window.Content().Refresh()
	}
}

// registerZoomShortcuts đăng ký các phím tắt Ctrl+=, Ctrl++, Ctrl+-, Ctrl+0 trên cửa sổ chính.
func (ui *NovelistUI) registerZoomShortcuts() {
	if ui.window == nil || ui.window.Canvas() == nil {
		return
	}
	canvas := ui.window.Canvas()

	// Ctrl + = và Ctrl + + để phóng to chữ
	canvas.AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyEqual,
		Modifier: fyne.KeyModifierControl,
	}, func(_ fyne.Shortcut) {
		ui.AdjustFontSize(EditorFontSizeStep)
	})
	canvas.AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyPlus,
		Modifier: fyne.KeyModifierControl,
	}, func(_ fyne.Shortcut) {
		ui.AdjustFontSize(EditorFontSizeStep)
	})

	// Ctrl + - để thu nhỏ chữ
	canvas.AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyMinus,
		Modifier: fyne.KeyModifierControl,
	}, func(_ fyne.Shortcut) {
		ui.AdjustFontSize(-EditorFontSizeStep)
	})

	// Ctrl + 0 để đặt lại cỡ chữ mặc định (18px)
	canvas.AddShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.Key0,
		Modifier: fyne.KeyModifierControl,
	}, func(_ fyne.Shortcut) {
		ui.ResetFontSize()
	})
}

// RefreshTreeData tải lại toàn bộ cấu trúc Hồi -> Chương -> Cảnh từ SQLite và mở toàn bộ nhánh.
func (ui *NovelistUI) RefreshTreeData() {
	ui.refreshTreeInternal(true)
}

// RefreshTreeStatsOnly cập nhật số từ và tiêu đề trên cây thư mục trong lúc tự động lưu
// mà KHÔNG gọi OpenAllBranches() để tránh làm gián đoạn tiêu điểm hoặc vị trí con trỏ của khung soạn thảo.
func (ui *NovelistUI) RefreshTreeStatsOnly() {
	ui.refreshTreeInternal(false)
}

func (ui *NovelistUI) refreshTreeInternal(openBranches bool) {
	ui.childrenMap = make(map[string][]string)
	ui.nodeMeta = make(map[string]HierarchyNode)

	acts, err := ui.store.ListActs(ui.activeProject.ID)
	if err != nil {
		return
	}

	totalManuscriptWords := 0

	for _, act := range acts {
		actUID := MakeNodeUID("act", act.ID)
		ui.childrenMap[""] = append(ui.childrenMap[""], actUID)

		chapters, _ := ui.store.ListChapters(act.ID)
		actWords := 0

		for _, ch := range chapters {
			chUID := MakeNodeUID("chapter", ch.ID)
			ui.childrenMap[actUID] = append(ui.childrenMap[actUID], chUID)

			scenes, _ := ui.store.ListScenes(ch.ID)
			chWords := 0
			for _, sc := range scenes {
				scUID := MakeNodeUID("scene", sc.ID)
				ui.childrenMap[chUID] = append(ui.childrenMap[chUID], scUID)
				ui.nodeMeta[scUID] = HierarchyNode{
					UID:         scUID,
					Kind:        "scene",
					DatabaseID:  sc.ID,
					ParentID:    ch.ID,
					Title:       sc.Title,
					WordCount:   sc.WordCount,
					TargetWords: sc.TargetWords,
					Status:      sc.Status,
				}
				chWords += sc.WordCount
			}

			ui.nodeMeta[chUID] = HierarchyNode{
				UID:         chUID,
				Kind:        "chapter",
				DatabaseID:  ch.ID,
				ParentID:    act.ID,
				Title:       ch.Title,
				WordCount:   chWords,
				TargetWords: ch.TargetWords,
			}
			actWords += chWords
		}

		ui.nodeMeta[actUID] = HierarchyNode{
			UID:        actUID,
			Kind:       "act",
			DatabaseID: act.ID,
			ParentID:   ui.activeProject.ID,
			Title:      act.Title,
			WordCount:  actWords,
		}
		totalManuscriptWords += actWords
	}

	if ui.tree != nil {
		ui.tree.Refresh()
		if openBranches {
			ui.tree.OpenAllBranches()
		}
	}
	if ui.statusFooter != nil {
		ui.statusFooter.SetText(fmt.Sprintf("Tổng cộng: %d / %d từ", totalManuscriptWords, ui.activeProject.TargetWords))
	}
}

func (ui *NovelistUI) selectFirstAvailableScene() {
	for _, actUID := range ui.childrenMap[""] {
		for _, chUID := range ui.childrenMap[actUID] {
			scenes := ui.childrenMap[chUID]
			if len(scenes) > 0 {
				ui.tree.Select(scenes[0])
				return
			}
		}
	}
}

func (ui *NovelistUI) resolveTargetActID() (int64, error) {
	if node, ok := ui.nodeMeta[ui.selectedUID]; ok {
		switch node.Kind {
		case "act":
			return node.DatabaseID, nil
		case "chapter":
			return node.ParentID, nil
		case "scene":
			chUID := MakeNodeUID("chapter", node.ParentID)
			if chNode, exists := ui.nodeMeta[chUID]; exists {
				return chNode.ParentID, nil
			}
		}
	}
	acts, err := ui.store.ListActs(ui.activeProject.ID)
	if err != nil || len(acts) == 0 {
		return 0, fmt.Errorf("vui lòng tạo ít nhất một Hồi trước")
	}
	return acts[len(acts)-1].ID, nil
}

func (ui *NovelistUI) resolveTargetChapterID() (int64, error) {
	if node, ok := ui.nodeMeta[ui.selectedUID]; ok {
		switch node.Kind {
		case "chapter":
			return node.DatabaseID, nil
		case "scene":
			return node.ParentID, nil
		case "act":
			chUIDs := ui.childrenMap[node.UID]
			if len(chUIDs) > 0 {
				if chNode, exists := ui.nodeMeta[chUIDs[len(chUIDs)-1]]; exists {
					return chNode.DatabaseID, nil
				}
			}
		}
	}
	acts, _ := ui.store.ListActs(ui.activeProject.ID)
	for _, a := range acts {
		chaps, _ := ui.store.ListChapters(a.ID)
		if len(chaps) > 0 {
			return chaps[len(chaps)-1].ID, nil
		}
	}
	return 0, fmt.Errorf("vui lòng tạo ít nhất một Chương trước")
}

// Các hộp thoại thêm / sửa / xóa Hồi, Chương, Cảnh hỗ trợ nhập Tiếng Việt Telex

func (ui *NovelistUI) showNewProjectDialog() {
	titleEntry := NewVietEntry()
	titleEntry.SetPlaceHolder("VD: Mùa Gió Chướng Trên Đỉnh Ngự Bình")
	authorEntry := NewVietEntry()
	authorEntry.SetPlaceHolder("Tên tác giả")
	genreEntry := NewVietEntry()
	genreEntry.SetPlaceHolder("Tiểu thuyết lịch sử / Văn học đương đại...")

	dialog.ShowForm("Khởi tạo Tác phẩm Mới", "Tạo tác phẩm", "Hủy", []*widget.FormItem{
		widget.NewFormItem("Tên tác phẩm", titleEntry),
		widget.NewFormItem("Tác giả", authorEntry),
		widget.NewFormItem("Thể loại", genreEntry),
	}, func(ok bool) {
		if !ok || titleEntry.Text == "" {
			return
		}
		proj, err := ui.store.CreateProject(titleEntry.Text, authorEntry.Text, genreEntry.Text, "", 50000)
		if err != nil {
			dialog.ShowError(err, ui.window)
			return
		}
		act, _ := ui.store.CreateAct(proj.ID, "Hồi I — Khởi Đầu")
		ch, _ := ui.store.CreateChapter(act.ID, "Chương 1", 3000)
		_, _ = ui.store.CreateScene(ch.ID, "Cảnh 1: Mở đầu", 1200)

		ui.reloadProjectSelector(*proj)
	}, ui.window)
}

func (ui *NovelistUI) showAddActDialog() {
	entry := NewVietEntry()
	entry.SetPlaceHolder("VD: Hồi III — Ngày Trở Về")
	dialog.ShowForm("Thêm Hồi Mới", "Tạo Hồi", "Hủy", []*widget.FormItem{
		widget.NewFormItem("Tiêu đề Hồi", entry),
	}, func(ok bool) {
		if !ok || entry.Text == "" {
			return
		}
		if _, err := ui.store.CreateAct(ui.activeProject.ID, entry.Text); err != nil {
			dialog.ShowError(err, ui.window)
			return
		}
		ui.RefreshTreeData()
	}, ui.window)
}

func (ui *NovelistUI) showAddChapterDialog() {
	actID, err := ui.resolveTargetActID()
	if err != nil {
		dialog.ShowError(err, ui.window)
		return
	}
	titleEntry := NewVietEntry()
	titleEntry.SetPlaceHolder("VD: Chương 4: Bến Đò Đêm Mưa")
	targetEntry := widget.NewEntry()
	targetEntry.SetText("3000")

	dialog.ShowForm("Thêm Chương Mới", "Tạo Chương", "Hủy", []*widget.FormItem{
		widget.NewFormItem("Tiêu đề Chương", titleEntry),
		widget.NewFormItem("Mục tiêu số từ", targetEntry),
	}, func(ok bool) {
		if !ok || titleEntry.Text == "" {
			return
		}
		var target int
		_, _ = fmt.Sscanf(targetEntry.Text, "%d", &target)
		if _, err := ui.store.CreateChapter(actID, titleEntry.Text, target); err != nil {
			dialog.ShowError(err, ui.window)
			return
		}
		ui.RefreshTreeData()
	}, ui.window)
}

func (ui *NovelistUI) showAddSceneDialog() {
	chID, err := ui.resolveTargetChapterID()
	if err != nil {
		dialog.ShowError(err, ui.window)
		return
	}
	titleEntry := NewVietEntry()
	titleEntry.SetPlaceHolder("VD: Cảnh 2: Cuộc Gặp Dưới Hiên Trà")
	targetEntry := widget.NewEntry()
	targetEntry.SetText("1200")

	dialog.ShowForm("Thêm Cảnh Mới", "Tạo Cảnh", "Hủy", []*widget.FormItem{
		widget.NewFormItem("Tiêu đề Cảnh", titleEntry),
		widget.NewFormItem("Mục tiêu số từ", targetEntry),
	}, func(ok bool) {
		if !ok || titleEntry.Text == "" {
			return
		}
		var target int
		_, _ = fmt.Sscanf(targetEntry.Text, "%d", &target)
		sc, err := ui.store.CreateScene(chID, titleEntry.Text, target)
		if err != nil {
			dialog.ShowError(err, ui.window)
			return
		}
		ui.RefreshTreeData()
		ui.tree.Select(MakeNodeUID("scene", sc.ID))
	}, ui.window)
}

func (ui *NovelistUI) showRenameNodeDialog() {
	node, ok := ui.nodeMeta[ui.selectedUID]
	if !ok {
		return
	}
	entry := NewVietEntry()
	entry.SetText(node.Title)

	dialog.ShowForm("Đổi Tên Mục", "Lưu thay đổi", "Hủy", []*widget.FormItem{
		widget.NewFormItem("Tiêu đề mới", entry),
	}, func(confirmed bool) {
		if !confirmed || entry.Text == "" {
			return
		}
		var err error
		switch node.Kind {
		case "act":
			err = ui.store.RenameAct(node.DatabaseID, entry.Text)
		case "chapter":
			err = ui.store.RenameChapter(node.DatabaseID, entry.Text)
		case "scene":
			err = ui.store.RenameScene(node.DatabaseID, entry.Text)
			_ = ui.editorPanel.LoadScene(node.DatabaseID, ui.activeProject.ID)
		}
		if err != nil {
			dialog.ShowError(err, ui.window)
			return
		}
		ui.RefreshTreeData()
	}, ui.window)
}

func (ui *NovelistUI) moveSelectedNode(direction int) {
	node, ok := ui.nodeMeta[ui.selectedUID]
	if !ok {
		return
	}
	var err error
	switch node.Kind {
	case "act":
		err = ui.store.MoveAct(node.DatabaseID, direction)
	case "chapter":
		err = ui.store.MoveChapter(node.DatabaseID, direction)
	case "scene":
		err = ui.store.MoveScene(node.DatabaseID, direction)
	}
	if err != nil {
		dialog.ShowError(err, ui.window)
		return
	}
	ui.RefreshTreeData()
}

func (ui *NovelistUI) confirmDeleteNode() {
	node, ok := ui.nodeMeta[ui.selectedUID]
	if !ok {
		return
	}
	dialog.ShowConfirm(
		"Xác nhận xóa",
		fmt.Sprintf("Bạn có chắc chắn muốn xóa %q và toàn bộ các mục con bên trong?", node.Title),
		func(confirmed bool) {
			if !confirmed {
				return
			}
			var err error
			switch node.Kind {
			case "act":
				err = ui.store.DeleteAct(node.DatabaseID)
			case "chapter":
				err = ui.store.DeleteChapter(node.DatabaseID)
			case "scene":
				err = ui.store.DeleteScene(node.DatabaseID)
			}
			if err != nil {
				dialog.ShowError(err, ui.window)
				return
			}
			ui.selectedUID = ""
			ui.RefreshTreeData()
			ui.selectFirstAvailableScene()
		},
		ui.window,
	)
}

func (ui *NovelistUI) exportManuscript(format string) {
	ui.ShowExportDialog(ExportFormat(format))
}
