package main

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// parseHexColor chuyển đổi chuỗi mã màu Hex (VD: "#3498db") sang đối tượng color.NRGBA của Fyne.
func parseHexColor(hexStr string) color.NRGBA {
	clean := strings.TrimPrefix(NormalizeHexColor(hexStr), "#")
	if len(clean) == 3 {
		clean = string([]byte{clean[0], clean[0], clean[1], clean[1], clean[2], clean[2]})
	}
	var r, g, b uint8 = 0x34, 0x98, 0xdb
	if len(clean) == 6 {
		_, _ = fmt.Sscanf(clean, "%02x%02x%02x", &r, &g, &b)
	}
	return color.NRGBA{R: r, G: g, B: b, A: 0xff}
}

// parseHexTintColor tạo màu nền bán trong suốt (alpha thấp) từ mã màu Hex để tô nền huy hiệu Thẻ.
func parseHexTintColor(hexStr string, alpha uint8) color.NRGBA {
	c := parseHexColor(hexStr)
	c.A = alpha
	return c
}

// newColorCircleIndicator khởi tạo một hình tròn màu (canvas.Circle) có kích thước cố định để hiển thị màu sắc thẻ.
func newColorCircleIndicator(hexStr string, diameter float32) (*canvas.Circle, *fyne.Container) {
	c := parseHexColor(hexStr)
	circle := canvas.NewCircle(c)
	circle.StrokeColor = color.NRGBA{R: 0, G: 0, B: 0, A: 45}
	circle.StrokeWidth = 1
	wrapper := container.NewGridWrap(fyne.NewSize(diameter, diameter), circle)
	return circle, wrapper
}

// firstTagColorOrDefault lấy mã màu Hex của thẻ đầu tiên trong danh sách (hoặc màu xám nhẹ nếu chưa gắn thẻ).
func firstTagColorOrDefault(tags []Tag) string {
	if len(tags) > 0 && strings.TrimSpace(tags[0].Color) != "" {
		return NormalizeHexColor(tags[0].Color)
	}
	return "#94a3b8"
}

const allTagsFilterLabel = "Tất cả thẻ"

// WorldBuildingHub quản lý cửa sổ đa tab cho Nhân vật, Địa điểm, Vật phẩm, Sự kiện và Hệ thống Thẻ.
type WorldBuildingHub struct {
	store     *Store
	parentWin fyne.Window
	bookID    int64
	onUpdated func()

	tags []Tag

	// Dữ liệu đã lọc cho từng Tab
	filteredChars  []Character
	filteredLocs   []Location
	filteredProps  []Prop
	filteredEvents []Event

	// Trạng thái bộ lọc theo thẻ của từng Tab
	charTagFilter  string
	locTagFilter   string
	propTagFilter  string
	eventTagFilter string

	// Các widget bộ lọc theo thẻ
	charFilterSelect  *widget.Select
	locFilterSelect   *widget.Select
	propFilterSelect  *widget.Select
	eventFilterSelect *widget.Select

	// Danh sách hiển thị (widget.List)
	charList  *widget.List
	locList   *widget.List
	propList  *widget.List
	eventList *widget.List
	tagList   *widget.List

	// Thực thể đang chọn để chỉnh sửa
	selectedCharID  int64
	selectedLocID   int64
	selectedPropID  int64
	selectedEventID int64
	selectedTagID   int64

	// Biểu mẫu Tab 1: Nhân vật
	charNameEntry *VietnameseEntry
	charRoleEntry *VietnameseEntry
	charDescEntry *VietnameseEntry
	charTagCheck  *widget.CheckGroup

	// Biểu mẫu Tab 2: Địa điểm
	locNameEntry *VietnameseEntry
	locDescEntry *VietnameseEntry
	locTagCheck  *widget.CheckGroup

	// Biểu mẫu Tab 3: Vật phẩm
	propNameEntry *VietnameseEntry
	propCatEntry  *VietnameseEntry
	propDescEntry *VietnameseEntry
	propSigEntry  *VietnameseEntry
	propTagCheck  *widget.CheckGroup

	// Biểu mẫu Tab 4: Sự kiện
	eventTitleEntry *VietnameseEntry
	eventOrderEntry *widget.Entry
	eventDescEntry  *VietnameseEntry
	eventTagCheck   *widget.CheckGroup

	// Biểu mẫu Quản lý Thẻ toàn cục & Chọn màu sắc thẻ
	newTagEntry         *VietnameseEntry
	quickColorSelect    *widget.Select
	quickColorCircle    *canvas.Circle
	tagFormNameEntry    *VietnameseEntry
	tagFormHexEntry     *VietnameseEntry
	tagFormPresetSelect *widget.Select
	tagFormPreviewDot   *canvas.Circle
	tagFormPreviewBg    *canvas.Rectangle
	tagFormPreviewLabel *widget.Label
}

// ShowWorldBuildingHub mở Trung tâm Quản lý Thế giới & Hệ thống Thẻ Đa năng (Multi-Tab Dialog).
func ShowWorldBuildingHub(store *Store, parentWin fyne.Window, bookID int64, onUpdated func()) {
	hub := &WorldBuildingHub{
		store:          store,
		parentWin:      parentWin,
		bookID:         bookID,
		onUpdated:      onUpdated,
		charTagFilter:  allTagsFilterLabel,
		locTagFilter:   allTagsFilterLabel,
		propTagFilter:  allTagsFilterLabel,
		eventTagFilter: allTagsFilterLabel,
	}

	content := hub.buildContent()
	hub.reloadAllData()

	d := dialog.NewCustom("Trung Tâm Xây Dựng Thế Giới & Quản Lý Thẻ", "Đóng cửa sổ", content, parentWin)
	d.Resize(fyne.NewSize(1120, 720))
	d.SetOnClosed(func() {
		if hub.onUpdated != nil {
			hub.onUpdated()
		}
	})
	d.Show()
}

func (h *WorldBuildingHub) buildContent() fyne.CanvasObject {
	// Thanh tạo Thẻ nhanh ở trên cùng kèm chọn Màu sắc thẻ (Global Tag Bar)
	h.newTagEntry = NewVietEntry()
	h.newTagEntry.SetPlaceHolder("Nhập tên thẻ mới (VD: Thiên giới, Khu vực cấm, Cổ vật)...")

	presets := DefaultTagColorPresets()
	presetLabels := make([]string, len(presets))
	labelToHex := make(map[string]string, len(presets))
	for i, p := range presets {
		presetLabels[i] = p.Label
		labelToHex[p.Label] = p.Hex
	}

	selectedQuickHex := DefaultTagColor
	var quickDotBox *fyne.Container
	h.quickColorCircle, quickDotBox = newColorCircleIndicator(selectedQuickHex, 18)

	h.quickColorSelect = widget.NewSelect(presetLabels, func(selected string) {
		if hex, ok := labelToHex[selected]; ok {
			selectedQuickHex = hex
			h.quickColorCircle.FillColor = parseHexColor(hex)
			h.quickColorCircle.Refresh()
		}
	})
	if len(presetLabels) > 0 {
		h.quickColorSelect.SetSelected(presetLabels[0])
	}

	quickAddTagBtn := widget.NewButtonWithIcon("Thêm thẻ mới", theme.ContentAddIcon(), func() {
		name := strings.TrimSpace(h.newTagEntry.Text)
		if name == "" {
			return
		}
		if _, err := h.store.CreateTag(h.bookID, name, selectedQuickHex); err != nil {
			dialog.ShowError(err, h.parentWin)
			return
		}
		h.newTagEntry.SetText("")
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	})
	quickAddTagBtn.Importance = widget.HighImportance

	rightQuickControls := container.NewHBox(
		widget.NewLabel("Màu sắc thẻ:"),
		container.NewCenter(quickDotBox),
		h.quickColorSelect,
		quickAddTagBtn,
	)

	globalTagHeader := container.NewBorder(
		nil,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("🏷️ Thêm thẻ nhanh:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		rightQuickControls,
		h.newTagEntry,
	)

	tabs := container.NewAppTabs(
		container.NewTabItem("Nhân vật", h.buildCharactersTab()),
		container.NewTabItem("Địa điểm", h.buildLocationsTab()),
		container.NewTabItem("Vật phẩm", h.buildPropsTab()),
		container.NewTabItem("Sự kiện", h.buildEventsTab()),
		container.NewTabItem("Quản lý Thẻ", h.buildTagManagerTab()),
	)

	return container.NewBorder(globalTagHeader, nil, nil, nil, tabs)
}

// ==================== TAB 1: NHÂN VẬT (CHARACTERS) ====================

func (h *WorldBuildingHub) buildCharactersTab() fyne.CanvasObject {
	h.charFilterSelect = widget.NewSelect([]string{allTagsFilterLabel}, func(selected string) {
		if selected == "" {
			selected = allTagsFilterLabel
		}
		h.charTagFilter = selected
		h.refreshCharactersList()
	})

	h.charList = widget.NewList(
		func() int { return len(h.filteredChars) },
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Tên nhân vật", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			_, dotWrap := newColorCircleIndicator(DefaultTagColor, 12)
			sub := widget.NewLabel("Vai trò • #Thẻ")
			sub.Truncation = fyne.TextTruncateEllipsis
			subRow := container.NewBorder(nil, nil, container.NewCenter(dotWrap), nil, sub)
			return container.NewVBox(title, subRow)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(h.filteredChars) {
				return
			}
			c := h.filteredChars[i]
			box := obj.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText("👤 " + c.Name)
			subRow := box.Objects[1].(*fyne.Container)
			for _, child := range subRow.Objects {
				switch w := child.(type) {
				case *widget.Label:
					w.SetText(fmt.Sprintf("%s  |  %s", c.Role, FormatTagNames(c.Tags)))
				case *fyne.Container:
					if len(w.Objects) > 0 {
						if wrap, ok := w.Objects[0].(*fyne.Container); ok && len(wrap.Objects) > 0 {
							if circle, ok := wrap.Objects[0].(*canvas.Circle); ok {
								circle.FillColor = parseHexColor(firstTagColorOrDefault(c.Tags))
								circle.Refresh()
							}
						}
					}
				}
			}
		},
	)

	h.charNameEntry = NewVietEntry()
	h.charNameEntry.SetPlaceHolder("Họ và tên nhân vật (VD: Lê Ngọc Liên)...")

	h.charRoleEntry = NewVietEntry()
	h.charRoleEntry.SetPlaceHolder("Vai trò (VD: Nhân vật chính, Phản diện, Đồng hành)...")

	h.charDescEntry = NewVietMultiLineEntry()
	h.charDescEntry.SetPlaceHolder("Tiểu sử, ngoại hình, tính cách, động cơ của nhân vật...")
	h.charDescEntry.SetMinRowsVisible(4)

	h.charTagCheck = widget.NewCheckGroup([]string{}, nil)

	h.charList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(h.filteredChars) {
			return
		}
		c := h.filteredChars[id]
		h.selectedCharID = c.ID
		h.charNameEntry.SetText(c.Name)
		h.charRoleEntry.SetText(c.Role)
		h.charDescEntry.SetText(c.Description)
		h.charTagCheck.SetSelected(extractTagNames(c.Tags))
	}

	newBtn := widget.NewButtonWithIcon("Làm mới biểu mẫu", theme.DocumentCreateIcon(), func() {
		h.clearCharacterForm()
	})

	saveBtn := widget.NewButtonWithIcon("Lưu Nhân Vật", theme.DocumentSaveIcon(), func() {
		name := strings.TrimSpace(h.charNameEntry.Text)
		if name == "" {
			return
		}
		role := strings.TrimSpace(h.charRoleEntry.Text)
		if role == "" {
			role = "Nhân vật chính"
		}
		tagIDs := h.resolveSelectedTagIDs(h.charTagCheck.Selected)

		if h.selectedCharID == 0 {
			created, err := h.store.CreateCharacter(h.bookID, name, role, h.charDescEntry.Text)
			if err != nil {
				dialog.ShowError(err, h.parentWin)
				return
			}
			_ = h.store.SetEntityTags(EntityCharacter, created.ID, tagIDs)
			h.selectedCharID = created.ID
		} else {
			err := h.store.UpdateCharacter(Character{
				ID:          h.selectedCharID,
				ProjectID:   h.bookID,
				Name:        name,
				Role:        role,
				Description: h.charDescEntry.Text,
			}, tagIDs)
			if err != nil {
				dialog.ShowError(err, h.parentWin)
				return
			}
		}
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	})
	saveBtn.Importance = widget.HighImportance

	delBtn := widget.NewButtonWithIcon("Xóa Nhân Vật", theme.DeleteIcon(), func() {
		if h.selectedCharID == 0 {
			return
		}
		_ = h.store.DeleteCharacter(h.selectedCharID)
		h.clearCharacterForm()
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	})
	delBtn.Importance = widget.DangerImportance

	leftPane := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Lọc theo thẻ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			h.charFilterSelect,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		h.charList,
	)

	rightForm := container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("THÔNG TIN NHÂN VẬT", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Tên nhân vật", h.charNameEntry),
			widget.NewFormItem("Vai trò", h.charRoleEntry),
			widget.NewFormItem("Mô tả & Tiểu sử", h.charDescEntry),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("GẮN THẺ PHÂN LOẠI (TAGS)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		h.charTagCheck,
		widget.NewSeparator(),
		container.NewHBox(newBtn, layout.NewSpacer(), delBtn, saveBtn),
	))

	split := container.NewHSplit(leftPane, rightForm)
	split.Offset = 0.38
	return split
}

func (h *WorldBuildingHub) clearCharacterForm() {
	h.selectedCharID = 0
	h.charList.UnselectAll()
	h.charNameEntry.SetText("")
	h.charRoleEntry.SetText("")
	h.charDescEntry.SetText("")
	h.charTagCheck.SetSelected([]string{})
}

// ==================== TAB 2: ĐỊA ĐIỂM (LOCATIONS) ====================

func (h *WorldBuildingHub) buildLocationsTab() fyne.CanvasObject {
	h.locFilterSelect = widget.NewSelect([]string{allTagsFilterLabel}, func(selected string) {
		if selected == "" {
			selected = allTagsFilterLabel
		}
		h.locTagFilter = selected
		h.refreshLocationsList()
	})

	h.locList = widget.NewList(
		func() int { return len(h.filteredLocs) },
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Tên địa điểm", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			_, dotWrap := newColorCircleIndicator(DefaultTagColor, 12)
			sub := widget.NewLabel("#Thẻ")
			sub.Truncation = fyne.TextTruncateEllipsis
			subRow := container.NewBorder(nil, nil, container.NewCenter(dotWrap), nil, sub)
			return container.NewVBox(title, subRow)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(h.filteredLocs) {
				return
			}
			l := h.filteredLocs[i]
			box := obj.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText("🏛️ " + l.Name)
			subRow := box.Objects[1].(*fyne.Container)
			for _, child := range subRow.Objects {
				switch w := child.(type) {
				case *widget.Label:
					w.SetText(FormatTagNames(l.Tags))
				case *fyne.Container:
					if len(w.Objects) > 0 {
						if wrap, ok := w.Objects[0].(*fyne.Container); ok && len(wrap.Objects) > 0 {
							if circle, ok := wrap.Objects[0].(*canvas.Circle); ok {
								circle.FillColor = parseHexColor(firstTagColorOrDefault(l.Tags))
								circle.Refresh()
							}
						}
					}
				}
			}
		},
	)

	h.locNameEntry = NewVietEntry()
	h.locNameEntry.SetPlaceHolder("Tên địa điểm / bối cảnh (VD: Xưởng Thủy Tinh Phố Cổ)...")

	h.locDescEntry = NewVietMultiLineEntry()
	h.locDescEntry.SetPlaceHolder("Kiến trúc, không khí, lịch sử và các chi tiết giác quan của địa điểm...")
	h.locDescEntry.SetMinRowsVisible(5)

	h.locTagCheck = widget.NewCheckGroup([]string{}, nil)

	h.locList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(h.filteredLocs) {
			return
		}
		l := h.filteredLocs[id]
		h.selectedLocID = l.ID
		h.locNameEntry.SetText(l.Name)
		h.locDescEntry.SetText(l.Description)
		h.locTagCheck.SetSelected(extractTagNames(l.Tags))
	}

	newBtn := widget.NewButtonWithIcon("Làm mới biểu mẫu", theme.DocumentCreateIcon(), func() {
		h.clearLocationForm()
	})

	saveBtn := widget.NewButtonWithIcon("Lưu Địa Điểm", theme.DocumentSaveIcon(), func() {
		name := strings.TrimSpace(h.locNameEntry.Text)
		if name == "" {
			return
		}
		tagIDs := h.resolveSelectedTagIDs(h.locTagCheck.Selected)

		if h.selectedLocID == 0 {
			created, err := h.store.CreateLocation(h.bookID, name, h.locDescEntry.Text)
			if err != nil {
				dialog.ShowError(err, h.parentWin)
				return
			}
			_ = h.store.SetEntityTags(EntityLocation, created.ID, tagIDs)
			h.selectedLocID = created.ID
		} else {
			err := h.store.UpdateLocation(Location{
				ID:          h.selectedLocID,
				ProjectID:   h.bookID,
				Name:        name,
				Description: h.locDescEntry.Text,
			}, tagIDs)
			if err != nil {
				dialog.ShowError(err, h.parentWin)
				return
			}
		}
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	})
	saveBtn.Importance = widget.HighImportance

	delBtn := widget.NewButtonWithIcon("Xóa Địa Điểm", theme.DeleteIcon(), func() {
		if h.selectedLocID == 0 {
			return
		}
		_ = h.store.DeleteLocation(h.selectedLocID)
		h.clearLocationForm()
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	})
	delBtn.Importance = widget.DangerImportance

	leftPane := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Lọc theo thẻ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			h.locFilterSelect,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		h.locList,
	)

	rightForm := container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("THÔNG TIN ĐỊA ĐIỂM / BỐI CẢNH", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Tên địa điểm", h.locNameEntry),
			widget.NewFormItem("Mô tả chi tiết", h.locDescEntry),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("GẮN THẺ PHÂN LOẠI (TAGS)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		h.locTagCheck,
		widget.NewSeparator(),
		container.NewHBox(newBtn, layout.NewSpacer(), delBtn, saveBtn),
	))

	split := container.NewHSplit(leftPane, rightForm)
	split.Offset = 0.38
	return split
}

func (h *WorldBuildingHub) clearLocationForm() {
	h.selectedLocID = 0
	h.locList.UnselectAll()
	h.locNameEntry.SetText("")
	h.locDescEntry.SetText("")
	h.locTagCheck.SetSelected([]string{})
}

// ==================== TAB 3: VẬT PHẨM (PROPS) ====================

func (h *WorldBuildingHub) buildPropsTab() fyne.CanvasObject {
	h.propFilterSelect = widget.NewSelect([]string{allTagsFilterLabel}, func(selected string) {
		if selected == "" {
			selected = allTagsFilterLabel
		}
		h.propTagFilter = selected
		h.refreshPropsList()
	})

	h.propList = widget.NewList(
		func() int { return len(h.filteredProps) },
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Tên vật phẩm", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			_, dotWrap := newColorCircleIndicator(DefaultTagColor, 12)
			sub := widget.NewLabel("Phân loại • #Thẻ")
			sub.Truncation = fyne.TextTruncateEllipsis
			subRow := container.NewBorder(nil, nil, container.NewCenter(dotWrap), nil, sub)
			return container.NewVBox(title, subRow)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(h.filteredProps) {
				return
			}
			p := h.filteredProps[i]
			box := obj.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText("🧭 " + p.Name)
			subRow := box.Objects[1].(*fyne.Container)
			for _, child := range subRow.Objects {
				switch w := child.(type) {
				case *widget.Label:
					w.SetText(fmt.Sprintf("[%s]  %s", p.Category, FormatTagNames(p.Tags)))
				case *fyne.Container:
					if len(w.Objects) > 0 {
						if wrap, ok := w.Objects[0].(*fyne.Container); ok && len(wrap.Objects) > 0 {
							if circle, ok := wrap.Objects[0].(*canvas.Circle); ok {
								circle.FillColor = parseHexColor(firstTagColorOrDefault(p.Tags))
								circle.Refresh()
							}
						}
					}
				}
			}
		},
	)

	h.propNameEntry = NewVietEntry()
	h.propNameEntry.SetPlaceHolder("Tên vật phẩm / bảo vật (VD: Thấu Kính Hải Đăng Cổ)...")

	h.propCatEntry = NewVietEntry()
	h.propCatEntry.SetPlaceHolder("Phân loại (VD: Cổ vật, Vũ khí, Thư tịch, Tín vật)...")

	h.propDescEntry = NewVietMultiLineEntry()
	h.propDescEntry.SetPlaceHolder("Hình dáng, chất liệu, nguồn gốc của vật phẩm...")
	h.propDescEntry.SetMinRowsVisible(3)

	h.propSigEntry = NewVietMultiLineEntry()
	h.propSigEntry.SetPlaceHolder("Ý nghĩa cốt truyện, tác động đến nhân vật hoặc bí mật ẩn giấu...")
	h.propSigEntry.SetMinRowsVisible(3)

	h.propTagCheck = widget.NewCheckGroup([]string{}, nil)

	h.propList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(h.filteredProps) {
			return
		}
		p := h.filteredProps[id]
		h.selectedPropID = p.ID
		h.propNameEntry.SetText(p.Name)
		h.propCatEntry.SetText(p.Category)
		h.propDescEntry.SetText(p.Description)
		h.propSigEntry.SetText(p.Significance)
		h.propTagCheck.SetSelected(extractTagNames(p.Tags))
	}

	newBtn := widget.NewButtonWithIcon("Làm mới biểu mẫu", theme.DocumentCreateIcon(), func() {
		h.clearPropForm()
	})

	saveBtn := widget.NewButtonWithIcon("Lưu Vật Phẩm", theme.DocumentSaveIcon(), func() {
		name := strings.TrimSpace(h.propNameEntry.Text)
		if name == "" {
			return
		}
		cat := strings.TrimSpace(h.propCatEntry.Text)
		if cat == "" {
			cat = "Cổ vật"
		}
		tagIDs := h.resolveSelectedTagIDs(h.propTagCheck.Selected)

		if h.selectedPropID == 0 {
			created, err := h.store.CreateProp(h.bookID, name, cat, h.propDescEntry.Text, h.propSigEntry.Text)
			if err != nil {
				dialog.ShowError(err, h.parentWin)
				return
			}
			_ = h.store.SetEntityTags(EntityProp, created.ID, tagIDs)
			h.selectedPropID = created.ID
		} else {
			err := h.store.UpdateProp(Prop{
				ID:           h.selectedPropID,
				BookID:       h.bookID,
				Name:         name,
				Category:     cat,
				Description:  h.propDescEntry.Text,
				Significance: h.propSigEntry.Text,
			}, tagIDs)
			if err != nil {
				dialog.ShowError(err, h.parentWin)
				return
			}
		}
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	})
	saveBtn.Importance = widget.HighImportance

	delBtn := widget.NewButtonWithIcon("Xóa Vật Phẩm", theme.DeleteIcon(), func() {
		if h.selectedPropID == 0 {
			return
		}
		_ = h.store.DeleteProp(h.selectedPropID)
		h.clearPropForm()
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	})
	delBtn.Importance = widget.DangerImportance

	leftPane := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Lọc theo thẻ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			h.propFilterSelect,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		h.propList,
	)

	rightForm := container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("THÔNG TIN VẬT PHẨM (PROPS)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Tên vật phẩm", h.propNameEntry),
			widget.NewFormItem("Phân loại", h.propCatEntry),
			widget.NewFormItem("Mô tả chi tiết", h.propDescEntry),
			widget.NewFormItem("Ý nghĩa cốt truyện", h.propSigEntry),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("GẮN THẺ PHÂN LOẠI (TAGS)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		h.propTagCheck,
		widget.NewSeparator(),
		container.NewHBox(newBtn, layout.NewSpacer(), delBtn, saveBtn),
	))

	split := container.NewHSplit(leftPane, rightForm)
	split.Offset = 0.38
	return split
}

func (h *WorldBuildingHub) clearPropForm() {
	h.selectedPropID = 0
	h.propList.UnselectAll()
	h.propNameEntry.SetText("")
	h.propCatEntry.SetText("")
	h.propDescEntry.SetText("")
	h.propSigEntry.SetText("")
	h.propTagCheck.SetSelected([]string{})
}

// ==================== TAB 4: SỰ KIỆN (EVENTS TIMELINE) ====================

func (h *WorldBuildingHub) buildEventsTab() fyne.CanvasObject {
	h.eventFilterSelect = widget.NewSelect([]string{allTagsFilterLabel}, func(selected string) {
		if selected == "" {
			selected = allTagsFilterLabel
		}
		h.eventTagFilter = selected
		h.refreshEventsList()
	})

	h.eventList = widget.NewList(
		func() int { return len(h.filteredEvents) },
		func() fyne.CanvasObject {
			title := widget.NewLabelWithStyle("Mốc sự kiện", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			_, dotWrap := newColorCircleIndicator(DefaultTagColor, 12)
			sub := widget.NewLabel("#Thẻ")
			sub.Truncation = fyne.TextTruncateEllipsis
			subRow := container.NewBorder(nil, nil, container.NewCenter(dotWrap), nil, sub)
			return container.NewVBox(title, subRow)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(h.filteredEvents) {
				return
			}
			ev := h.filteredEvents[i]
			box := obj.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText(fmt.Sprintf("⏳ [Mốc #%d] %s", ev.TimelineOrder, ev.Title))
			subRow := box.Objects[1].(*fyne.Container)
			for _, child := range subRow.Objects {
				switch w := child.(type) {
				case *widget.Label:
					w.SetText(FormatTagNames(ev.Tags))
				case *fyne.Container:
					if len(w.Objects) > 0 {
						if wrap, ok := w.Objects[0].(*fyne.Container); ok && len(wrap.Objects) > 0 {
							if circle, ok := wrap.Objects[0].(*canvas.Circle); ok {
								circle.FillColor = parseHexColor(firstTagColorOrDefault(ev.Tags))
								circle.Refresh()
							}
						}
					}
				}
			}
		},
	)

	h.eventTitleEntry = NewVietEntry()
	h.eventTitleEntry.SetPlaceHolder("Tiêu đề sự kiện (VD: Đêm Thủy Triều Thấp Rằm Tháng Tám)...")

	h.eventOrderEntry = widget.NewEntry()
	h.eventOrderEntry.SetPlaceHolder("Thứ tự dòng thời gian (VD: 1, 2, 3)...")
	h.eventOrderEntry.SetText("1")

	h.eventDescEntry = NewVietMultiLineEntry()
	h.eventDescEntry.SetPlaceHolder("Diễn biến chính, nguyên nhân và hệ quả của sự kiện đối với mạch truyện...")
	h.eventDescEntry.SetMinRowsVisible(5)

	h.eventTagCheck = widget.NewCheckGroup([]string{}, nil)

	h.eventList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(h.filteredEvents) {
			return
		}
		ev := h.filteredEvents[id]
		h.selectedEventID = ev.ID
		h.eventTitleEntry.SetText(ev.Title)
		h.eventOrderEntry.SetText(fmt.Sprintf("%d", ev.TimelineOrder))
		h.eventDescEntry.SetText(ev.Description)
		h.eventTagCheck.SetSelected(extractTagNames(ev.Tags))
	}

	newBtn := widget.NewButtonWithIcon("Làm mới biểu mẫu", theme.DocumentCreateIcon(), func() {
		h.clearEventForm()
	})

	saveBtn := widget.NewButtonWithIcon("Lưu Sự Kiện", theme.DocumentSaveIcon(), func() {
		title := strings.TrimSpace(h.eventTitleEntry.Text)
		if title == "" {
			return
		}
		order := 1
		_, _ = fmt.Sscanf(h.eventOrderEntry.Text, "%d", &order)
		if order <= 0 {
			order = 1
		}
		tagIDs := h.resolveSelectedTagIDs(h.eventTagCheck.Selected)

		if h.selectedEventID == 0 {
			created, err := h.store.CreateEvent(h.bookID, title, order, h.eventDescEntry.Text)
			if err != nil {
				dialog.ShowError(err, h.parentWin)
				return
			}
			_ = h.store.SetEntityTags(EntityEvent, created.ID, tagIDs)
			h.selectedEventID = created.ID
		} else {
			err := h.store.UpdateEvent(Event{
				ID:            h.selectedEventID,
				BookID:        h.bookID,
				Title:         title,
				TimelineOrder: order,
				Description:   h.eventDescEntry.Text,
			}, tagIDs)
			if err != nil {
				dialog.ShowError(err, h.parentWin)
				return
			}
		}
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	})
	saveBtn.Importance = widget.HighImportance

	delBtn := widget.NewButtonWithIcon("Xóa Sự Kiện", theme.DeleteIcon(), func() {
		if h.selectedEventID == 0 {
			return
		}
		_ = h.store.DeleteEvent(h.selectedEventID)
		h.clearEventForm()
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	})
	delBtn.Importance = widget.DangerImportance

	leftPane := container.NewBorder(
		container.NewVBox(
			widget.NewLabelWithStyle("Lọc theo thẻ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			h.eventFilterSelect,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		h.eventList,
	)

	rightForm := container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("THÔNG TIN SỰ KIỆN DÒNG THỜI GIAN", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Tiêu đề sự kiện", h.eventTitleEntry),
			widget.NewFormItem("Thứ tự thời gian", h.eventOrderEntry),
			widget.NewFormItem("Mô tả diễn biến", h.eventDescEntry),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("GẮN THẺ PHÂN LOẠI (TAGS)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		h.eventTagCheck,
		widget.NewSeparator(),
		container.NewHBox(newBtn, layout.NewSpacer(), delBtn, saveBtn),
	))

	split := container.NewHSplit(leftPane, rightForm)
	split.Offset = 0.38
	return split
}

func (h *WorldBuildingHub) clearEventForm() {
	h.selectedEventID = 0
	h.eventList.UnselectAll()
	h.eventTitleEntry.SetText("")
	h.eventOrderEntry.SetText(fmt.Sprintf("%d", len(h.filteredEvents)+1))
	h.eventDescEntry.SetText("")
	h.eventTagCheck.SetSelected([]string{})
}

// ==================== TAB 5: QUẢN LÝ THẺ TOÀN CỤC & MÀU SẮC THẺ (COLOR-CODED TAG MANAGER) ====================

func (h *WorldBuildingHub) buildTagManagerTab() fyne.CanvasObject {
	presets := DefaultTagColorPresets()
	presetLabels := make([]string, len(presets))
	labelToHex := make(map[string]string, len(presets))
	hexToLabel := make(map[string]string, len(presets))
	for i, p := range presets {
		presetLabels[i] = p.Label
		labelToHex[p.Label] = p.Hex
		hexToLabel[strings.ToLower(p.Hex)] = p.Label
	}

	h.tagFormNameEntry = NewVietEntry()
	h.tagFormNameEntry.SetPlaceHolder("Nhập tên thẻ (VD: Thiên giới, Khu vực cấm, Cổ vật)...")

	h.tagFormHexEntry = NewVietEntry()
	h.tagFormHexEntry.SetPlaceHolder("#3498db")
	h.tagFormHexEntry.SetText(DefaultTagColor)

	// Huy hiệu xem trước màu sắc thẻ trực tiếp (sử dụng canvas.Rectangle & canvas.Circle)
	var previewDotWrap *fyne.Container
	h.tagFormPreviewDot, previewDotWrap = newColorCircleIndicator(DefaultTagColor, 16)
	h.tagFormPreviewBg = canvas.NewRectangle(parseHexTintColor(DefaultTagColor, 42))
	h.tagFormPreviewBg.StrokeColor = parseHexColor(DefaultTagColor)
	h.tagFormPreviewBg.StrokeWidth = 1.5
	h.tagFormPreviewBg.CornerRadius = 6
	h.tagFormPreviewLabel = widget.NewLabelWithStyle("🏷️ #Xem trước thẻ  (#3498db)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	previewBadge := container.NewMax(
		h.tagFormPreviewBg,
		container.NewPadded(container.NewHBox(
			container.NewCenter(previewDotWrap),
			h.tagFormPreviewLabel,
		)),
	)

	updateLiveTagPreview := func() {
		hex := NormalizeHexColor(h.tagFormHexEntry.Text)
		tagName := strings.TrimSpace(h.tagFormNameEntry.Text)
		if tagName == "" {
			tagName = "Xem trước thẻ"
		}
		h.tagFormPreviewDot.FillColor = parseHexColor(hex)
		h.tagFormPreviewDot.Refresh()
		h.tagFormPreviewBg.FillColor = parseHexTintColor(hex, 42)
		h.tagFormPreviewBg.StrokeColor = parseHexColor(hex)
		h.tagFormPreviewBg.Refresh()
		h.tagFormPreviewLabel.SetText(fmt.Sprintf("🏷️ #%s  (%s)", tagName, hex))
	}

	h.tagFormNameEntry.SetOnChangedCallback(func(_ string) {
		updateLiveTagPreview()
	})
	h.tagFormHexEntry.SetOnChangedCallback(func(_ string) {
		updateLiveTagPreview()
	})

	h.tagFormPresetSelect = widget.NewSelect(presetLabels, func(selected string) {
		if hex, ok := labelToHex[selected]; ok {
			h.tagFormHexEntry.SetText(hex)
			updateLiveTagPreview()
		}
	})
	if len(presetLabels) > 0 {
		h.tagFormPresetSelect.SetSelected(presetLabels[0])
	}

	// Bảng nút chọn màu nhanh trực quan
	paletteGrid := container.NewGridWithColumns(3)
	for _, preset := range presets {
		pHex := preset.Hex
		pLabel := preset.Label
		_, dotBox := newColorCircleIndicator(pHex, 14)
		colorBtn := widget.NewButton(pLabel, func() {
			h.tagFormHexEntry.SetText(pHex)
			h.tagFormPresetSelect.SetSelected(pLabel)
			updateLiveTagPreview()
		})
		colorBtn.Importance = widget.LowImportance
		paletteGrid.Add(container.NewBorder(nil, nil, container.NewCenter(dotBox), nil, colorBtn))
	}

	// Danh sách Thẻ bên trái với huy hiệu màu sắc (canvas.Circle + canvas.Rectangle)
	h.tagList = widget.NewList(
		func() int { return len(h.tags) },
		func() fyne.CanvasObject {
			bgRect := canvas.NewRectangle(parseHexTintColor(DefaultTagColor, 32))
			bgRect.CornerRadius = 6
			bgRect.StrokeWidth = 1
			bgRect.StrokeColor = parseHexColor(DefaultTagColor)

			_, dotWrap := newColorCircleIndicator(DefaultTagColor, 16)
			nameLbl := widget.NewLabelWithStyle("#Tên thẻ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			hexLbl := widget.NewLabelWithStyle("#3498db", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})

			editBtn := widget.NewButtonWithIcon("Chỉnh sửa thẻ", theme.DocumentCreateIcon(), nil)
			editBtn.Importance = widget.LowImportance

			delBtn := widget.NewButtonWithIcon("Xóa thẻ", theme.DeleteIcon(), nil)
			delBtn.Importance = widget.DangerImportance

			leftInfo := container.NewHBox(container.NewCenter(dotWrap), nameLbl, hexLbl)
			rightActions := container.NewHBox(editBtn, delBtn)
			rowContent := container.NewBorder(nil, nil, nil, rightActions, leftInfo)

			return container.NewMax(bgRect, container.NewPadded(rowContent))
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(h.tags) {
				return
			}
			tag := h.tags[i]
			tagColor := NormalizeHexColor(tag.Color)

			maxBox := obj.(*fyne.Container)
			if bgRect, ok := maxBox.Objects[0].(*canvas.Rectangle); ok {
				bgRect.FillColor = parseHexTintColor(tagColor, 30)
				bgRect.StrokeColor = parseHexColor(tagColor)
				bgRect.Refresh()
			}

			paddedBox := maxBox.Objects[1].(*fyne.Container)
			borderBox := paddedBox.Objects[0].(*fyne.Container)

			for _, child := range borderBox.Objects {
				hbox, ok := child.(*fyne.Container)
				if !ok {
					continue
				}
				if len(hbox.Objects) == 3 {
					// leftInfo: [center(dotWrap), nameLbl, hexLbl]
					if centerWrap, ok := hbox.Objects[0].(*fyne.Container); ok && len(centerWrap.Objects) > 0 {
						if gridWrap, ok := centerWrap.Objects[0].(*fyne.Container); ok && len(gridWrap.Objects) > 0 {
							if circle, ok := gridWrap.Objects[0].(*canvas.Circle); ok {
								circle.FillColor = parseHexColor(tagColor)
								circle.Refresh()
							}
						}
					}
					if nameLbl, ok := hbox.Objects[1].(*widget.Label); ok {
						nameLbl.SetText(fmt.Sprintf("#%s", tag.Name))
					}
					if hexLbl, ok := hbox.Objects[2].(*widget.Label); ok {
						hexLbl.SetText(fmt.Sprintf("(%s)", tagColor))
					}
				} else if len(hbox.Objects) == 2 {
					// rightActions: [editBtn, delBtn]
					currentTag := tag
					if editBtn, ok := hbox.Objects[0].(*widget.Button); ok {
						editBtn.OnTapped = func() {
							h.showEditTagDialog(currentTag)
						}
					}
					if delBtn, ok := hbox.Objects[1].(*widget.Button); ok {
						delBtn.OnTapped = func() {
							_ = h.store.DeleteTag(currentTag.ID)
							if h.selectedTagID == currentTag.ID {
								h.clearTagForm()
							}
							h.reloadAllData()
							if h.onUpdated != nil {
								h.onUpdated()
							}
						}
					}
				}
			}
		},
	)

	h.tagList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(h.tags) {
			return
		}
		t := h.tags[id]
		h.selectedTagID = t.ID
		h.tagFormNameEntry.SetText(t.Name)
		h.tagFormHexEntry.SetText(NormalizeHexColor(t.Color))
		if lbl, ok := hexToLabel[strings.ToLower(NormalizeHexColor(t.Color))]; ok {
			h.tagFormPresetSelect.SetSelected(lbl)
		}
		updateLiveTagPreview()
	}

	resetBtn := widget.NewButtonWithIcon("Thêm thẻ mới (Làm trống)", theme.DocumentCreateIcon(), func() {
		h.clearTagForm()
		updateLiveTagPreview()
	})

	saveTagBtn := widget.NewButtonWithIcon("Lưu thẻ", theme.DocumentSaveIcon(), func() {
		name := strings.TrimSpace(h.tagFormNameEntry.Text)
		if name == "" {
			return
		}
		hexColor := NormalizeHexColor(h.tagFormHexEntry.Text)
		if h.selectedTagID == 0 {
			created, err := h.store.CreateTag(h.bookID, name, hexColor)
			if err != nil {
				dialog.ShowError(err, h.parentWin)
				return
			}
			h.selectedTagID = created.ID
		} else {
			if err := h.store.UpdateTag(h.selectedTagID, name, hexColor); err != nil {
				dialog.ShowError(err, h.parentWin)
				return
			}
		}
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	})
	saveTagBtn.Importance = widget.HighImportance

	delTagBtn := widget.NewButtonWithIcon("Xóa thẻ đang chọn", theme.DeleteIcon(), func() {
		if h.selectedTagID == 0 {
			return
		}
		_ = h.store.DeleteTag(h.selectedTagID)
		h.clearTagForm()
		updateLiveTagPreview()
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	})
	delTagBtn.Importance = widget.DangerImportance

	info := widget.NewLabel("Danh sách các Thẻ phân loại (Tags) kèm Màu sắc thẻ. Nhấp vào một thẻ để chỉnh sửa tên/màu sắc hoặc nhấn 'Chỉnh sửa thẻ'.")
	info.Wrapping = fyne.TextWrapWord

	leftPane := container.NewBorder(
		container.NewVBox(info, widget.NewSeparator()),
		nil, nil, nil,
		h.tagList,
	)

	rightForm := container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("THÊM THẺ MỚI / CHỈNH SỬA THẺ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Tên thẻ", h.tagFormNameEntry),
			widget.NewFormItem("Màu sắc thẻ (Gợi ý)", h.tagFormPresetSelect),
			widget.NewFormItem("Mã màu Hex", h.tagFormHexEntry),
			widget.NewFormItem("Xem trước huy hiệu", previewBadge),
		),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("BẢNG CHỌN MÀU NHANH", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		paletteGrid,
		widget.NewSeparator(),
		container.NewHBox(resetBtn, layout.NewSpacer(), delTagBtn, saveTagBtn),
	))

	split := container.NewHSplit(leftPane, rightForm)
	split.Offset = 0.48
	return split
}

func (h *WorldBuildingHub) clearTagForm() {
	h.selectedTagID = 0
	if h.tagList != nil {
		h.tagList.UnselectAll()
	}
	if h.tagFormNameEntry != nil {
		h.tagFormNameEntry.SetText("")
	}
	if h.tagFormHexEntry != nil {
		h.tagFormHexEntry.SetText(DefaultTagColor)
	}
}

// showEditTagDialog hiển thị hộp thoại chỉnh sửa nhanh tên và màu sắc của một Thẻ.
func (h *WorldBuildingHub) showEditTagDialog(tag Tag) {
	nameEntry := NewVietEntry()
	nameEntry.SetText(tag.Name)

	hexEntry := NewVietEntry()
	hexEntry.SetText(NormalizeHexColor(tag.Color))

	dotCircle, dotWrap := newColorCircleIndicator(tag.Color, 20)

	presets := DefaultTagColorPresets()
	presetLabels := make([]string, len(presets))
	labelToHex := make(map[string]string, len(presets))
	for i, p := range presets {
		presetLabels[i] = p.Label
		labelToHex[p.Label] = p.Hex
	}

	colorSelect := widget.NewSelect(presetLabels, func(selected string) {
		if hex, ok := labelToHex[selected]; ok {
			hexEntry.SetText(hex)
			dotCircle.FillColor = parseHexColor(hex)
			dotCircle.Refresh()
		}
	})
	hexEntry.SetOnChangedCallback(func(val string) {
		dotCircle.FillColor = parseHexColor(val)
		dotCircle.Refresh()
	})

	hexRow := container.NewBorder(nil, nil, container.NewCenter(dotWrap), nil, hexEntry)

	dialog.ShowForm("Chỉnh sửa thẻ", "Lưu thay đổi", "Hủy", []*widget.FormItem{
		widget.NewFormItem("Tên thẻ", nameEntry),
		widget.NewFormItem("Chọn màu", colorSelect),
		widget.NewFormItem("Màu sắc thẻ (Hex)", hexRow),
	}, func(confirmed bool) {
		if !confirmed || strings.TrimSpace(nameEntry.Text) == "" {
			return
		}
		if err := h.store.UpdateTag(tag.ID, nameEntry.Text, hexEntry.Text); err != nil {
			dialog.ShowError(err, h.parentWin)
			return
		}
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	}, h.parentWin)
}

// ==================== NẠP DỮ LIỆU & ÁP DỤNG BỘ LỌC THẺ ====================

func (h *WorldBuildingHub) reloadAllData() {
	tags, _ := h.store.ListTags(h.bookID)
	h.tags = tags

	tagNames := make([]string, 0, len(tags))
	filterOpts := []string{allTagsFilterLabel}
	for _, t := range tags {
		tagNames = append(tagNames, t.Name)
		filterOpts = append(filterOpts, t.Name)
	}

	// Cập nhật danh sách chọn Thẻ trong các biểu mẫu
	h.charTagCheck.Options = tagNames
	h.charTagCheck.Refresh()
	h.locTagCheck.Options = tagNames
	h.locTagCheck.Refresh()
	h.propTagCheck.Options = tagNames
	h.propTagCheck.Refresh()
	h.eventTagCheck.Options = tagNames
	h.eventTagCheck.Refresh()

	// Cập nhật các bộ lọc theo Thẻ
	h.charFilterSelect.Options = filterOpts
	if h.charFilterSelect.Selected == "" {
		h.charFilterSelect.SetSelected(allTagsFilterLabel)
	}
	h.locFilterSelect.Options = filterOpts
	if h.locFilterSelect.Selected == "" {
		h.locFilterSelect.SetSelected(allTagsFilterLabel)
	}
	h.propFilterSelect.Options = filterOpts
	if h.propFilterSelect.Selected == "" {
		h.propFilterSelect.SetSelected(allTagsFilterLabel)
	}
	h.eventFilterSelect.Options = filterOpts
	if h.eventFilterSelect.Selected == "" {
		h.eventFilterSelect.SetSelected(allTagsFilterLabel)
	}

	h.refreshCharactersList()
	h.refreshLocationsList()
	h.refreshPropsList()
	h.refreshEventsList()
	if h.tagList != nil {
		h.tagList.Refresh()
	}
}

func (h *WorldBuildingHub) refreshCharactersList() {
	all, _ := h.store.ListCharacters(h.bookID)
	var out []Character
	for _, c := range all {
		if HasTagName(c.Tags, h.charTagFilter) {
			out = append(out, c)
		}
	}
	h.filteredChars = out
	if h.charList != nil {
		h.charList.Refresh()
	}
}

func (h *WorldBuildingHub) refreshLocationsList() {
	all, _ := h.store.ListLocations(h.bookID)
	var out []Location
	for _, l := range all {
		if HasTagName(l.Tags, h.locTagFilter) {
			out = append(out, l)
		}
	}
	h.filteredLocs = out
	if h.locList != nil {
		h.locList.Refresh()
	}
}

func (h *WorldBuildingHub) refreshPropsList() {
	all, _ := h.store.ListProps(h.bookID)
	var out []Prop
	for _, p := range all {
		if HasTagName(p.Tags, h.propTagFilter) {
			out = append(out, p)
		}
	}
	h.filteredProps = out
	if h.propList != nil {
		h.propList.Refresh()
	}
}

func (h *WorldBuildingHub) refreshEventsList() {
	all, _ := h.store.ListEvents(h.bookID)
	var out []Event
	for _, ev := range all {
		if HasTagName(ev.Tags, h.eventTagFilter) {
			out = append(out, ev)
		}
	}
	h.filteredEvents = out
	if h.eventList != nil {
		h.eventList.Refresh()
	}
}

func (h *WorldBuildingHub) resolveSelectedTagIDs(selectedNames []string) []int64 {
	var ids []int64
	for _, name := range selectedNames {
		for _, t := range h.tags {
			if t.Name == name {
				ids = append(ids, t.ID)
				break
			}
		}
	}
	return ids
}

func extractTagNames(tags []Tag) []string {
	out := make([]string, len(tags))
	for i, t := range tags {
		out[i] = t.Name
	}
	return out
}
