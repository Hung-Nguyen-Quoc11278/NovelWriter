package main

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
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

// colorToHex chuyển đổi đối tượng color.Color bất kỳ từ hộp thoại chọn màu của Fyne sang mã Hex "#rrggbb".
func colorToHex(c color.Color) string {
	if c == nil {
		return DefaultTagColor
	}
	r, g, b, _ := c.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
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

// ============================================================================
// Ô MÀU TRỰC QUAN & BẢNG MÀU NHANH (VISUAL COLOR SWATCH & "+" CUSTOM PICKER)
// ============================================================================

// ColorSwatchWidget là một ô vuông màu sắc (Custom Fyne Widget) có thể nhấp chuột trực tiếp.
// Hỗ trợ 2 chế độ:
//   - Ô màu có sẵn (IsCustomPicker = false): Tô màu nền theo mã Hex; khi được chọn sẽ hiển thị viền trắng dày và dấu kiểm "✓".
//   - Ô chọn màu tùy chỉnh "+" (IsCustomPicker = true): Có viền trắng nổi bật và dấu "+" ở chính giữa;
//     khi nhấp sẽ mở hộp thoại bảng màu tự do (dialog.NewColorPicker).
type ColorSwatchWidget struct {
	widget.BaseWidget
	Hex            string
	IsCustomPicker bool
	Selected       bool
	SwatchSize     float32
	OnTappedSwatch func()
}

// NewPresetColorSwatch khởi tạo một ô vuông màu sắc mẫu (Preset Swatch).
func NewPresetColorSwatch(hex string, size float32, onTap func()) *ColorSwatchWidget {
	w := &ColorSwatchWidget{
		Hex:            NormalizeHexColor(hex),
		IsCustomPicker: false,
		SwatchSize:     size,
		OnTappedSwatch: onTap,
	}
	w.ExtendBaseWidget(w)
	return w
}

// NewCustomColorPickerSwatch khởi tạo ô vuông đặc biệt có viền trắng và biểu tượng "+" ở chính giữa
// để mở bảng chọn màu tùy chỉnh (Custom Color Picker).
func NewCustomColorPickerSwatch(size float32, onTap func()) *ColorSwatchWidget {
	w := &ColorSwatchWidget{
		Hex:            "#1e293b",
		IsCustomPicker: true,
		SwatchSize:     size,
		OnTappedSwatch: onTap,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *ColorSwatchWidget) Tapped(_ *fyne.PointEvent) {
	PlayUIClickSound()
	if w.OnTappedSwatch != nil {
		w.OnTappedSwatch()
	}
}

func (w *ColorSwatchWidget) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (w *ColorSwatchWidget) CreateRenderer() fyne.WidgetRenderer {
	outerRing := canvas.NewRectangle(color.Transparent)
	outerRing.CornerRadius = 7

	bgRect := canvas.NewRectangle(parseHexColor(w.Hex))
	bgRect.CornerRadius = 6

	symbolText := canvas.NewText("", color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	symbolText.Alignment = fyne.TextAlignCenter
	symbolText.TextStyle = fyne.TextStyle{Bold: true}

	r := &colorSwatchRenderer{
		swatch:     w,
		outerRing:  outerRing,
		bgRect:     bgRect,
		symbolText: symbolText,
	}
	r.Refresh()
	return r
}

type colorSwatchRenderer struct {
	swatch     *ColorSwatchWidget
	outerRing  *canvas.Rectangle
	bgRect     *canvas.Rectangle
	symbolText *canvas.Text
}

func (r *colorSwatchRenderer) Layout(size fyne.Size) {
	r.outerRing.Move(fyne.NewPos(0, 0))
	r.outerRing.Resize(size)

	inset := float32(2)
	r.bgRect.Move(fyne.NewPos(inset, inset))
	r.bgRect.Resize(fyne.NewSize(size.Width-inset*2, size.Height-inset*2))

	textMin := r.symbolText.MinSize()
	r.symbolText.Move(fyne.NewPos((size.Width-textMin.Width)/2, (size.Height-textMin.Height)/2))
	r.symbolText.Resize(textMin)
}

func (r *colorSwatchRenderer) MinSize() fyne.Size {
	sz := r.swatch.SwatchSize
	if sz < 24 {
		sz = 34
	}
	return fyne.NewSize(sz, sz)
}

func (r *colorSwatchRenderer) Refresh() {
	fill := parseHexColor(r.swatch.Hex)
	r.bgRect.FillColor = fill

	if r.swatch.IsCustomPicker {
		// Ô chọn màu tùy chỉnh ("+"): Luôn có viền trắng rõ nét và dấu "+" ở tâm
		r.bgRect.StrokeColor = color.NRGBA{R: 255, G: 255, B: 255, A: 245}
		if r.swatch.Selected {
			r.bgRect.StrokeWidth = 2.6
			r.outerRing.StrokeColor = fill
			r.outerRing.StrokeWidth = 1.6
			r.symbolText.Text = "+✓"
			r.symbolText.TextSize = 13
		} else {
			r.bgRect.StrokeWidth = 2.0
			r.outerRing.StrokeColor = color.Transparent
			r.outerRing.StrokeWidth = 0
			r.symbolText.Text = "+"
			r.symbolText.TextSize = 18
		}
		r.symbolText.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	} else {
		// Ô màu có sẵn (Preset Swatch): Hiển thị viền trắng nổi bật và dấu "✓" khi đang được chọn
		if r.swatch.Selected {
			r.bgRect.StrokeColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			r.bgRect.StrokeWidth = 2.6
			r.outerRing.StrokeColor = fill
			r.outerRing.StrokeWidth = 1.6
			r.symbolText.Text = "✓"
			r.symbolText.TextSize = 14
			r.symbolText.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		} else {
			r.bgRect.StrokeColor = color.NRGBA{R: 0, G: 0, B: 0, A: 70}
			r.bgRect.StrokeWidth = 1.0
			r.outerRing.StrokeColor = color.Transparent
			r.outerRing.StrokeWidth = 0
			r.symbolText.Text = ""
		}
	}

	r.outerRing.Refresh()
	r.bgRect.Refresh()
	r.symbolText.Refresh()
	r.Layout(r.swatch.Size())
}

func (r *colorSwatchRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.outerRing, r.bgRect, r.symbolText}
}

func (r *colorSwatchRenderer) Destroy() {}

// TagColorSwatchPicker quản lý lưới các ô màu vuông trực quan (Preset Swatches)
// kèm ô "+" có viền trắng để mở bảng chọn màu tùy chỉnh (Fyne Color Picker Dialog).
type TagColorSwatchPicker struct {
	parentWin      fyne.Window
	selectedHex    string
	swatchSize     float32
	presetSwatches []*ColorSwatchWidget
	customSwatch   *ColorSwatchWidget
	container      *fyne.Container
	onChanged      func(hex string)
}

// NewTagColorSwatchPicker khởi tạo bộ chọn màu bằng lưới ô vuông trực quan và nút "+" chọn màu tùy chỉnh.
func NewTagColorSwatchPicker(parentWin fyne.Window, initialHex string, swatchSize float32, onChanged func(hex string)) *TagColorSwatchPicker {
	p := &TagColorSwatchPicker{
		parentWin:   parentWin,
		selectedHex: NormalizeHexColor(initialHex),
		swatchSize:  swatchSize,
		onChanged:   onChanged,
	}

	presets := DefaultTagColorPresets()
	objects := make([]fyne.CanvasObject, 0, len(presets)+1)

	for _, preset := range presets {
		presetHex := NormalizeHexColor(preset.Hex)
		sw := NewPresetColorSwatch(presetHex, swatchSize, func() {
			p.SelectHex(presetHex, true)
		})
		p.presetSwatches = append(p.presetSwatches, sw)
		objects = append(objects, sw)
	}

	// Ô vuông đặc biệt có viền trắng và dấu "+" ở chính giữa để mở bảng màu tùy chỉnh (Custom Color Picker)
	p.customSwatch = NewCustomColorPickerSwatch(swatchSize, func() {
		p.openCustomColorPickerDialog()
	})
	objects = append(objects, p.customSwatch)

	wrapLayout := NewResponsiveToolbarWrapLayout(8, 8, false, nil)
	p.container = container.New(wrapLayout, objects...)
	p.SelectHex(p.selectedHex, false)
	return p
}

func (p *TagColorSwatchPicker) openCustomColorPickerDialog() {
	if p.parentWin == nil {
		return
	}
	picker := dialog.NewColorPicker(
		"Chọn màu tùy chỉnh",
		"Chọn màu sắc bất kỳ trên bảng màu cho thẻ:",
		func(c color.Color) {
			hex := colorToHex(c)
			p.SelectHex(hex, true)
		},
		p.parentWin,
	)
	picker.Advanced = true
	attachWindowRefreshOnClose(picker, p.parentWin)
	picker.SetColor(parseHexColor(p.selectedHex))
	picker.Show()
}

// Container trả về đối tượng fyne.CanvasObject chứa lưới các ô màu vuông và nút "+".
func (p *TagColorSwatchPicker) Container() *fyne.Container {
	return p.container
}

// SelectedHex trả về mã màu Hex đang được chọn.
func (p *TagColorSwatchPicker) SelectedHex() string {
	return p.selectedHex
}

// SelectHex cập nhật trạng thái ô màu đang chọn (dấu kiểm "✓" trên ô màu mẫu hoặc trên ô "+")
// và tùy chọn kích hoạt callback onChanged.
func (p *TagColorSwatchPicker) SelectHex(rawHex string, notify bool) {
	norm := NormalizeHexColor(rawHex)
	p.selectedHex = norm

	matchedPreset := false
	for _, sw := range p.presetSwatches {
		if strings.EqualFold(sw.Hex, norm) {
			sw.Selected = true
			matchedPreset = true
		} else {
			sw.Selected = false
		}
		sw.Refresh()
	}

	if p.customSwatch != nil {
		if !matchedPreset {
			p.customSwatch.Hex = norm
			p.customSwatch.Selected = true
		} else {
			p.customSwatch.Hex = "#1e293b"
			p.customSwatch.Selected = false
		}
		p.customSwatch.Refresh()
	}

	if notify && p.onChanged != nil {
		p.onChanged(norm)
	}
}

// ============================================================================
// WIDGET Ô CHỌN THẺ VỚI MÀU CHỮ ĐỘNG (COLORED TAG CHECKBOX & CHECKGROUP)
// ============================================================================

// ColoredTagCheckbox là một widget hiển thị một ô chọn Thẻ với màu sắc tùy chỉnh.
// Nhãn văn bản (Text Label) kế thừa và hiển thị chính xác mã màu Hex của Thẻ thông qua canvas.Text.
type ColoredTagCheckbox struct {
	widget.BaseWidget
	Tag       Tag
	Checked   bool
	OnChanged func(checked bool)
}

// NewColoredTagCheckbox khởi tạo một ô chọn Thẻ có màu chữ và màu viền đồng bộ với mã màu của Thẻ.
func NewColoredTagCheckbox(tag Tag, checked bool, onChanged func(checked bool)) *ColoredTagCheckbox {
	c := &ColoredTagCheckbox{
		Tag:       tag,
		Checked:   checked,
		OnChanged: onChanged,
	}
	c.ExtendBaseWidget(c)
	return c
}

func (c *ColoredTagCheckbox) Tapped(_ *fyne.PointEvent) {
	PlayUIClickSound()
	c.Checked = !c.Checked
	c.Refresh()
	if c.OnChanged != nil {
		c.OnChanged(c.Checked)
	}
}

func (c *ColoredTagCheckbox) Cursor() desktop.Cursor {
	return desktop.PointerCursor
}

func (c *ColoredTagCheckbox) SetChecked(checked bool) {
	if c.Checked != checked {
		c.Checked = checked
		c.Refresh()
	}
}

type coloredTagCheckboxRenderer struct {
	checkbox  *ColoredTagCheckbox
	bgRect    *canvas.Rectangle
	checkBg   *canvas.Rectangle
	checkMark *canvas.Text
	dot       *canvas.Circle
	label     *canvas.Text
	container *fyne.Container
}

func (c *ColoredTagCheckbox) CreateRenderer() fyne.WidgetRenderer {
	tagCol := parseHexColor(c.Tag.Color)

	bgRect := canvas.NewRectangle(color.Transparent)
	bgRect.CornerRadius = 5

	checkBg := canvas.NewRectangle(color.Transparent)
	checkBg.CornerRadius = 3

	checkMark := canvas.NewText("", color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	checkMark.Alignment = fyne.TextAlignCenter
	checkMark.TextStyle = fyne.TextStyle{Bold: true}
	checkMark.TextSize = 11

	checkStack := container.NewStack(checkBg, container.NewCenter(checkMark))
	checkWrap := container.NewGridWrap(fyne.NewSize(18, 18), checkStack)

	dot := canvas.NewCircle(tagCol)
	dotWrap := container.NewGridWrap(fyne.NewSize(10, 10), dot)

	// Áp dụng trực tiếp mã màu Hex của Thẻ lên nhãn văn bản bằng canvas.Text
	label := canvas.NewText("# "+c.Tag.Name, tagCol)
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.TextSize = 13

	row := container.NewHBox(
		container.NewCenter(checkWrap),
		container.NewCenter(dotWrap),
		label,
	)
	padded := container.NewPadded(row)
	root := container.NewMax(bgRect, padded)

	r := &coloredTagCheckboxRenderer{
		checkbox:  c,
		bgRect:    bgRect,
		checkBg:   checkBg,
		checkMark: checkMark,
		dot:       dot,
		label:     label,
		container: root,
	}
	r.Refresh()
	return r
}

func (r *coloredTagCheckboxRenderer) Layout(size fyne.Size) {
	r.container.Move(fyne.NewPos(0, 0))
	r.container.Resize(size)
}

func (r *coloredTagCheckboxRenderer) MinSize() fyne.Size {
	return r.container.MinSize()
}

func (r *coloredTagCheckboxRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.container}
}

func (r *coloredTagCheckboxRenderer) Destroy() {}

func (r *coloredTagCheckboxRenderer) Refresh() {
	tagCol := parseHexColor(r.checkbox.Tag.Color)

	// Nhãn chữ kế thừa chính xác mã màu Hex của Thẻ
	r.label.Text = "# " + r.checkbox.Tag.Name
	r.label.Color = tagCol
	r.label.Refresh()

	r.dot.FillColor = tagCol
	r.dot.Refresh()

	if r.checkbox.Checked {
		r.bgRect.FillColor = parseHexTintColor(r.checkbox.Tag.Color, 28)
		r.bgRect.StrokeColor = parseHexTintColor(r.checkbox.Tag.Color, 85)
		r.bgRect.StrokeWidth = 1.0

		r.checkBg.FillColor = tagCol
		r.checkBg.StrokeColor = tagCol
		r.checkBg.StrokeWidth = 1.0

		r.checkMark.Text = "✓"
	} else {
		r.bgRect.FillColor = color.Transparent
		r.bgRect.StrokeColor = color.NRGBA{R: 0, G: 0, B: 0, A: 25}
		r.bgRect.StrokeWidth = 0.5

		r.checkBg.FillColor = color.Transparent
		r.checkBg.StrokeColor = color.NRGBA{R: 140, G: 140, B: 140, A: 200}
		r.checkBg.StrokeWidth = 1.5

		r.checkMark.Text = ""
	}

	r.bgRect.Refresh()
	r.checkBg.Refresh()
	r.checkMark.Refresh()
	r.container.Refresh()
}

// ColoredTagCheckGroup quản lý danh sách các ô chọn Thẻ có màu sắc động,
// cung cấp API tương thích với CheckGroup (Options, Selected, SetOptions, SetSelected, SetTags, Container).
type ColoredTagCheckGroup struct {
	container *fyne.Container
	Options   []string
	tags      []Tag
	tagMap    map[string]Tag
	items     []*ColoredTagCheckbox
	Selected  []string
	OnChanged func(selected []string)
}

// NewColoredTagCheckGroup khởi tạo một nhóm chọn Thẻ với màu sắc văn bản tùy chỉnh.
func NewColoredTagCheckGroup(onChanged func(selected []string)) *ColoredTagCheckGroup {
	g := &ColoredTagCheckGroup{
		OnChanged: onChanged,
		tagMap:    make(map[string]Tag),
	}
	emptyLbl := widget.NewLabelWithStyle("(Chưa có thẻ nào cho danh mục này. Nhấn nút 'Thêm Thẻ...' ở trên để tạo thẻ mới)", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
	g.container = container.NewVBox(emptyLbl)
	return g
}

// Container trả về đối tượng *fyne.Container chứa các ô chọn thẻ để đưa vào cây giao diện Fyne.
func (g *ColoredTagCheckGroup) Container() *fyne.Container {
	return g.container
}

// Refresh làm mới container hiển thị các ô chọn thẻ.
func (g *ColoredTagCheckGroup) Refresh() {
	if g.container != nil {
		g.container.Refresh()
	}
}

// SetOptions thiết lập danh sách tên thẻ theo danh mục (category-specific tag names) kèm danh sách Thẻ,
// đảm bảo chỉ các thẻ thuộc danh mục này được hiển thị và màu sắc được kế thừa chính xác.
func (g *ColoredTagCheckGroup) SetOptions(options []string, tags []Tag) {
	g.Options = append([]string{}, options...)
	g.tags = tags
	g.tagMap = make(map[string]Tag, len(tags))
	for _, t := range tags {
		g.tagMap[t.Name] = t
	}
	g.rebuild()
}

// SetTags cập nhật danh sách Thẻ trong nhóm và tự động xây dựng lại các ô chọn có màu tương ứng.
func (g *ColoredTagCheckGroup) SetTags(tags []Tag) {
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = t.Name
	}
	g.SetOptions(names, tags)
}

// SetSelected chọn các thẻ theo danh sách tên truyền vào.
func (g *ColoredTagCheckGroup) SetSelected(selected []string) {
	g.Selected = append([]string{}, selected...)
	selectedMap := make(map[string]bool, len(selected))
	for _, s := range selected {
		selectedMap[s] = true
	}
	for _, item := range g.items {
		item.SetChecked(selectedMap[item.Tag.Name])
	}
	if g.container != nil {
		g.container.Refresh()
	}
}

func (g *ColoredTagCheckGroup) rebuild() {
	g.items = nil
	g.container.Objects = nil

	if len(g.Options) == 0 {
		emptyLbl := widget.NewLabelWithStyle("(Chưa có thẻ nào cho danh mục này. Nhấn nút 'Thêm Thẻ...' ở trên để tạo thẻ mới)", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
		g.container.Add(emptyLbl)
		g.container.Refresh()
		return
	}

	selectedMap := make(map[string]bool, len(g.Selected))
	for _, s := range g.Selected {
		selectedMap[s] = true
	}

	for _, name := range g.Options {
		tagName := name
		tag, exists := g.tagMap[tagName]
		if !exists {
			tag = Tag{Name: tagName, Color: DefaultTagColor}
		}
		isChecked := selectedMap[tagName]
		chk := NewColoredTagCheckbox(tag, isChecked, func(checked bool) {
			g.updateSelection(tagName, checked)
		})
		g.items = append(g.items, chk)
		g.container.Add(chk)
	}

	g.container.Refresh()
}

func (g *ColoredTagCheckGroup) updateSelection(name string, checked bool) {
	if checked {
		found := false
		for _, s := range g.Selected {
			if s == name {
				found = true
				break
			}
		}
		if !found {
			g.Selected = append(g.Selected, name)
		}
	} else {
		var newSel []string
		for _, s := range g.Selected {
			if s != name {
				newSel = append(newSel, s)
			}
		}
		g.Selected = newSel
	}

	if g.OnChanged != nil {
		g.OnChanged(g.Selected)
	}
}

const (
	allTagsFilterLabel       = "Tất cả thẻ"
	allCategoriesFilterLabel = "Tất cả danh mục thẻ"
)

// WorldBuildingHub quản lý cửa sổ đa tab cho Nhân vật, Địa điểm, Vật phẩm, Sự kiện và Hệ thống Thẻ phân tách theo Danh mục.
type WorldBuildingHub struct {
	store     *Store
	parentWin fyne.Window
	bookID    int64
	onUpdated func()

	// Danh sách Thẻ được cô lập riêng biệt theo từng loại thực thể (EntityType)
	charTags      []Tag
	locTags       []Tag
	propTags      []Tag
	eventTags     []Tag
	tags          []Tag
	filteredTags  []Tag
	charTagNames  []string
	locTagNames   []string
	propTagNames  []string
	eventTagNames []string

	// Dữ liệu đã lọc cho từng Tab
	filteredChars  []Character
	filteredLocs   []Location
	filteredProps  []Prop
	filteredEvents []Event

	// Trạng thái bộ lọc theo thẻ của từng Tab
	charTagFilter            string
	locTagFilter             string
	propTagFilter            string
	eventTagFilter           string
	tagManagerCategoryFilter string

	// Các widget bộ lọc theo thẻ
	charFilterSelect       *widget.Select
	locFilterSelect        *widget.Select
	propFilterSelect       *widget.Select
	eventFilterSelect      *widget.Select
	tagManagerFilterSelect *widget.Select

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
	charTagCheck  *ColoredTagCheckGroup

	// Biểu mẫu Tab 2: Địa điểm
	locNameEntry *VietnameseEntry
	locDescEntry *VietnameseEntry
	locTagCheck  *ColoredTagCheckGroup

	// Biểu mẫu Tab 3: Vật phẩm
	propNameEntry *VietnameseEntry
	propCatEntry  *VietnameseEntry
	propDescEntry *VietnameseEntry
	propSigEntry  *VietnameseEntry
	propTagCheck  *ColoredTagCheckGroup

	// Biểu mẫu Tab 4: Sự kiện
	eventTitleEntry *VietnameseEntry
	eventOrderEntry *widget.Entry
	eventDescEntry  *VietnameseEntry
	eventTagCheck   *ColoredTagCheckGroup

	// Container giao diện bên phải và thanh cuộn của từng Tab thực thể (hỗ trợ Refresh tức thời)
	charRightForm    *fyne.Container
	charRightScroll  *container.Scroll
	locRightForm     *fyne.Container
	locRightScroll   *container.Scroll
	propRightForm    *fyne.Container
	propRightScroll  *container.Scroll
	eventRightForm   *fyne.Container
	eventRightScroll *container.Scroll

	// Biểu mẫu Quản lý Thẻ theo danh mục & Lưới ô màu trực quan (Visual Swatch Picker)
	newTagEntry           *VietnameseEntry
	quickCategorySelect   *widget.Select
	quickColorPicker      *TagColorSwatchPicker
	tagFormCategorySelect *widget.Select
	tagFormNameEntry      *VietnameseEntry
	tagFormHexEntry       *VietnameseEntry
	tagFormColorPicker    *TagColorSwatchPicker
	tagFormPreviewDot     *canvas.Circle
	tagFormPreviewBg      *canvas.Rectangle
	tagFormPreviewLabel   *canvas.Text
}

// ShowWorldBuildingHub mở Trung tâm Quản lý Thế giới & Hệ thống Thẻ Đa năng (Multi-Tab Dialog).
func ShowWorldBuildingHub(store *Store, parentWin fyne.Window, bookID int64, onUpdated func()) {
	hub := &WorldBuildingHub{
		store:                    store,
		parentWin:                parentWin,
		bookID:                   bookID,
		onUpdated:                onUpdated,
		charTagFilter:            allTagsFilterLabel,
		locTagFilter:             allTagsFilterLabel,
		propTagFilter:            allTagsFilterLabel,
		eventTagFilter:           allTagsFilterLabel,
		tagManagerCategoryFilter: allCategoriesFilterLabel,
	}

	content := hub.buildContent()
	hub.reloadAllData()

	d := dialog.NewCustom("Trung Tâm Xây Dựng Thế Giới & Quản Lý Thẻ Theo Danh Mục", "Đóng cửa sổ", content, parentWin)
	attachWindowRefreshOnClose(d, parentWin)
	d.Resize(fyne.NewSize(1140, 740))
	d.SetOnClosed(func() {
		if hub.onUpdated != nil {
			hub.onUpdated()
		}
	})
	d.Show()
}

func (h *WorldBuildingHub) buildContent() fyne.CanvasObject {
	// Thanh tạo Thẻ nhanh ở trên cùng kèm chọn Danh mục thẻ & Màu sắc thẻ
	h.newTagEntry = NewVietEntry()
	h.newTagEntry.SetPlaceHolder("Nhập tên thẻ mới cho danh mục đang chọn...")

	categoryLabels := AllEntityTypeTagLabels()
	selectedQuickCategory := EntityCharacter
	h.quickCategorySelect = widget.NewSelect(categoryLabels, func(selected string) {
		selectedQuickCategory = ParseEntityTypeTagLabel(selected)
	})
	if len(categoryLabels) > 0 {
		h.quickCategorySelect.SetSelected(EntityTypeTagLabel(EntityCharacter))
	}

	selectedQuickHex := DefaultTagColor
	h.quickColorPicker = NewTagColorSwatchPicker(h.parentWin, selectedQuickHex, 26, func(hex string) {
		selectedQuickHex = hex
	})

	quickAddTagBtn := widget.NewButtonWithIcon("Thêm thẻ mới", theme.ContentAddIcon(), func() {
		name := strings.TrimSpace(h.newTagEntry.Text)
		if name == "" {
			return
		}
		if _, err := h.store.CreateTagForType(h.bookID, selectedQuickCategory, name, selectedQuickHex); err != nil {
			showWindowErrorDialog(err, h.parentWin)
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
		widget.NewLabel("Danh mục:"),
		h.quickCategorySelect,
		widget.NewLabel("Bảng màu nhanh:"),
		container.NewCenter(h.quickColorPicker.Container()),
		quickAddTagBtn,
	)

	globalTagHeader := container.NewBorder(
		nil,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("🏷️ Thêm thẻ nhanh:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		rightQuickControls,
		h.newTagEntry,
	)

	tabChars := container.NewTabItem("Nhân vật", h.buildCharactersTab())
	tabLocs := container.NewTabItem("Địa điểm", h.buildLocationsTab())
	tabProps := container.NewTabItem("Vật phẩm", h.buildPropsTab())
	tabEvents := container.NewTabItem("Sự kiện", h.buildEventsTab())
	tabTags := container.NewTabItem("Quản lý Thẻ", h.buildTagManagerTab())

	tabs := container.NewAppTabs(tabChars, tabLocs, tabProps, tabEvents, tabTags)
	// Tự động đồng bộ Danh mục thẻ trên thanh tạo nhanh và làm mới bộ chọn thẻ khi chuyển Tab
	tabs.OnSelected = func(item *container.TabItem) {
		switch item {
		case tabChars:
			h.quickCategorySelect.SetSelected(EntityTypeTagLabel(EntityCharacter))
			if h.charTagCheck != nil {
				h.charTagCheck.Refresh()
			}
			if h.charRightForm != nil {
				h.charRightForm.Refresh()
			}
			if h.charRightScroll != nil {
				h.charRightScroll.Refresh()
			}
		case tabLocs:
			h.quickCategorySelect.SetSelected(EntityTypeTagLabel(EntityLocation))
			if h.locTagCheck != nil {
				h.locTagCheck.Refresh()
			}
			if h.locRightForm != nil {
				h.locRightForm.Refresh()
			}
			if h.locRightScroll != nil {
				h.locRightScroll.Refresh()
			}
		case tabProps:
			h.quickCategorySelect.SetSelected(EntityTypeTagLabel(EntityProp))
			if h.propTagCheck != nil {
				h.propTagCheck.Refresh()
			}
			if h.propRightForm != nil {
				h.propRightForm.Refresh()
			}
			if h.propRightScroll != nil {
				h.propRightScroll.Refresh()
			}
		case tabEvents:
			h.quickCategorySelect.SetSelected(EntityTypeTagLabel(EntityEvent))
			if h.eventTagCheck != nil {
				h.eventTagCheck.Refresh()
			}
			if h.eventRightForm != nil {
				h.eventRightForm.Refresh()
			}
			if h.eventRightScroll != nil {
				h.eventRightScroll.Refresh()
			}
		case tabTags:
			h.refreshFilteredTagsList()
		}
	}

	return container.NewBorder(globalTagHeader, nil, nil, nil, tabs)
}

// ==================== TAB 1: NHÂN VẬT (CHARACTERS - THẺ NHÂN VẬT RIÊNG BIỆT) ====================

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
			sub := widget.NewLabel("Vai trò • #Thẻ Nhân Vật")
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

	h.charTagCheck = NewColoredTagCheckGroup(nil)

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
		if h.charRightForm != nil {
			h.charRightForm.Refresh()
		}
	}

	addCharTagBtn := widget.NewButtonWithIcon("Thêm Thẻ...", theme.ContentAddIcon(), func() {
		h.showCreateScopedTagDialog(EntityCharacter, h.charTagCheck)
	})
	addCharTagBtn.Importance = widget.LowImportance

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
		tagIDs := h.resolveSelectedTagIDsForType(EntityCharacter, h.charTagCheck.Selected)

		if h.selectedCharID == 0 {
			created, err := h.store.CreateCharacter(h.bookID, name, role, h.charDescEntry.Text)
			if err != nil {
				showWindowErrorDialog(err, h.parentWin)
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
				showWindowErrorDialog(err, h.parentWin)
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
			widget.NewLabelWithStyle("Lọc theo Thẻ Nhân Vật:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			h.charFilterSelect,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		h.charList,
	)

	tagSectionHeader := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("GẮN THẺ NHÂN VẬT (CHỈ HIỂN THỊ THẺ NHÂN VẬT)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		addCharTagBtn,
	)

	h.charRightForm = container.NewVBox(
		widget.NewLabelWithStyle("THÔNG TIN NHÂN VẬT", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Tên nhân vật", h.charNameEntry),
			widget.NewFormItem("Vai trò", h.charRoleEntry),
			widget.NewFormItem("Mô tả & Tiểu sử", h.charDescEntry),
		),
		widget.NewSeparator(),
		tagSectionHeader,
		h.charTagCheck.Container(),
		widget.NewSeparator(),
		container.NewHBox(newBtn, layout.NewSpacer(), delBtn, saveBtn),
	)
	h.charRightScroll = container.NewVScroll(h.charRightForm)

	split := container.NewHSplit(leftPane, h.charRightScroll)
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
	if h.charRightForm != nil {
		h.charRightForm.Refresh()
	}
}

// ==================== TAB 2: ĐỊA ĐIỂM (LOCATIONS - THẺ ĐỊA ĐIỂM RIÊNG BIỆT) ====================

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
			sub := widget.NewLabel("#Thẻ Địa Điểm")
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

	h.locTagCheck = NewColoredTagCheckGroup(nil)

	h.locList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(h.filteredLocs) {
			return
		}
		l := h.filteredLocs[id]
		h.selectedLocID = l.ID
		h.locNameEntry.SetText(l.Name)
		h.locDescEntry.SetText(l.Description)
		h.locTagCheck.SetSelected(extractTagNames(l.Tags))
		if h.locRightForm != nil {
			h.locRightForm.Refresh()
		}
	}

	addLocTagBtn := widget.NewButtonWithIcon("Thêm Thẻ...", theme.ContentAddIcon(), func() {
		h.showCreateScopedTagDialog(EntityLocation, h.locTagCheck)
	})
	addLocTagBtn.Importance = widget.LowImportance

	newBtn := widget.NewButtonWithIcon("Làm mới biểu mẫu", theme.DocumentCreateIcon(), func() {
		h.clearLocationForm()
	})

	saveBtn := widget.NewButtonWithIcon("Lưu Địa Điểm", theme.DocumentSaveIcon(), func() {
		name := strings.TrimSpace(h.locNameEntry.Text)
		if name == "" {
			return
		}
		tagIDs := h.resolveSelectedTagIDsForType(EntityLocation, h.locTagCheck.Selected)

		if h.selectedLocID == 0 {
			created, err := h.store.CreateLocation(h.bookID, name, h.locDescEntry.Text)
			if err != nil {
				showWindowErrorDialog(err, h.parentWin)
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
				showWindowErrorDialog(err, h.parentWin)
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
			widget.NewLabelWithStyle("Lọc theo Thẻ Địa Điểm:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			h.locFilterSelect,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		h.locList,
	)

	tagSectionHeader := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("GẮN THẺ ĐỊA ĐIỂM (CHỈ HIỂN THỊ THẺ ĐỊA ĐIỂM)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		addLocTagBtn,
	)

	h.locRightForm = container.NewVBox(
		widget.NewLabelWithStyle("THÔNG TIN ĐỊA ĐIỂM / BỐI CẢNH", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Tên địa điểm", h.locNameEntry),
			widget.NewFormItem("Mô tả chi tiết", h.locDescEntry),
		),
		widget.NewSeparator(),
		tagSectionHeader,
		h.locTagCheck.Container(),
		widget.NewSeparator(),
		container.NewHBox(newBtn, layout.NewSpacer(), delBtn, saveBtn),
	)
	h.locRightScroll = container.NewVScroll(h.locRightForm)

	split := container.NewHSplit(leftPane, h.locRightScroll)
	split.Offset = 0.38
	return split
}

func (h *WorldBuildingHub) clearLocationForm() {
	h.selectedLocID = 0
	h.locList.UnselectAll()
	h.locNameEntry.SetText("")
	h.locDescEntry.SetText("")
	h.locTagCheck.SetSelected([]string{})
	if h.locRightForm != nil {
		h.locRightForm.Refresh()
	}
}

// ==================== TAB 3: VẬT PHẨM (PROPS - THẺ VẬT PHẨM RIÊNG BIỆT) ====================

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
			sub := widget.NewLabel("Phân loại • #Thẻ Vật Phẩm")
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

	h.propTagCheck = NewColoredTagCheckGroup(nil)

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
		if h.propRightForm != nil {
			h.propRightForm.Refresh()
		}
	}

	addPropTagBtn := widget.NewButtonWithIcon("Thêm Thẻ...", theme.ContentAddIcon(), func() {
		h.showCreateScopedTagDialog(EntityProp, h.propTagCheck)
	})
	addPropTagBtn.Importance = widget.LowImportance

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
		tagIDs := h.resolveSelectedTagIDsForType(EntityProp, h.propTagCheck.Selected)

		if h.selectedPropID == 0 {
			created, err := h.store.CreateProp(h.bookID, name, cat, h.propDescEntry.Text, h.propSigEntry.Text)
			if err != nil {
				showWindowErrorDialog(err, h.parentWin)
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
				showWindowErrorDialog(err, h.parentWin)
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
			widget.NewLabelWithStyle("Lọc theo Thẻ Vật Phẩm:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			h.propFilterSelect,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		h.propList,
	)

	tagSectionHeader := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("GẮN THẺ VẬT PHẨM (CHỈ HIỂN THỊ THẺ VẬT PHẨM)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		addPropTagBtn,
	)

	h.propRightForm = container.NewVBox(
		widget.NewLabelWithStyle("THÔNG TIN VẬT PHẨM (PROPS)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Tên vật phẩm", h.propNameEntry),
			widget.NewFormItem("Phân loại", h.propCatEntry),
			widget.NewFormItem("Mô tả chi tiết", h.propDescEntry),
			widget.NewFormItem("Ý nghĩa cốt truyện", h.propSigEntry),
		),
		widget.NewSeparator(),
		tagSectionHeader,
		h.propTagCheck.Container(),
		widget.NewSeparator(),
		container.NewHBox(newBtn, layout.NewSpacer(), delBtn, saveBtn),
	)
	h.propRightScroll = container.NewVScroll(h.propRightForm)

	split := container.NewHSplit(leftPane, h.propRightScroll)
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
	if h.propRightForm != nil {
		h.propRightForm.Refresh()
	}
}

// ==================== TAB 4: SỰ KIỆN (EVENTS TIMELINE - THẺ SỰ KIỆN RIÊNG BIỆT) ====================

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
			sub := widget.NewLabel("#Thẻ Sự Kiện")
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

	h.eventTagCheck = NewColoredTagCheckGroup(nil)

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
		if h.eventRightForm != nil {
			h.eventRightForm.Refresh()
		}
	}

	addEventTagBtn := widget.NewButtonWithIcon("Thêm Thẻ...", theme.ContentAddIcon(), func() {
		h.showCreateScopedTagDialog(EntityEvent, h.eventTagCheck)
	})
	addEventTagBtn.Importance = widget.LowImportance

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
		tagIDs := h.resolveSelectedTagIDsForType(EntityEvent, h.eventTagCheck.Selected)

		if h.selectedEventID == 0 {
			created, err := h.store.CreateEvent(h.bookID, title, order, h.eventDescEntry.Text)
			if err != nil {
				showWindowErrorDialog(err, h.parentWin)
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
				showWindowErrorDialog(err, h.parentWin)
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
			widget.NewLabelWithStyle("Lọc theo Thẻ Sự Kiện:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			h.eventFilterSelect,
			widget.NewSeparator(),
		),
		nil, nil, nil,
		h.eventList,
	)

	tagSectionHeader := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("GẮN THẺ SỰ KIỆN (CHỈ HIỂN THỊ THẺ SỰ KIỆN)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		addEventTagBtn,
	)

	h.eventRightForm = container.NewVBox(
		widget.NewLabelWithStyle("THÔNG TIN SỰ KIỆN DÒNG THỜI GIAN", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Tiêu đề sự kiện", h.eventTitleEntry),
			widget.NewFormItem("Thứ tự thời gian", h.eventOrderEntry),
			widget.NewFormItem("Mô tả diễn biến", h.eventDescEntry),
		),
		widget.NewSeparator(),
		tagSectionHeader,
		h.eventTagCheck.Container(),
		widget.NewSeparator(),
		container.NewHBox(newBtn, layout.NewSpacer(), delBtn, saveBtn),
	)
	h.eventRightScroll = container.NewVScroll(h.eventRightForm)

	split := container.NewHSplit(leftPane, h.eventRightScroll)
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
	if h.eventRightForm != nil {
		h.eventRightForm.Refresh()
	}
}

// ==================== TAB 5: QUẢN LÝ THẺ THEO DANH MỤC & MÀU SẮC THẺ ====================

func (h *WorldBuildingHub) buildTagManagerTab() fyne.CanvasObject {
	categoryLabels := AllEntityTypeTagLabels()
	filterCategoryOpts := append([]string{allCategoriesFilterLabel}, categoryLabels...)

	h.tagManagerFilterSelect = widget.NewSelect(filterCategoryOpts, func(selected string) {
		if selected == "" {
			selected = allCategoriesFilterLabel
		}
		h.tagManagerCategoryFilter = selected
		h.refreshFilteredTagsList()
	})
	h.tagManagerFilterSelect.SetSelected(allCategoriesFilterLabel)

	h.tagFormCategorySelect = widget.NewSelect(categoryLabels, nil)
	if len(categoryLabels) > 0 {
		h.tagFormCategorySelect.SetSelected(EntityTypeTagLabel(EntityCharacter))
	}

	h.tagFormNameEntry = NewVietEntry()
	h.tagFormNameEntry.SetPlaceHolder("Nhập tên thẻ (VD: Hội Hoa Tiêu, Khu vực cấm, Cổ vật)...")

	h.tagFormHexEntry = NewVietEntry()
	h.tagFormHexEntry.SetPlaceHolder("#3498db")
	h.tagFormHexEntry.SetText(DefaultTagColor)

	// Huy hiệu xem trước màu sắc thẻ trực tiếp (sử dụng canvas.Rectangle & canvas.Circle & canvas.Text)
	var previewDotWrap *fyne.Container
	h.tagFormPreviewDot, previewDotWrap = newColorCircleIndicator(DefaultTagColor, 16)
	h.tagFormPreviewBg = canvas.NewRectangle(parseHexTintColor(DefaultTagColor, 42))
	h.tagFormPreviewBg.StrokeColor = parseHexColor(DefaultTagColor)
	h.tagFormPreviewBg.StrokeWidth = 1.5
	h.tagFormPreviewBg.CornerRadius = 6
	h.tagFormPreviewLabel = canvas.NewText("🏷️ [Thẻ Nhân Vật] #Xem trước thẻ  (#3498db)", parseHexColor(DefaultTagColor))
	h.tagFormPreviewLabel.TextStyle = fyne.TextStyle{Bold: true}
	h.tagFormPreviewLabel.TextSize = 13

	previewBadge := container.NewMax(
		h.tagFormPreviewBg,
		container.NewPadded(container.NewHBox(
			container.NewCenter(previewDotWrap),
			h.tagFormPreviewLabel,
		)),
	)

	syncingHex := false
	updateLiveTagPreview := func() {
		hex := NormalizeHexColor(h.tagFormHexEntry.Text)
		tagName := strings.TrimSpace(h.tagFormNameEntry.Text)
		if tagName == "" {
			tagName = "Xem trước thẻ"
		}
		catLabel := h.tagFormCategorySelect.Selected
		if catLabel == "" {
			catLabel = EntityTypeTagLabel(EntityCharacter)
		}
		h.tagFormPreviewDot.FillColor = parseHexColor(hex)
		h.tagFormPreviewDot.Refresh()
		h.tagFormPreviewBg.FillColor = parseHexTintColor(hex, 42)
		h.tagFormPreviewBg.StrokeColor = parseHexColor(hex)
		h.tagFormPreviewBg.Refresh()
		h.tagFormPreviewLabel.Text = fmt.Sprintf("🏷️ [%s] #%s  (%s)", catLabel, tagName, hex)
		h.tagFormPreviewLabel.Color = parseHexColor(hex)
		h.tagFormPreviewLabel.Refresh()
	}

	// Lưới ô màu trực quan (Visual Color Swatch Grid) kèm ô vuông "+" mở bảng chọn màu tùy chỉnh
	h.tagFormColorPicker = NewTagColorSwatchPicker(h.parentWin, DefaultTagColor, 34, func(selectedHex string) {
		syncingHex = true
		h.tagFormHexEntry.SetText(selectedHex)
		syncingHex = false
		updateLiveTagPreview()
	})

	h.tagFormCategorySelect.OnChanged = func(_ string) {
		updateLiveTagPreview()
	}
	h.tagFormNameEntry.SetOnChangedCallback(func(_ string) {
		updateLiveTagPreview()
	})
	h.tagFormHexEntry.SetOnChangedCallback(func(val string) {
		if !syncingHex && h.tagFormColorPicker != nil {
			h.tagFormColorPicker.SelectHex(val, false)
		}
		updateLiveTagPreview()
	})

	// Danh sách Thẻ bên trái với huy hiệu màu sắc & nhãn Danh mục thẻ
	h.tagList = widget.NewList(
		func() int { return len(h.filteredTags) },
		func() fyne.CanvasObject {
			bgRect := canvas.NewRectangle(parseHexTintColor(DefaultTagColor, 32))
			bgRect.CornerRadius = 6
			bgRect.StrokeWidth = 1
			bgRect.StrokeColor = parseHexColor(DefaultTagColor)

			_, dotWrap := newColorCircleIndicator(DefaultTagColor, 16)
			catLbl := widget.NewLabelWithStyle("[Thẻ Nhân Vật]", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})
			nameText := canvas.NewText("#Tên thẻ", parseHexColor(DefaultTagColor))
			nameText.TextStyle = fyne.TextStyle{Bold: true}
			nameText.TextSize = 13
			hexLbl := widget.NewLabelWithStyle("#3498db", fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})

			editBtn := widget.NewButtonWithIcon("Chỉnh sửa thẻ", theme.DocumentCreateIcon(), nil)
			editBtn.Importance = widget.LowImportance

			delBtn := widget.NewButtonWithIcon("Xóa thẻ", theme.DeleteIcon(), nil)
			delBtn.Importance = widget.DangerImportance

			leftInfo := container.NewHBox(container.NewCenter(dotWrap), catLbl, nameText, hexLbl)
			rightActions := container.NewHBox(editBtn, delBtn)
			rowContent := container.NewBorder(nil, nil, nil, rightActions, leftInfo)

			return container.NewMax(bgRect, container.NewPadded(rowContent))
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if i < 0 || i >= len(h.filteredTags) {
				return
			}
			tag := h.filteredTags[i]
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
				if len(hbox.Objects) == 4 {
					// leftInfo: [center(dotWrap), catLbl, nameText, hexLbl]
					if centerWrap, ok := hbox.Objects[0].(*fyne.Container); ok && len(centerWrap.Objects) > 0 {
						if gridWrap, ok := centerWrap.Objects[0].(*fyne.Container); ok && len(gridWrap.Objects) > 0 {
							if circle, ok := gridWrap.Objects[0].(*canvas.Circle); ok {
								circle.FillColor = parseHexColor(tagColor)
								circle.Refresh()
							}
						}
					}
					if catLbl, ok := hbox.Objects[1].(*widget.Label); ok {
						catLbl.SetText(fmt.Sprintf("[%s]", EntityTypeTagLabel(tag.EntityType)))
					}
					if nameText, ok := hbox.Objects[2].(*canvas.Text); ok {
						nameText.Text = fmt.Sprintf("#%s", tag.Name)
						nameText.Color = parseHexColor(tagColor)
						nameText.Refresh()
					}
					if hexLbl, ok := hbox.Objects[3].(*widget.Label); ok {
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
		if id < 0 || id >= len(h.filteredTags) {
			return
		}
		t := h.filteredTags[id]
		h.selectedTagID = t.ID
		h.tagFormCategorySelect.SetSelected(EntityTypeTagLabel(t.EntityType))
		h.tagFormNameEntry.SetText(t.Name)
		normHex := NormalizeHexColor(t.Color)
		syncingHex = true
		h.tagFormHexEntry.SetText(normHex)
		syncingHex = false
		if h.tagFormColorPicker != nil {
			h.tagFormColorPicker.SelectHex(normHex, false)
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
		cat := ParseEntityTypeTagLabel(h.tagFormCategorySelect.Selected)
		if h.selectedTagID == 0 {
			created, err := h.store.CreateTagForType(h.bookID, cat, name, hexColor)
			if err != nil {
				showWindowErrorDialog(err, h.parentWin)
				return
			}
			h.selectedTagID = created.ID
		} else {
			if err := h.store.UpdateTag(h.selectedTagID, name, hexColor, cat); err != nil {
				showWindowErrorDialog(err, h.parentWin)
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

	info := widget.NewLabel("Mỗi danh mục thực thể (Thẻ Nhân Vật, Thẻ Địa Điểm, Thẻ Vật Phẩm, Thẻ Sự Kiện) có bộ thẻ tách biệt hoàn toàn để tránh lộn xộn.")
	info.Wrapping = fyne.TextWrapWord

	leftPane := container.NewBorder(
		container.NewVBox(
			info,
			container.NewBorder(
				nil, nil,
				widget.NewLabelWithStyle("Lọc theo danh mục thẻ:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				nil,
				h.tagManagerFilterSelect,
			),
			widget.NewSeparator(),
		),
		nil, nil, nil,
		h.tagList,
	)

	rightForm := container.NewVScroll(container.NewVBox(
		widget.NewLabelWithStyle("THÊM THẺ MỚI / CHỈNH SỬA THẺ THEO DANH MỤC", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Danh mục thẻ", h.tagFormCategorySelect),
			widget.NewFormItem("Tên thẻ", h.tagFormNameEntry),
			widget.NewFormItem("Bảng màu nhanh", h.tagFormColorPicker.Container()),
			widget.NewFormItem("Mã màu Hex", h.tagFormHexEntry),
			widget.NewFormItem("Xem trước huy hiệu", previewBadge),
		),
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
	if h.tagFormColorPicker != nil {
		h.tagFormColorPicker.SelectHex(DefaultTagColor, false)
	}
}

// showCreateScopedTagDialog mở hộp thoại tạo nhanh một Thẻ có màu sắc cho đúng Danh mục thực thể (EntityType) đang mở.
func (h *WorldBuildingHub) showCreateScopedTagDialog(entityType EntityType, targetCheckGroup *ColoredTagCheckGroup) {
	catLabel := EntityTypeTagLabel(entityType)

	nameEntry := NewVietEntry()
	nameEntry.SetPlaceHolder(fmt.Sprintf("Nhập tên %s mới...", catLabel))

	hexEntry := NewVietEntry()
	hexEntry.SetText(DefaultTagColor)

	dotCircle, dotWrap := newColorCircleIndicator(DefaultTagColor, 20)

	syncing := false
	var swatchPicker *TagColorSwatchPicker
	swatchPicker = NewTagColorSwatchPicker(h.parentWin, DefaultTagColor, 32, func(selectedHex string) {
		syncing = true
		hexEntry.SetText(selectedHex)
		syncing = false
		dotCircle.FillColor = parseHexColor(selectedHex)
		dotCircle.Refresh()
	})

	hexEntry.SetOnChangedCallback(func(val string) {
		if !syncing && swatchPicker != nil {
			swatchPicker.SelectHex(val, false)
		}
		dotCircle.FillColor = parseHexColor(val)
		dotCircle.Refresh()
	})

	hexRow := container.NewBorder(nil, nil, container.NewCenter(dotWrap), nil, hexEntry)

	showWindowFormDialog("Thêm "+catLabel+" mới", "Tạo thẻ", "Hủy", []*widget.FormItem{
		widget.NewFormItem("Danh mục", widget.NewLabelWithStyle(catLabel, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})),
		widget.NewFormItem("Tên thẻ", nameEntry),
		widget.NewFormItem("Bảng màu nhanh", swatchPicker.Container()),
		widget.NewFormItem("Màu sắc thẻ (Hex)", hexRow),
	}, func(confirmed bool) {
		if !confirmed || strings.TrimSpace(nameEntry.Text) == "" {
			return
		}
		created, err := h.store.CreateTagForType(h.bookID, entityType, nameEntry.Text, hexEntry.Text)
		if err != nil {
			showWindowErrorDialog(err, h.parentWin)
			return
		}
		prevSelected := append([]string{}, targetCheckGroup.Selected...)
		h.reloadAllData()
		if created != nil {
			already := false
			for _, s := range prevSelected {
				if s == created.Name {
					already = true
					break
				}
			}
			if !already {
				prevSelected = append(prevSelected, created.Name)
			}
			targetCheckGroup.SetSelected(prevSelected)
		}
		if h.onUpdated != nil {
			h.onUpdated()
		}
	}, h.parentWin)
}

// showEditTagDialog hiển thị hộp thoại chỉnh sửa nhanh danh mục, tên và màu sắc của một Thẻ.
func (h *WorldBuildingHub) showEditTagDialog(tag Tag) {
	categorySelect := widget.NewSelect(AllEntityTypeTagLabels(), nil)
	categorySelect.SetSelected(EntityTypeTagLabel(tag.EntityType))

	nameEntry := NewVietEntry()
	nameEntry.SetText(tag.Name)

	initialHex := NormalizeHexColor(tag.Color)
	hexEntry := NewVietEntry()
	hexEntry.SetText(initialHex)

	dotCircle, dotWrap := newColorCircleIndicator(initialHex, 20)

	syncing := false
	var swatchPicker *TagColorSwatchPicker
	swatchPicker = NewTagColorSwatchPicker(h.parentWin, initialHex, 32, func(selectedHex string) {
		syncing = true
		hexEntry.SetText(selectedHex)
		syncing = false
		dotCircle.FillColor = parseHexColor(selectedHex)
		dotCircle.Refresh()
	})

	hexEntry.SetOnChangedCallback(func(val string) {
		if !syncing && swatchPicker != nil {
			swatchPicker.SelectHex(val, false)
		}
		dotCircle.FillColor = parseHexColor(val)
		dotCircle.Refresh()
	})

	hexRow := container.NewBorder(nil, nil, container.NewCenter(dotWrap), nil, hexEntry)

	showWindowFormDialog("Chỉnh sửa thẻ", "Lưu thay đổi", "Hủy", []*widget.FormItem{
		widget.NewFormItem("Danh mục thẻ", categorySelect),
		widget.NewFormItem("Tên thẻ", nameEntry),
		widget.NewFormItem("Bảng màu nhanh", swatchPicker.Container()),
		widget.NewFormItem("Màu sắc thẻ (Hex)", hexRow),
	}, func(confirmed bool) {
		if !confirmed || strings.TrimSpace(nameEntry.Text) == "" {
			return
		}
		newCat := ParseEntityTypeTagLabel(categorySelect.Selected)
		if err := h.store.UpdateTag(tag.ID, nameEntry.Text, hexEntry.Text, newCat); err != nil {
			showWindowErrorDialog(err, h.parentWin)
			return
		}
		h.reloadAllData()
		if h.onUpdated != nil {
			h.onUpdated()
		}
	}, h.parentWin)
}

// ==================== NẠP DỮ LIỆU & ÁP DỤNG BỘ LỌC THẺ THEO DANH MỤC ====================

func buildTagOptionsAndFilter(tags []Tag) ([]string, []string) {
	names := make([]string, 0, len(tags))
	filterOpts := make([]string, 0, len(tags)+1)
	filterOpts = append(filterOpts, allTagsFilterLabel)
	for _, t := range tags {
		names = append(names, t.Name)
		filterOpts = append(filterOpts, t.Name)
	}
	return names, filterOpts
}

func updateSelectOptionsSafely(sel *widget.Select, opts []string, currentFilter *string) {
	if sel == nil {
		return
	}
	sel.Options = opts
	valid := false
	for _, opt := range opts {
		if opt == *currentFilter {
			valid = true
			break
		}
	}
	if !valid {
		*currentFilter = allTagsFilterLabel
	}
	if sel.Selected != *currentFilter {
		sel.SetSelected(*currentFilter)
	} else {
		sel.Refresh()
	}
}

func (h *WorldBuildingHub) reloadAllData() {
	// 1. Nạp danh sách Thẻ được cô lập riêng cho từng Danh mục thực thể (GetTagsByType) từ SQLite
	h.charTags, _ = h.store.GetTagsByType(h.bookID, EntityCharacter)
	h.locTags, _ = h.store.GetTagsByType(h.bookID, EntityLocation)
	h.propTags, _ = h.store.GetTagsByType(h.bookID, EntityProp)
	h.eventTags, _ = h.store.GetTagsByType(h.bookID, EntityEvent)
	h.tags, _ = h.store.ListTags(h.bookID)

	// Lấy danh sách tên thẻ phân tách theo từng danh mục
	charTagNames, charFilterOpts := buildTagOptionsAndFilter(h.charTags)
	locTagNames, locFilterOpts := buildTagOptionsAndFilter(h.locTags)
	propTagNames, propFilterOpts := buildTagOptionsAndFilter(h.propTags)
	eventTagNames, eventFilterOpts := buildTagOptionsAndFilter(h.eventTags)

	// Lưu trữ trên struct để sẵn sàng truy xuất
	h.charTagNames = charTagNames
	h.locTagNames = locTagNames
	h.propTagNames = propTagNames
	h.eventTagNames = eventTagNames

	// 2. Gán trực tiếp các biến danh sách tên thẻ theo danh mục vào các nhóm checkbox UI tương ứng
	if h.charTagCheck != nil {
		h.charTagCheck.SetOptions(charTagNames, h.charTags)
	}
	if h.locTagCheck != nil {
		h.locTagCheck.SetOptions(locTagNames, h.locTags)
	}
	if h.propTagCheck != nil {
		h.propTagCheck.SetOptions(propTagNames, h.propTags)
	}
	if h.eventTagCheck != nil {
		h.eventTagCheck.SetOptions(eventTagNames, h.eventTags)
	}

	// Cập nhật bộ lọc theo Thẻ riêng biệt của từng Tab
	updateSelectOptionsSafely(h.charFilterSelect, charFilterOpts, &h.charTagFilter)
	updateSelectOptionsSafely(h.locFilterSelect, locFilterOpts, &h.locTagFilter)
	updateSelectOptionsSafely(h.propFilterSelect, propFilterOpts, &h.propTagFilter)
	updateSelectOptionsSafely(h.eventFilterSelect, eventFilterOpts, &h.eventTagFilter)

	h.refreshCharactersList()
	h.refreshLocationsList()
	h.refreshPropsList()
	h.refreshEventsList()
	h.refreshFilteredTagsList()

	// 3. Làm mới toàn bộ container chứa các bộ chọn thẻ để các thẻ lập tức xuất hiện trên giao diện
	h.refreshTagContainers()
}

// refreshTagContainers làm mới toàn bộ widget và container chứa các bộ chọn Thẻ trên tất cả các Tab thực thể.
func (h *WorldBuildingHub) refreshTagContainers() {
	if h.charTagCheck != nil {
		h.charTagCheck.Refresh()
	}
	if h.charRightForm != nil {
		h.charRightForm.Refresh()
	}
	if h.charRightScroll != nil {
		h.charRightScroll.Refresh()
	}

	if h.locTagCheck != nil {
		h.locTagCheck.Refresh()
	}
	if h.locRightForm != nil {
		h.locRightForm.Refresh()
	}
	if h.locRightScroll != nil {
		h.locRightScroll.Refresh()
	}

	if h.propTagCheck != nil {
		h.propTagCheck.Refresh()
	}
	if h.propRightForm != nil {
		h.propRightForm.Refresh()
	}
	if h.propRightScroll != nil {
		h.propRightScroll.Refresh()
	}

	if h.eventTagCheck != nil {
		h.eventTagCheck.Refresh()
	}
	if h.eventRightForm != nil {
		h.eventRightForm.Refresh()
	}
	if h.eventRightScroll != nil {
		h.eventRightScroll.Refresh()
	}
}

func (h *WorldBuildingHub) refreshFilteredTagsList() {
	var out []Tag
	if h.tagManagerCategoryFilter == "" || h.tagManagerCategoryFilter == allCategoriesFilterLabel {
		out = append(out, h.tags...)
	} else {
		targetCat := ParseEntityTypeTagLabel(h.tagManagerCategoryFilter)
		for _, t := range h.tags {
			if t.EntityType == targetCat {
				out = append(out, t)
			}
		}
	}
	h.filteredTags = out
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

// resolveSelectedTagIDsForType ánh xạ tên các thẻ được chọn sang ID thẻ thuộc đúng danh mục thực thể (EntityType).
func (h *WorldBuildingHub) resolveSelectedTagIDsForType(entityType EntityType, selectedNames []string) []int64 {
	var pool []Tag
	switch NormalizeEntityType(string(entityType)) {
	case EntityCharacter:
		pool = h.charTags
	case EntityLocation:
		pool = h.locTags
	case EntityProp:
		pool = h.propTags
	case EntityEvent:
		pool = h.eventTags
	default:
		pool = h.charTags
	}

	var ids []int64
	for _, name := range selectedNames {
		for _, t := range pool {
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
