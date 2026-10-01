package main

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

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

	// Biểu mẫu Tab 1: Nhân vật
	charNameEntry *widget.Entry
	charRoleEntry *widget.Entry
	charDescEntry *widget.Entry
	charTagCheck  *widget.CheckGroup

	// Biểu mẫu Tab 2: Địa điểm
	locNameEntry *widget.Entry
	locDescEntry *widget.Entry
	locTagCheck  *widget.CheckGroup

	// Biểu mẫu Tab 3: Vật phẩm
	propNameEntry *widget.Entry
	propCatEntry  *widget.Entry
	propDescEntry *widget.Entry
	propSigEntry  *widget.Entry
	propTagCheck  *widget.CheckGroup

	// Biểu mẫu Tab 4: Sự kiện
	eventTitleEntry *widget.Entry
	eventOrderEntry *widget.Entry
	eventDescEntry  *widget.Entry
	eventTagCheck   *widget.CheckGroup

	// Biểu mẫu Quản lý Thẻ toàn cục
	newTagEntry *widget.Entry
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
	// Thanh tạo Thẻ nhanh ở trên cùng (Global Tag Bar)
	h.newTagEntry = NewVietEntry()
	h.newTagEntry.SetPlaceHolder("Nhập tên thẻ mới (VD: Thiên giới, Khu vực cấm, Cổ vật)...")

	quickAddTagBtn := widget.NewButtonWithIcon("Tạo Thẻ Mới", theme.ContentAddIcon(), func() {
		name := strings.TrimSpace(h.newTagEntry.Text)
		if name == "" {
			return
		}
		if _, err := h.store.CreateTag(h.bookID, name); err != nil {
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

	globalTagHeader := container.NewBorder(
		nil,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("🏷️ Quản lý Thẻ nhanh:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		quickAddTagBtn,
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
			sub := widget.NewLabel("Vai trò • #Thẻ")
			sub.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(title, sub)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(h.filteredChars) {
				return
			}
			c := h.filteredChars[i]
			box := obj.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText("👤 " + c.Name)
			box.Objects[1].(*widget.Label).SetText(fmt.Sprintf("%s  |  %s", c.Role, FormatTagNames(c.Tags)))
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
			sub := widget.NewLabel("#Thẻ")
			sub.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(title, sub)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(h.filteredLocs) {
				return
			}
			l := h.filteredLocs[i]
			box := obj.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText("🏛️ " + l.Name)
			box.Objects[1].(*widget.Label).SetText(FormatTagNames(l.Tags))
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
			sub := widget.NewLabel("Phân loại • #Thẻ")
			sub.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(title, sub)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(h.filteredProps) {
				return
			}
			p := h.filteredProps[i]
			box := obj.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText("🧭 " + p.Name)
			box.Objects[1].(*widget.Label).SetText(fmt.Sprintf("[%s]  %s", p.Category, FormatTagNames(p.Tags)))
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
			sub := widget.NewLabel("#Thẻ")
			sub.Truncation = fyne.TextTruncateEllipsis
			return container.NewVBox(title, sub)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(h.filteredEvents) {
				return
			}
			ev := h.filteredEvents[i]
			box := obj.(*fyne.Container)
			box.Objects[0].(*widget.Label).SetText(fmt.Sprintf("⏳ [Mốc #%d] %s", ev.TimelineOrder, ev.Title))
			box.Objects[1].(*widget.Label).SetText(FormatTagNames(ev.Tags))
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

// ==================== TAB 5: QUẢN LÝ THẺ TOÀN CỤC (GLOBAL TAG MANAGER) ====================

func (h *WorldBuildingHub) buildTagManagerTab() fyne.CanvasObject {
	h.tagList = widget.NewList(
		func() int { return len(h.tags) },
		func() fyne.CanvasObject {
			nameLbl := widget.NewLabelWithStyle("#Tên thẻ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
			delBtn := widget.NewButtonWithIcon("Xóa thẻ", theme.DeleteIcon(), nil)
			delBtn.Importance = widget.DangerImportance
			return container.NewBorder(nil, nil, nil, delBtn, nameLbl)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(h.tags) {
				return
			}
			tag := h.tags[i]
			box := obj.(*fyne.Container)
			for _, child := range box.Objects {
				switch w := child.(type) {
				case *widget.Label:
					w.SetText(fmt.Sprintf("🏷️ #%s  (ID: %d)", tag.Name, tag.ID))
				case *widget.Button:
					tagID := tag.ID
					w.OnTapped = func() {
						_ = h.store.DeleteTag(tagID)
						h.reloadAllData()
						if h.onUpdated != nil {
							h.onUpdated()
						}
					}
				}
			}
		},
	)

	info := widget.NewLabel("Danh sách toàn bộ các Thẻ (Tags) của tác phẩm. Bạn có thể gắn các thẻ này cho Nhân vật, Địa điểm, Vật phẩm và Sự kiện để lọc nhanh theo chủ đề (VD: 'Thiên giới', 'Khu vực cấm', 'Cổ vật').")
	info.Wrapping = fyne.TextWrapWord

	return container.NewBorder(
		container.NewVBox(info, widget.NewSeparator()),
		nil, nil, nil,
		h.tagList,
	)
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
