package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// SceneStatus biểu thị trạng thái biên tập của một Cảnh (Scene).
type SceneStatus string

const (
	StatusIdea      SceneStatus = "Ý tưởng"
	StatusDrafting  SceneStatus = "Đang viết"
	StatusCompleted SceneStatus = "Hoàn thành"
	StatusEdited    SceneStatus = "Đã biên tập"
)

// EntityType định danh loại thực thể cho hệ thống Thẻ phân tách theo danh mục (character / location / prop / event).
type EntityType string

const (
	EntityCharacter EntityType = "character"
	EntityLocation  EntityType = "location"
	EntityProp      EntityType = "prop"
	EntityEvent     EntityType = "event"
)

// AllEntityTypes trả về danh sách tất cả các phân nhóm thực thể hỗ trợ gắn thẻ riêng biệt.
func AllEntityTypes() []EntityType {
	return []EntityType{
		EntityCharacter,
		EntityLocation,
		EntityProp,
		EntityEvent,
	}
}

// EntityTypeTagLabel trả về tên hiển thị Tiếng Việt của danh mục Thẻ tương ứng.
func EntityTypeTagLabel(et EntityType) string {
	switch et {
	case EntityCharacter:
		return "Thẻ Nhân Vật"
	case EntityLocation:
		return "Thẻ Địa Điểm"
	case EntityProp:
		return "Thẻ Vật Phẩm"
	case EntityEvent:
		return "Thẻ Sự Kiện"
	default:
		return "Thẻ Nhân Vật"
	}
}

// AllEntityTypeTagLabels trả về danh sách nhãn Tiếng Việt của 4 danh mục Thẻ.
func AllEntityTypeTagLabels() []string {
	return []string{
		EntityTypeTagLabel(EntityCharacter),
		EntityTypeTagLabel(EntityLocation),
		EntityTypeTagLabel(EntityProp),
		EntityTypeTagLabel(EntityEvent),
	}
}

// ParseEntityTypeTagLabel chuyển đổi nhãn Tiếng Việt trên giao diện về mã EntityType.
func ParseEntityTypeTagLabel(label string) EntityType {
	switch strings.TrimSpace(label) {
	case "Thẻ Địa Điểm", string(EntityLocation):
		return EntityLocation
	case "Thẻ Vật Phẩm", string(EntityProp):
		return EntityProp
	case "Thẻ Sự Kiện", string(EntityEvent):
		return EntityEvent
	default:
		return EntityCharacter
	}
}

// NormalizeEntityType chuẩn hóa chuỗi entity_type từ cơ sở dữ liệu.
func NormalizeEntityType(raw string) EntityType {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(EntityLocation):
		return EntityLocation
	case string(EntityProp):
		return EntityProp
	case string(EntityEvent):
		return EntityEvent
	default:
		return EntityCharacter
	}
}

// AllSceneStatuses trả về danh sách các trạng thái chuẩn tiếng Việt cho UI.
func AllSceneStatuses() []string {
	return []string{
		string(StatusIdea),
		string(StatusDrafting),
		string(StatusCompleted),
		string(StatusEdited),
	}
}

// NormalizeStatus chuyển đổi các trạng thái tiếng Anh cũ (nếu có trong DB cũ) sang tiếng Việt.
func NormalizeStatus(raw SceneStatus) SceneStatus {
	switch strings.TrimSpace(string(raw)) {
	case "Idea", string(StatusIdea):
		return StatusIdea
	case "Drafting", string(StatusDrafting):
		return StatusDrafting
	case "Completed", string(StatusCompleted):
		return StatusCompleted
	case "Edited", string(StatusEdited):
		return StatusEdited
	default:
		return StatusIdea
	}
}

// Project đại diện cho một dự án tiểu thuyết / tác phẩm (Book).
type Project struct {
	ID          int64
	Title       string
	Author      string
	Genre       string
	Synopsis    string
	TargetWords int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Act đại diện cho một Hồi / Phần lớn trong cấu trúc tiểu thuyết.
type Act struct {
	ID        int64
	ProjectID int64
	Title     string
	Position  int
	CreatedAt time.Time
}

// Chapter đại diện cho một Chương thuộc về một Hồi.
type Chapter struct {
	ID          int64
	ActID       int64
	Title       string
	Position    int
	TargetWords int
	CreatedAt   time.Time
}

// DefaultTagColor là mã màu Hex mặc định cho Thẻ mới nếu người dùng chưa chọn màu.
const DefaultTagColor = "#3498db"

// Tag đại diện cho một Thẻ phân loại có màu sắc và được cô lập riêng theo từng danh mục thực thể (EntityType).
type Tag struct {
	ID         int64
	BookID     int64
	EntityType EntityType
	Name       string
	Color      string
}

// TagColorPreset định nghĩa một mẫu màu gợi ý kèm tên gọi Tiếng Việt cho giao diện chọn màu thẻ.
type TagColorPreset struct {
	Label string
	Hex   string
}

// DefaultTagColorPresets trả về bảng màu Hex gợi ý cho các nhóm Thẻ trong tiểu thuyết.
func DefaultTagColorPresets() []TagColorPreset {
	return []TagColorPreset{
		{Label: "Xanh lam (#3498db)", Hex: "#3498db"},
		{Label: "Đỏ chu sa (#e74c3c)", Hex: "#e74c3c"},
		{Label: "Xanh ngọc bích (#2ecc71)", Hex: "#2ecc71"},
		{Label: "Tím huyền bí (#9b59b6)", Hex: "#9b59b6"},
		{Label: "Vàng hổ phách (#f39c12)", Hex: "#f39c12"},
		{Label: "Xanh lục bảo (#1abc9c)", Hex: "#1abc9c"},
		{Label: "Cam hoàng hôn (#e67e22)", Hex: "#e67e22"},
		{Label: "Hồng san hô (#d81b60)", Hex: "#d81b60"},
		{Label: "Xám đá phiến (#34495e)", Hex: "#34495e"},
	}
}

// NormalizeHexColor chuẩn hóa chuỗi mã màu Hex (VD: "#3498db"), trả về DefaultTagColor nếu không hợp lệ.
func NormalizeHexColor(raw string) string {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return DefaultTagColor
	}
	if !strings.HasPrefix(clean, "#") {
		clean = "#" + clean
	}
	if len(clean) != 7 && len(clean) != 4 {
		return DefaultTagColor
	}
	for _, r := range clean[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return DefaultTagColor
		}
	}
	return strings.ToLower(clean)
}

// Character đại diện cho một Nhân vật trong dự án tiểu thuyết.
type Character struct {
	ID          int64
	ProjectID   int64
	Name        string
	Role        string
	Description string
	Tags        []Tag
}

// Location đại diện cho một Bối cảnh / Địa điểm trong dự án tiểu thuyết.
type Location struct {
	ID          int64
	ProjectID   int64
	Name        string
	Description string
	Tags        []Tag
}

// Prop đại diện cho một Vật phẩm / Đạo cụ quan trọng trong thế giới tiểu thuyết.
type Prop struct {
	ID           int64
	BookID       int64
	Name         string
	Category     string
	Description  string
	Significance string
	Tags         []Tag
}

// Event đại diện cho một Sự kiện lịch sử / cốt truyện theo dòng thời gian.
type Event struct {
	ID            int64
	BookID        int64
	Title         string
	TimelineOrder int
	Description   string
	Tags          []Tag
}

// Scene là đơn vị viết chính chứa nội dung văn xuôi, ghi chú bên lề và siêu dữ liệu ngữ cảnh.
type Scene struct {
	ID             int64
	ChapterID      int64
	Title          string
	Summary        string
	Content        string
	SideNotes      string
	Status         SceneStatus
	POVCharacterID *int64
	LocationID     *int64
	TargetWords    int
	WordCount      int
	Position       int
	UpdatedAt      time.Time
	CharacterIDs   []int64
	PropIDs        []int64
	EventIDs       []int64
}

// HierarchyNode biểu diễn một nút trên cây phân cấp (Hồi, Chương, hoặc Cảnh).
type HierarchyNode struct {
	UID         string
	Kind        string // "act", "chapter", "scene"
	DatabaseID  int64
	ParentID    int64
	Title       string
	Subtitle    string
	WordCount   int
	TargetWords int
	Status      SceneStatus
}

// MakeNodeUID tạo định danh duy nhất cho mỗi nút trên Fyne Tree.
func MakeNodeUID(kind string, id int64) string {
	return fmt.Sprintf("%s:%d", kind, id)
}

// ParseNodeUID tách định danh nút cây thành loại ("act", "chapter", "scene") và ID cơ sở dữ liệu.
func ParseNodeUID(uid string) (string, int64, error) {
	parts := strings.Split(uid, ":")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("UID nút không hợp lệ: %s", uid)
	}
	var id int64
	_, err := fmt.Sscanf(parts[1], "%d", &id)
	if err != nil {
		return "", 0, err
	}
	return parts[0], id, nil
}

// CountWords đếm số từ chuẩn xác cho văn bản tiếng Việt (UTF-8 đa byte) bằng gói unicode/utf8,
// tự động loại bỏ các thẻ định dạng Rich Text (**, *, <u>, </u>, >, ###) để không làm sai lệch thống kê từ.
func CountWords(text string) int {
	clean := strings.NewReplacer(
		"<u>", " ", "</u>", " ",
		"<b>", " ", "</b>", " ",
		"<strong>", " ", "</strong>", " ",
		"<i>", " ", "</i>", " ",
		"<em>", " ", "</em>", " ",
		"<ins>", " ", "</ins>", " ",
		"<blockquote>", " ", "</blockquote>", " ",
		"***", " ", "**", " ", "__", " ", "++", " ",
		"* * *", " ", "###", " ",
	).Replace(text)

	trimmed := strings.TrimSpace(clean)
	if trimmed == "" || utf8.RuneCountInString(trimmed) == 0 {
		return 0
	}
	count := 0
	inWord := false
	for len(trimmed) > 0 {
		r, width := utf8.DecodeRuneInString(trimmed)
		trimmed = trimmed[width:]
		if r == utf8.RuneError && width == 1 {
			continue
		}
		if unicode.IsSpace(r) || r == '*' || r == '_' || r == '>' {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}

// FormatTagNames ghép danh sách thẻ thành chuỗi hiển thị ngắn gọn (VD: "#Cổ vật, #Khu vực cấm").
func FormatTagNames(tags []Tag) string {
	if len(tags) == 0 {
		return "Chưa gắn thẻ"
	}
	names := make([]string, len(tags))
	for i, t := range tags {
		names[i] = "#" + t.Name
	}
	return strings.Join(names, "  ")
}

// HasTagName kiểm tra xem danh sách thẻ có chứa thẻ tên cụ thể hay không.
func HasTagName(tags []Tag, filterTag string) bool {
	if filterTag == "" || filterTag == "Tất cả thẻ" {
		return true
	}
	for _, t := range tags {
		if t.Name == filterTag {
			return true
		}
	}
	return false
}
