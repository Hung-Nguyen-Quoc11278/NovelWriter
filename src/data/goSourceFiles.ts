export interface GoSourceFile {
  filename: string;
  path: string;
  layer: 'Entry Point' | 'Domain Layer' | 'Data Access & Service Layer' | 'UI Layer (Fyne v2)' | 'Database DDL' | 'Module Config';
  summary: string;
  code: string;
}

export const PROJECT_TREE_LAYOUT = `gonovelist/
├── go.mod          # Go 1.22+ module definition (fyne.io/fyne/v2, modernc.org/sqlite)
├── schema.sql      # Reference SQLite DDL schema with cascading foreign keys & WAL
├── main.go         # Application bootstrap, OS config path resolver, Fyne window lifecycle
├── models.go       # Domain entities (Project, Act, Chapter, Scene, Character, Location)
├── database.go     # SQLite Store repository, hierarchy queries, reordering, MD/HTML export
├── ui_main.go      # Main split layout, widget.Tree controller, dialogs, distraction-free mode
└── ui_editor.go    # RichText/Prose editor, 750ms debounced auto-save, metadata & side notes`;

export const GO_SOURCE_FILES: GoSourceFile[] = [
  {
    filename: 'main.go',
    path: 'gonovelist/main.go',
    layer: 'Entry Point',
    summary:
      'Initializes the Fyne desktop application, resolves the local SQLite database path in the OS user config directory, runs schema migrations, and wires the window close hook to flush any pending debounced saves.',
    code: `package main

import (
	"log"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
)

func main() {
	novelistApp := app.NewWithID("io.gonovelist.desktop")
	mainWindow := novelistApp.NewWindow("GoNovelist — Cross-Platform Novel Writing Studio")
	mainWindow.Resize(fyne.NewSize(1360, 860))
	mainWindow.CenterOnScreen()

	dbPath := resolveDatabasePath()
	store, err := NewStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize GoNovelist SQLite store at %s: %v", dbPath, err)
	}
	defer func() {
		_ = store.Close()
	}()

	ui, err := NewNovelistUI(novelistApp, mainWindow, store)
	if err != nil {
		log.Fatalf("Failed to build GoNovelist Fyne UI: %v", err)
	}

	mainWindow.SetOnClosed(func() {
		ui.editorPanel.FlushPendingSave()
	})

	mainWindow.ShowAndRun()
}

func resolveDatabasePath() string {
	if custom := os.Getenv("GONOVELIST_DB"); custom != "" {
		return custom
	}
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "gonovelist.db"
	}
	appDir := filepath.Join(cfgDir, "GoNovelist")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "gonovelist.db"
	}
	return filepath.Join(appDir, "gonovelist.db")
}`,
  },
  {
    filename: 'models.go',
    path: 'gonovelist/models.go',
    layer: 'Domain Layer',
    summary:
      'Defines core domain structs (Project, Act, Chapter, Scene, Character, Location), editorial status constants, Fyne TreeNode UID encoding/decoding helpers, and an allocation-free Unicode word counter.',
    code: `package main

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
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '\\'' || r == '’' {
			if !inWord {
				inWord = true
				count++
			}
		} else {
			inWord = false
		}
	}
	return count
}`,
  },
  {
    filename: 'database.go',
    path: 'gonovelist/database.go',
    layer: 'Data Access & Service Layer',
    summary:
      'Encapsulates SQLite persistence using database/sql and modernc.org/sqlite (or mattn/go-sqlite3), automatic DDL migration, transactional Scene updates with M:N character links, sibling sort_order reordering, and sequential Markdown/HTML compilation.',
    code: `package main

import (
	"database/sql"
	"fmt"
	"html"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store encapsulates all SQLite data access and business service operations.
type Store struct {
	db *sql.DB
}

// NewStore opens or creates the local SQLite database, enables WAL + foreign keys,
// executes schema migrations, and seeds initial sample data if the database is empty.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.Migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}
	if err := store.SeedIfNeeded(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("seed initial data: %w", err)
	}
	return store, nil
}

// Close cleanly closes the underlying SQLite connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Migrate creates all required tables and indices idempotently.
func (s *Store) Migrate() error {
	ddl := \`
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    author TEXT NOT NULL DEFAULT '',
    genre TEXT NOT NULL DEFAULT '',
    synopsis TEXT NOT NULL DEFAULT '',
    target_words INTEGER NOT NULL DEFAULT 80000,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS acts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS chapters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    act_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    target_words INTEGER NOT NULL DEFAULT 3000,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (act_id) REFERENCES acts(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS characters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'Supporting',
    bio TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS locations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS scenes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chapter_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    side_notes TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'Idea' CHECK (status IN ('Idea', 'Drafting', 'Completed', 'Edited')),
    pov_character_id INTEGER NULL,
    location_id INTEGER NULL,
    target_words INTEGER NOT NULL DEFAULT 1200,
    word_count INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chapter_id) REFERENCES chapters(id) ON DELETE CASCADE,
    FOREIGN KEY (pov_character_id) REFERENCES characters(id) ON DELETE SET NULL,
    FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS scene_characters (
    scene_id INTEGER NOT NULL,
    character_id INTEGER NOT NULL,
    PRIMARY KEY (scene_id, character_id),
    FOREIGN KEY (scene_id) REFERENCES scenes(id) ON DELETE CASCADE,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_acts_project_sort ON acts(project_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_chapters_act_sort ON chapters(act_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_scenes_chapter_sort ON scenes(chapter_id, sort_order);
\`
	_, err := s.db.Exec(ddl)
	return err
}

// SeedIfNeeded populates a complete starter manuscript if no projects exist yet.
func (s *Store) SeedIfNeeded() error {
	var count int
	if err := s.db.QueryRow(\`SELECT COUNT(*) FROM projects\`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	proj, err := s.CreateProject("The Glass Cartographer", "Clara Vance", "Literary Speculative Fiction", 75000)
	if err != nil {
		return err
	}

	elena, err := s.CreateCharacter(proj.ID, "Elena Rostova", "Protagonist", "Senior restorer of seventeenth-century celestial lenses at the Maritime Archive.")
	if err != nil {
		return err
	}
	julian, err := s.CreateCharacter(proj.ID, "Julian Vane", "Deuteragonist", "Hydrographer who charted the submerged tidal Causeway of St. Jude.")
	if err != nil {
		return err
	}
	maren, err := s.CreateCharacter(proj.ID, "Archivist Maren", "Antagonist", "Keeper of the Sealed Ledger and custodian of the Admiralty Vault.")
	if err != nil {
		return err
	}

	locArchive, err := s.CreateLocation(proj.ID, "The Lantern Vault, Old Admiralty", "Subterranean copper-domed archive lit by oil refraction prisms.")
	if err != nil {
		return err
	}
	locObservatory, err := s.CreateLocation(proj.ID, "Cliffside Meridian Tower", "Wind-scoured basalt tower overlooking the North Breakwater.")
	if err != nil {
		return err
	}

	act1, err := s.CreateAct(proj.ID, "Act I: The Refracted Meridian")
	if err != nil {
		return err
	}
	act2, err := s.CreateAct(proj.ID, "Act II: Soundings in Amber")
	if err != nil {
		return err
	}

	ch1, err := s.CreateChapter(act1.ID, "Chapter 1: Salt on the Objective Lens", 2500)
	if err != nil {
		return err
	}
	ch2, err := s.CreateChapter(act1.ID, "Chapter 2: The Admiralty Ledger", 2800)
	if err != nil {
		return err
	}
	ch3, err := s.CreateChapter(act2.ID, "Chapter 3: Low Tide at St. Jude", 3000)
	if err != nil {
		return err
	}

	sc1, err := s.CreateScene(ch1.ID, "Scene 1: The Cracked Astrolabe", 900)
	if err != nil {
		return err
	}
	sc1.Content = "Elena held the seventeenth-century crown glass up to the sodium lamp. Inside the annealing striae, a hairline fracture traced the exact contour of an archipelago no Admiralty chart admitted existed.\\n\\nOutside the Vault windows, the November tide struck the sea wall in slow, deliberate intervals. Julian set his brass calipers beside her tray of rouge powder.\\n\\n\\"You're looking at the third grinding mark,\\" Julian said quietly. \\"Spinoza didn't polish it out. He engraved a latitude.\\""
	sc1.SideNotes = "Establish the sensory contrast between the warm sodium lamp inside the Lantern Vault and the freezing storm surge outside.\\nForeshadow Maren's audit of Vault drawer 14."
	sc1.Status = StatusCompleted
	sc1.POVCharacterID = &elena.ID
	sc1.LocationID = &locArchive.ID
	sc1.CharacterIDs = []int64{elena.ID, julian.ID}
	if err := s.UpdateScene(sc1); err != nil {
		return err
	}

	sc2, err := s.CreateScene(ch1.ID, "Scene 2: Maren's Inventory", 1100)
	if err != nil {
		return err
	}
	sc2.Content = "Before the bell for night lockup finished its third chime, Archivist Maren stood on the iron gallery above the restoration tables. Her keyring did not rattle; she held the warded bronze key pinched between gloved fingers.\\n\\n\\"Crate nineteen from the Dogger Bank salvage,\\" Maren called down. \\"The Board requires the lens blanks sealed before midnight.\\""
	sc2.SideNotes = "Keep dialogue sparse. Elena swaps the genuine lens with the flint glass blank from Drawer 9."
	sc2.Status = StatusDrafting
	sc2.POVCharacterID = &elena.ID
	sc2.LocationID = &locArchive.ID
	sc2.CharacterIDs = []int64{elena.ID, maren.ID}
	if err := s.UpdateScene(sc2); err != nil {
		return err
	}

	sc3, err := s.CreateScene(ch2.ID, "Scene 1: Calibration at the Tower", 1200)
	if err != nil {
		return err
	}
	sc3.Content = "At moonrise they mounted the refractor in the Meridian Tower shutter. When the star Fomalhaut crossed the hairline wire, the phantom shoreline resolved in silver relief across the zinc projection plate."
	sc3.SideNotes = "Verify astronomical azimuth for late November at 54 degrees North."
	sc3.Status = StatusDrafting
	sc3.POVCharacterID = &julian.ID
	sc3.LocationID = &locObservatory.ID
	sc3.CharacterIDs = []int64{elena.ID, julian.ID}
	if err := s.UpdateScene(sc3); err != nil {
		return err
	}

	sc4, err := s.CreateScene(ch3.ID, "Scene 1: The Submerged Causeway", 1400)
	if err != nil {
		return err
	}
	sc4.Content = "Outline: At the equinoctial spring ebb, the granite survey markers surface for forty-two minutes."
	sc4.SideNotes = "Climax of Act II Part 1. Need sensory details of kelp, rusted iron rings, and foghorn intervals."
	sc4.Status = StatusIdea
	sc4.POVCharacterID = &julian.ID
	sc4.LocationID = &locObservatory.ID
	sc4.CharacterIDs = []int64{elena.ID, julian.ID, maren.ID}
	return s.UpdateScene(sc4)
}

// ListProjects returns all projects ordered by ID.
func (s *Store) ListProjects() ([]Project, error) {
	rows, err := s.db.Query(\`SELECT id, title, author, genre, synopsis, target_words, created_at, updated_at FROM projects ORDER BY id ASC\`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Title, &p.Author, &p.Genre, &p.Synopsis, &p.TargetWords, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// CreateProject inserts a new project into SQLite.
func (s *Store) CreateProject(title, author, genre string, targetWords int) (*Project, error) {
	if targetWords <= 0 {
		targetWords = 80000
	}
	res, err := s.db.Exec(
		\`INSERT INTO projects (title, author, genre, target_words) VALUES (?, ?, ?, ?)\`,
		strings.TrimSpace(title), strings.TrimSpace(author), strings.TrimSpace(genre), targetWords,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Project{
		ID:          id,
		Title:       title,
		Author:      author,
		Genre:       genre,
		TargetWords: targetWords,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// LoadHierarchy fetches the full Act -> Chapter -> Scene tree for a project.
func (s *Store) LoadHierarchy(projectID int64) ([]Act, error) {
	actRows, err := s.db.Query(\`SELECT id, project_id, title, sort_order FROM acts WHERE project_id = ? ORDER BY sort_order ASC, id ASC\`, projectID)
	if err != nil {
		return nil, err
	}
	defer actRows.Close()

	var acts []Act
	for actRows.Next() {
		var a Act
		if err := actRows.Scan(&a.ID, &a.ProjectID, &a.Title, &a.SortOrder); err != nil {
			return nil, err
		}
		acts = append(acts, a)
	}
	if err := actRows.Err(); err != nil {
		return nil, err
	}

	for i := range acts {
		chapters, err := s.listChaptersByAct(acts[i].ID)
		if err != nil {
			return nil, err
		}
		acts[i].Chapters = chapters
	}
	return acts, nil
}

func (s *Store) listChaptersByAct(actID int64) ([]Chapter, error) {
	rows, err := s.db.Query(\`SELECT id, act_id, title, target_words, sort_order FROM chapters WHERE act_id = ? ORDER BY sort_order ASC, id ASC\`, actID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chapters []Chapter
	for rows.Next() {
		var c Chapter
		if err := rows.Scan(&c.ID, &c.ActID, &c.Title, &c.TargetWords, &c.SortOrder); err != nil {
			return nil, err
		}
		chapters = append(chapters, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range chapters {
		scenes, err := s.listScenesByChapter(chapters[i].ID)
		if err != nil {
			return nil, err
		}
		chapters[i].Scenes = scenes
	}
	return chapters, nil
}

func (s *Store) listScenesByChapter(chapterID int64) ([]Scene, error) {
	rows, err := s.db.Query(\`
		SELECT id, chapter_id, title, content, side_notes, status, pov_character_id, location_id, target_words, word_count, sort_order, updated_at
		FROM scenes WHERE chapter_id = ? ORDER BY sort_order ASC, id ASC\`, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenes []Scene
	for rows.Next() {
		var sc Scene
		var statusStr string
		var povID, locID sql.NullInt64
		if err := rows.Scan(&sc.ID, &sc.ChapterID, &sc.Title, &sc.Content, &sc.SideNotes, &statusStr, &povID, &locID, &sc.TargetWords, &sc.WordCount, &sc.SortOrder, &sc.UpdatedAt); err != nil {
			return nil, err
		}
		sc.Status = SceneStatus(statusStr)
		if povID.Valid {
			v := povID.Int64
			sc.POVCharacterID = &v
		}
		if locID.Valid {
			v := locID.Int64
			sc.LocationID = &v
		}
		scenes = append(scenes, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range scenes {
		charIDs, err := s.GetSceneCharacters(scenes[i].ID)
		if err != nil {
			return nil, err
		}
		scenes[i].CharacterIDs = charIDs
	}
	return scenes, nil
}

// CreateAct appends a new Act to the given Project.
func (s *Store) CreateAct(projectID int64, title string) (*Act, error) {
	var nextOrder int
	_ = s.db.QueryRow(\`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM acts WHERE project_id = ?\`, projectID).Scan(&nextOrder)
	res, err := s.db.Exec(\`INSERT INTO acts (project_id, title, sort_order) VALUES (?, ?, ?)\`, projectID, strings.TrimSpace(title), nextOrder)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Act{ID: id, ProjectID: projectID, Title: title, SortOrder: nextOrder}, nil
}

// RenameAct updates an Act's title.
func (s *Store) RenameAct(actID int64, newTitle string) error {
	_, err := s.db.Exec(\`UPDATE acts SET title = ? WHERE id = ?\`, strings.TrimSpace(newTitle), actID)
	return err
}

// DeleteAct removes an Act and cascades to its Chapters and Scenes.
func (s *Store) DeleteAct(actID int64) error {
	_, err := s.db.Exec(\`DELETE FROM acts WHERE id = ?\`, actID)
	return err
}

// MoveAct swaps an Act's sort_order with its neighbor (-1 for up, +1 for down).
func (s *Store) MoveAct(actID int64, direction int) error {
	var projectID int64
	var currentOrder int
	if err := s.db.QueryRow(\`SELECT project_id, sort_order FROM acts WHERE id = ?\`, actID).Scan(&projectID, &currentOrder); err != nil {
		return err
	}
	var neighborID int64
	var neighborOrder int
	var query string
	if direction < 0 {
		query = \`SELECT id, sort_order FROM acts WHERE project_id = ? AND sort_order < ? ORDER BY sort_order DESC LIMIT 1\`
	} else {
		query = \`SELECT id, sort_order FROM acts WHERE project_id = ? AND sort_order > ? ORDER BY sort_order ASC LIMIT 1\`
	}
	if err := s.db.QueryRow(query, projectID, currentOrder).Scan(&neighborID, &neighborOrder); err != nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	_, _ = tx.Exec(\`UPDATE acts SET sort_order = ? WHERE id = ?\`, neighborOrder, actID)
	_, _ = tx.Exec(\`UPDATE acts SET sort_order = ? WHERE id = ?\`, currentOrder, neighborID)
	return tx.Commit()
}

// CreateChapter appends a new Chapter to the given Act.
func (s *Store) CreateChapter(actID int64, title string, targetWords int) (*Chapter, error) {
	if targetWords <= 0 {
		targetWords = 3000
	}
	var nextOrder int
	_ = s.db.QueryRow(\`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM chapters WHERE act_id = ?\`, actID).Scan(&nextOrder)
	res, err := s.db.Exec(\`INSERT INTO chapters (act_id, title, target_words, sort_order) VALUES (?, ?, ?, ?)\`, actID, strings.TrimSpace(title), targetWords, nextOrder)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Chapter{ID: id, ActID: actID, Title: title, TargetWords: targetWords, SortOrder: nextOrder}, nil
}

// UpdateChapter updates a Chapter's title and target word count.
func (s *Store) UpdateChapter(chapterID int64, title string, targetWords int) error {
	_, err := s.db.Exec(\`UPDATE chapters SET title = ?, target_words = ? WHERE id = ?\`, strings.TrimSpace(title), targetWords, chapterID)
	return err
}

// DeleteChapter deletes a Chapter and cascades to its Scenes.
func (s *Store) DeleteChapter(chapterID int64) error {
	_, err := s.db.Exec(\`DELETE FROM chapters WHERE id = ?\`, chapterID)
	return err
}

// MoveChapter swaps a Chapter's sort_order with its neighbor (-1 for up, +1 for down).
func (s *Store) MoveChapter(chapterID int64, direction int) error {
	var actID int64
	var currentOrder int
	if err := s.db.QueryRow(\`SELECT act_id, sort_order FROM chapters WHERE id = ?\`, chapterID).Scan(&actID, &currentOrder); err != nil {
		return err
	}
	var neighborID int64
	var neighborOrder int
	var query string
	if direction < 0 {
		query = \`SELECT id, sort_order FROM chapters WHERE act_id = ? AND sort_order < ? ORDER BY sort_order DESC LIMIT 1\`
	} else {
		query = \`SELECT id, sort_order FROM chapters WHERE act_id = ? AND sort_order > ? ORDER BY sort_order ASC LIMIT 1\`
	}
	if err := s.db.QueryRow(query, actID, currentOrder).Scan(&neighborID, &neighborOrder); err != nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	_, _ = tx.Exec(\`UPDATE chapters SET sort_order = ? WHERE id = ?\`, neighborOrder, chapterID)
	_, _ = tx.Exec(\`UPDATE chapters SET sort_order = ? WHERE id = ?\`, currentOrder, neighborID)
	return tx.Commit()
}

// CreateScene appends a new Scene to the given Chapter.
func (s *Store) CreateScene(chapterID int64, title string, targetWords int) (*Scene, error) {
	if targetWords <= 0 {
		targetWords = 1200
	}
	var nextOrder int
	_ = s.db.QueryRow(\`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM scenes WHERE chapter_id = ?\`, chapterID).Scan(&nextOrder)
	res, err := s.db.Exec(
		\`INSERT INTO scenes (chapter_id, title, status, target_words, sort_order) VALUES (?, ?, ?, ?, ?)\`,
		chapterID, strings.TrimSpace(title), string(StatusIdea), targetWords, nextOrder,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Scene{
		ID:          id,
		ChapterID:   chapterID,
		Title:       title,
		Status:      StatusIdea,
		TargetWords: targetWords,
		SortOrder:   nextOrder,
		UpdatedAt:   time.Now(),
	}, nil
}

// GetScene retrieves a single Scene by ID including its assigned character IDs.
func (s *Store) GetScene(sceneID int64) (*Scene, error) {
	var sc Scene
	var statusStr string
	var povID, locID sql.NullInt64
	err := s.db.QueryRow(\`
		SELECT id, chapter_id, title, content, side_notes, status, pov_character_id, location_id, target_words, word_count, sort_order, updated_at
		FROM scenes WHERE id = ?\`, sceneID,
	).Scan(&sc.ID, &sc.ChapterID, &sc.Title, &sc.Content, &sc.SideNotes, &statusStr, &povID, &locID, &sc.TargetWords, &sc.WordCount, &sc.SortOrder, &sc.UpdatedAt)
	if err != nil {
		return nil, err
	}
	sc.Status = SceneStatus(statusStr)
	if povID.Valid {
		v := povID.Int64
		sc.POVCharacterID = &v
	}
	if locID.Valid {
		v := locID.Int64
		sc.LocationID = &v
	}
	chars, err := s.GetSceneCharacters(sc.ID)
	if err != nil {
		return nil, err
	}
	sc.CharacterIDs = chars
	return &sc, nil
}

// GetChapter retrieves a Chapter and calculates its aggregate word count across all child Scenes.
func (s *Store) GetChapter(chapterID int64) (*Chapter, int, error) {
	var c Chapter
	err := s.db.QueryRow(\`SELECT id, act_id, title, target_words, sort_order FROM chapters WHERE id = ?\`, chapterID).
		Scan(&c.ID, &c.ActID, &c.Title, &c.TargetWords, &c.SortOrder)
	if err != nil {
		return nil, 0, err
	}
	var totalWords int
	_ = s.db.QueryRow(\`SELECT COALESCE(SUM(word_count), 0) FROM scenes WHERE chapter_id = ?\`, chapterID).Scan(&totalWords)
	return &c, totalWords, nil
}

// UpdateScene persists all prose content, side notes, metadata, word count, and character links.
func (s *Store) UpdateScene(sc *Scene) error {
	sc.WordCount = CountWords(sc.Content)
	sc.UpdatedAt = time.Now()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(\`
		UPDATE scenes
		SET title = ?, content = ?, side_notes = ?, status = ?, pov_character_id = ?, location_id = ?,
		    target_words = ?, word_count = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?\`,
		strings.TrimSpace(sc.Title), sc.Content, sc.SideNotes, string(sc.Status),
		sc.POVCharacterID, sc.LocationID, sc.TargetWords, sc.WordCount, sc.ID,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err := tx.Exec(\`DELETE FROM scene_characters WHERE scene_id = ?\`, sc.ID); err != nil {
		_ = tx.Rollback()
		return err
	}

	for _, charID := range sc.CharacterIDs {
		if _, err := tx.Exec(\`INSERT OR IGNORE INTO scene_characters (scene_id, character_id) VALUES (?, ?)\`, sc.ID, charID); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// DeleteScene removes a Scene from SQLite.
func (s *Store) DeleteScene(sceneID int64) error {
	_, err := s.db.Exec(\`DELETE FROM scenes WHERE id = ?\`, sceneID)
	return err
}

// MoveScene swaps a Scene's sort_order with its neighbor (-1 for up, +1 for down).
func (s *Store) MoveScene(sceneID int64, direction int) error {
	var chapterID int64
	var currentOrder int
	if err := s.db.QueryRow(\`SELECT chapter_id, sort_order FROM scenes WHERE id = ?\`, sceneID).Scan(&chapterID, &currentOrder); err != nil {
		return err
	}
	var neighborID int64
	var neighborOrder int
	var query string
	if direction < 0 {
		query = \`SELECT id, sort_order FROM scenes WHERE chapter_id = ? AND sort_order < ? ORDER BY sort_order DESC LIMIT 1\`
	} else {
		query = \`SELECT id, sort_order FROM scenes WHERE chapter_id = ? AND sort_order > ? ORDER BY sort_order ASC LIMIT 1\`
	}
	if err := s.db.QueryRow(query, chapterID, currentOrder).Scan(&neighborID, &neighborOrder); err != nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	_, _ = tx.Exec(\`UPDATE scenes SET sort_order = ? WHERE id = ?\`, neighborOrder, sceneID)
	_, _ = tx.Exec(\`UPDATE scenes SET sort_order = ? WHERE id = ?\`, currentOrder, neighborID)
	return tx.Commit()
}

// GetSceneCharacters returns all character IDs present in a given Scene.
func (s *Store) GetSceneCharacters(sceneID int64) ([]int64, error) {
	rows, err := s.db.Query(\`SELECT character_id FROM scene_characters WHERE scene_id = ? ORDER BY character_id ASC\`, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListCharacters returns all Characters registered to a Project.
func (s *Store) ListCharacters(projectID int64) ([]Character, error) {
	rows, err := s.db.Query(\`SELECT id, project_id, name, role, bio FROM characters WHERE project_id = ? ORDER BY name ASC\`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Character
	for rows.Next() {
		var c Character
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Role, &c.Bio); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// CreateCharacter adds a new Character to the Project cast.
func (s *Store) CreateCharacter(projectID int64, name, role, bio string) (*Character, error) {
	res, err := s.db.Exec(\`INSERT INTO characters (project_id, name, role, bio) VALUES (?, ?, ?, ?)\`,
		projectID, strings.TrimSpace(name), strings.TrimSpace(role), strings.TrimSpace(bio))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Character{ID: id, ProjectID: projectID, Name: name, Role: role, Bio: bio}, nil
}

// ListLocations returns all Locations registered to a Project.
func (s *Store) ListLocations(projectID int64) ([]Location, error) {
	rows, err := s.db.Query(\`SELECT id, project_id, name, description FROM locations WHERE project_id = ? ORDER BY name ASC\`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Location
	for rows.Next() {
		var l Location
		if err := rows.Scan(&l.ID, &l.ProjectID, &l.Name, &l.Description); err != nil {
			return nil, err
		}
		list = append(list, l)
	}
	return list, rows.Err()
}

// CreateLocation adds a new Location/Setting to the Project.
func (s *Store) CreateLocation(projectID int64, name, description string) (*Location, error) {
	res, err := s.db.Exec(\`INSERT INTO locations (project_id, name, description) VALUES (?, ?, ?)\`,
		projectID, strings.TrimSpace(name), strings.TrimSpace(description))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Location{ID: id, ProjectID: projectID, Name: name, Description: description}, nil
}

// ExportManuscriptMarkdown compiles all Acts, Chapters, and Scenes sequentially into clean Markdown.
func (s *Store) ExportManuscriptMarkdown(project Project) (string, error) {
	acts, err := s.LoadHierarchy(project.ID)
	if err != nil {
		return "", err
	}
	chars, _ := s.ListCharacters(project.ID)
	locs, _ := s.ListLocations(project.ID)

	charMap := make(map[int64]string)
	for _, c := range chars {
		charMap[c.ID] = c.Name
	}
	locMap := make(map[int64]string)
	for _, l := range locs {
		locMap[l.ID] = l.Name
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\\n\\n", project.Title))
	if project.Author != "" {
		b.WriteString(fmt.Sprintf("**By %s**  \\n", project.Author))
	}
	if project.Genre != "" {
		b.WriteString(fmt.Sprintf("*%s*\\n\\n", project.Genre))
	}
	b.WriteString("---\\n\\n")

	for _, act := range acts {
		b.WriteString(fmt.Sprintf("# %s\\n\\n", act.Title))
		for _, ch := range act.Chapters {
			b.WriteString(fmt.Sprintf("## %s\\n\\n", ch.Title))
			for sIdx, sc := range ch.Scenes {
				b.WriteString(fmt.Sprintf("### %s\\n\\n", sc.Title))
				var metaParts []string
				metaParts = append(metaParts, fmt.Sprintf("Status: %s", sc.Status))
				if sc.POVCharacterID != nil {
					if name, ok := charMap[*sc.POVCharacterID]; ok {
						metaParts = append(metaParts, fmt.Sprintf("POV: %s", name))
					}
				}
				if sc.LocationID != nil {
					if name, ok := locMap[*sc.LocationID]; ok {
						metaParts = append(metaParts, fmt.Sprintf("Setting: %s", name))
					}
				}
				b.WriteString(fmt.Sprintf("> *%s*\\n\\n", strings.Join(metaParts, " · ")))
				b.WriteString(strings.TrimSpace(sc.Content) + "\\n\\n")
				if sIdx < len(ch.Scenes)-1 {
					b.WriteString("* * *\\n\\n")
				}
			}
		}
	}
	return b.String(), nil
}

// ExportManuscriptHTML compiles all Acts, Chapters, and Scenes sequentially into a standalone typeset HTML document.
func (s *Store) ExportManuscriptHTML(project Project) (string, error) {
	acts, err := s.LoadHierarchy(project.ID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString(\`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>\` + html.EscapeString(project.Title) + \`</title>
<style>
  body { max-width: 44rem; margin: 4rem auto; padding: 0 1.5rem; font-family: Georgia, 'Cormorant Garamond', serif; line-height: 1.75; color: #1c1b18; background: #faf9f5; }
  header { text-align: center; margin-bottom: 4rem; border-bottom: 1px solid #dcd9d0; padding-bottom: 2rem; }
  h1.book-title { font-size: 2.5rem; margin-bottom: 0.25rem; }
  h2.act-title { font-size: 1.75rem; margin-top: 3.5rem; text-transform: uppercase; letter-spacing: 0.08em; border-bottom: 1px solid #e5e2d9; padding-bottom: 0.5rem; }
  h3.chapter-title { font-size: 1.4rem; margin-top: 2.5rem; }
  h4.scene-title { font-size: 1.05rem; color: #57534e; margin-top: 1.5rem; }
  p { margin: 1rem 0; text-indent: 1.5rem; }
  hr.scene-break { border: none; text-align: center; margin: 2rem 0; }
  hr.scene-break::after { content: "* * *"; letter-spacing: 0.5rem; color: #78716c; }
</style>
</head>
<body>
<header>
  <h1 class="book-title">\` + html.EscapeString(project.Title) + \`</h1>
  <div>By \` + html.EscapeString(project.Author) + \` &middot; \` + html.EscapeString(project.Genre) + \`</div>
</header>
\`)

	for _, act := range acts {
		b.WriteString(fmt.Sprintf(\`<h2 class="act-title">%s</h2>\`+"\\n", html.EscapeString(act.Title)))
		for _, ch := range act.Chapters {
			b.WriteString(fmt.Sprintf(\`<h3 class="chapter-title">%s</h3>\`+"\\n", html.EscapeString(ch.Title)))
			for sIdx, sc := range ch.Scenes {
				b.WriteString(fmt.Sprintf(\`<h4 class="scene-title">%s</h4>\`+"\\n", html.EscapeString(sc.Title)))
				paragraphs := strings.Split(strings.TrimSpace(sc.Content), "\\n\\n")
				for _, para := range paragraphs {
					clean := strings.TrimSpace(para)
					if clean != "" {
						b.WriteString(fmt.Sprintf("<p>%s</p>\\n", html.EscapeString(clean)))
					}
				}
				if sIdx < len(ch.Scenes)-1 {
					b.WriteString(\`<hr class="scene-break" />\` + "\\n")
				}
			}
		}
	}
	b.WriteString("</body>\\n</html>\\n")
	return b.String(), nil
}`,
  },
  {
    filename: 'ui_main.go',
    path: 'gonovelist/ui_main.go',
    layer: 'UI Layer (Fyne v2)',
    summary:
      'Constructs the primary Fyne window, top action toolbar, hierarchical widget.Tree (Act -> Chapter -> Scene) with create/rename/move/delete controls, Cast & Locations modal dialog, and Markdown/HTML export actions.',
    code: `package main

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
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			return ui.childrenMap[uid]
		},
		func(uid widget.TreeNodeID) bool {
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
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			c := obj.(*fyne.Container)
			titleLbl := c.Objects[0].(*widget.Label)
			metaLbl := c.Objects[1].(*widget.Label)

			node, ok := ui.nodeLookup[uid]
			if !ok {
				return
			}
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
			titleLbl.SetText(node.Title)
			titleLbl.Refresh()
		},
	)

	ui.tree.OnSelected = func(uid widget.TreeNodeID) {
		ui.selectedUID = uid
		node, ok := ui.nodeLookup[uid]
		if !ok {
			return
		}
		if node.Kind == NodeScene {
			ui.editorPanel.LoadScene(ui.activeProject.ID, node.ID)
		}
	}

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
		fmt.Sprintf("Compiled sequential Acts, Chapters, and Scenes to:\\n%s", outName),
		ui.window)
}`,
  },
  {
    filename: 'ui_editor.go',
    path: 'gonovelist/ui_editor.go',
    layer: 'UI Layer (Fyne v2)',
    summary:
      'Implements the center multi-line prose editor + Markdown RichText preview, thread-safe 750ms debounced auto-save timer, real-time Scene & Chapter word-count ProgressBars, Context/Metadata mapping (Status, POV, Location, Characters CheckGroup), and Side Notes tab.',
    code: `package main

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

	boldBtn := widget.NewButton("B", func() { ep.insertSnippet("**bold**") })
	italicBtn := widget.NewButton("I", func() { ep.insertSnippet("*italic*") })
	h2Btn := widget.NewButton("H2", func() { ep.insertSnippet("\\n## Subheading\\n") })
	quoteBtn := widget.NewButton("Quote", func() { ep.insertSnippet("\\n> ") })
	breakBtn := widget.NewButton("* * *", func() { ep.insertSnippet("\\n\\n* * *\\n\\n") })

	headerBar := container.NewBorder(
		nil, nil,
		container.NewHBox(boldBtn, italicBtn, h2Btn, quoteBtn, breakBtn),
		ep.saveStateLabel,
		ep.sceneTitleEntry,
	)

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
}`,
  },
  {
    filename: 'schema.sql',
    path: 'gonovelist/schema.sql',
    layer: 'Database DDL',
    summary:
      'Complete SQLite relational schema enforcing foreign key cascades across Projects -> Acts -> Chapters -> Scenes, plus Characters, Locations, and the scene_characters many-to-many join table.',
    code: `-- GoNovelist SQLite Database Schema
-- Enables foreign key constraints for cascading deletes across the hierarchy.
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL,
    author TEXT NOT NULL DEFAULT '',
    genre TEXT NOT NULL DEFAULT '',
    synopsis TEXT NOT NULL DEFAULT '',
    target_words INTEGER NOT NULL DEFAULT 80000,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS acts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS chapters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    act_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    target_words INTEGER NOT NULL DEFAULT 3000,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (act_id) REFERENCES acts(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS characters (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'Supporting',
    bio TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS locations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS scenes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chapter_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    side_notes TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'Idea' CHECK (status IN ('Idea', 'Drafting', 'Completed', 'Edited')),
    pov_character_id INTEGER NULL,
    location_id INTEGER NULL,
    target_words INTEGER NOT NULL DEFAULT 1200,
    word_count INTEGER NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chapter_id) REFERENCES chapters(id) ON DELETE CASCADE,
    FOREIGN KEY (pov_character_id) REFERENCES characters(id) ON DELETE SET NULL,
    FOREIGN KEY (location_id) REFERENCES locations(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS scene_characters (
    scene_id INTEGER NOT NULL,
    character_id INTEGER NOT NULL,
    PRIMARY KEY (scene_id, character_id),
    FOREIGN KEY (scene_id) REFERENCES scenes(id) ON DELETE CASCADE,
    FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_acts_project_sort ON acts(project_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_chapters_act_sort ON chapters(act_id, sort_order);
CREATE INDEX IF NOT EXISTS idx_scenes_chapter_sort ON scenes(chapter_id, sort_order);`,
  },
  {
    filename: 'go.mod',
    path: 'gonovelist/go.mod',
    layer: 'Module Config',
    summary:
      'Go 1.22+ module specification declaring fyne.io/fyne/v2 and modernc.org/sqlite for zero-config cross-platform desktop compilation.',
    code: `module gonovelist

go 1.22

require (
	fyne.io/fyne/v2 v2.5.3
	modernc.org/sqlite v1.34.4
)`,
  },
];

export const BUILD_COMMANDS = `# 1. Create the project directory and initialize the Go module
mkdir -p gonovelist && cd gonovelist
go mod init gonovelist

# 2. Install Fyne v2 and the SQLite driver
go get fyne.io/fyne/v2@v2.5.3
go get modernc.org/sqlite@v1.34.4
go mod tidy

# (Optional) If you prefer CGO-based mattn/go-sqlite3 instead of pure-Go modernc.org/sqlite:
# go get github.com/mattn/go-sqlite3
# Then change the blank import in database.go to: _ "github.com/mattn/go-sqlite3"
# and sql.Open("sqlite3", dbPath)

# 3. Run GoNovelist directly in development mode
go run .

# 4. Build an optimized native desktop binary
# Linux / macOS:
go build -ldflags="-s -w" -o gonovelist .

# Windows (hides console window):
go build -ldflags="-s -w -H=windowsgui" -o GoNovelist.exe .

# 5. Package as a native OS bundle (.app / .exe / .tar.xz) using the Fyne CLI
go install fyne.io/fyne/v2/cmd/fyne@latest
fyne package -name "GoNovelist" -appID "io.gonovelist.desktop"`;
