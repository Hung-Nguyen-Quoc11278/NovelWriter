package main

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// SceneStatus represents the editorial lifecycle stage of a Scene.
type SceneStatus string

const (
	StatusIdea      SceneStatus = "Idea"
	StatusDrafting  SceneStatus = "Drafting"
	StatusCompleted SceneStatus = "Completed"
	StatusEdited    SceneStatus = "Edited"
)

// AllStatuses returns all valid SceneStatus values in workflow order.
func AllStatuses() []string {
	return []string{
		string(StatusIdea),
		string(StatusDrafting),
		string(StatusCompleted),
		string(StatusEdited),
	}
}

// Project is the root container of a novel manuscript.
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

// Act represents a major structural division within a Project (e.g., Act I, Act II).
type Act struct {
	ID        int64
	ProjectID int64
	Title     string
	SortOrder int
	Chapters  []Chapter
}

// Chapter represents a chapter belonging to an Act.
type Chapter struct {
	ID          int64
	ActID       int64
	Title       string
	TargetWords int
	SortOrder   int
	Scenes      []Scene
}

// Character represents a named cast member in the Project.
type Character struct {
	ID        int64
	ProjectID int64
	Name      string
	Role      string
	Bio       string
}

// Location represents a setting or backdrop in the Project.
type Location struct {
	ID          int64
	ProjectID   int64
	Name        string
	Description string
}

// Scene is the atomic writing unit containing prose, side notes, and context mappings.
type Scene struct {
	ID             int64
	ChapterID      int64
	Title          string
	Content        string
	SideNotes      string
	Status         SceneStatus
	POVCharacterID *int64
	LocationID     *int64
	TargetWords    int
	WordCount      int
	SortOrder      int
	UpdatedAt      time.Time
	CharacterIDs   []int64
}

// NodeKind identifies the level of a node in the Fyne sidebar Tree.
type NodeKind string

const (
	NodeAct     NodeKind = "act"
	NodeChapter NodeKind = "chapter"
	NodeScene   NodeKind = "scene"
)

// TreeNode wraps a hierarchical item for Fyne's widget.Tree UID system.
type TreeNode struct {
	UID       string
	Kind      NodeKind
	ID        int64
	ParentID  int64
	Title     string
	Status    SceneStatus
	WordCount int
	Target    int
}

// MakeUID generates a deterministic Fyne TreeNodeUID string such as "act:1" or "scene:14".
func MakeUID(kind NodeKind, id int64) string {
	return fmt.Sprintf("%s:%d", kind, id)
}

// ParseUID extracts the NodeKind and database ID from a Fyne TreeNodeUID.
func ParseUID(uid string) (NodeKind, int64, error) {
	parts := strings.SplitN(uid, ":", 2)
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid tree UID: %s", uid)
	}
	var id int64
	_, err := fmt.Sscanf(parts[1], "%d", &id)
	if err != nil {
		return "", 0, err
	}
	return NodeKind(parts[0]), id, nil
}

// CountWords performs a fast, allocation-free Unicode word count on prose content.
func CountWords(text string) int {
	inWord := false
	count := 0
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '\'' || r == '’' {
			if !inWord {
				inWord = true
				count++
			}
		} else {
			inWord = false
		}
	}
	return count
}
