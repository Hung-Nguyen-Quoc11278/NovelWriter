package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// SceneStatus biểu thị trạng thái biên tập của một Cảnh (Scene).
type SceneStatus string

const (
	StatusIdea      SceneStatus = "Ý tưởng"
	StatusDrafting  SceneStatus = "Đang viết"
	StatusCompleted SceneStatus = "Hoàn thành"
	StatusEdited    SceneStatus = "Đã biên tập"
)

// EntityType định danh loại thực thể trong bảng ánh xạ thẻ đa năng (entity_tags).
type EntityType string

const (
	EntityCharacter EntityType = "character"
	EntityLocation  EntityType = "location"
	EntityProp      EntityType = "prop"
	EntityEvent     EntityType = "event"
)

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

// Tag đại diện cho một Thẻ phân loại đa năng trong tác phẩm (Ví dụ: "Thiên giới", "Khu vực cấm").
type Tag struct {
	ID     int64
	BookID int64
	Name   string
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

// CountWords đếm số từ chuẩn xác cho văn bản tiếng Việt (Unicode) và tiếng Anh.
func CountWords(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}
	count := 0
	inWord := false
	for _, r := range trimmed {
		if unicode.IsSpace(r) {
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
