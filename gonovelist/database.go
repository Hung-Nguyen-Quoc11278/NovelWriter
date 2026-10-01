package main

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
	ddl := `
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
`
	_, err := s.db.Exec(ddl)
	return err
}

// SeedIfNeeded populates a complete starter manuscript if no projects exist yet.
func (s *Store) SeedIfNeeded() error {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&count); err != nil {
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
	sc1.Content = "Elena held the seventeenth-century crown glass up to the sodium lamp. Inside the annealing striae, a hairline fracture traced the exact contour of an archipelago no Admiralty chart admitted existed.\n\nOutside the Vault windows, the November tide struck the sea wall in slow, deliberate intervals. Julian set his brass calipers beside her tray ofrouge powder.\n\n\"You're looking at the third grinding mark,\" Julian said quietly. \"Spinoza didn't polish it out. He engraved a latitude.\""
	sc1.SideNotes = "Establish the sensory contrast between the warm sodium lamp inside the Lantern Vault and the freezing storm surge outside.\n Foreshadow Maren's audit of Vault drawer 14."
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
	sc2.Content = "Before the bell for night lockup finished its third chime, Archivist Maren stood on the iron gallery above the restoration tables. Her keyring did not rattle; she held the warded bronze key pinched between gloved fingers.\n\n\"Crate nineteen from the Dogger Bank salvage,\" Maren called down. \"The Board requires the lens blanks sealed before midnight.\""
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

// ListProjects returns all projects ordered by most recently updated.
func (s *Store) ListProjects() ([]Project, error) {
	rows, err := s.db.Query(`SELECT id, title, author, genre, synopsis, target_words, created_at, updated_at FROM projects ORDER BY id ASC`)
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
		`INSERT INTO projects (title, author, genre, target_words) VALUES (?, ?, ?, ?)`,
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
	actRows, err := s.db.Query(`SELECT id, project_id, title, sort_order FROM acts WHERE project_id = ? ORDER BY sort_order ASC, id ASC`, projectID)
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
	rows, err := s.db.Query(`SELECT id, act_id, title, target_words, sort_order FROM chapters WHERE act_id = ? ORDER BY sort_order ASC, id ASC`, actID)
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
	rows, err := s.db.Query(`
		SELECT id, chapter_id, title, content, side_notes, status, pov_character_id, location_id, target_words, word_count, sort_order, updated_at
		FROM scenes WHERE chapter_id = ? ORDER BY sort_order ASC, id ASC`, chapterID)
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
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM acts WHERE project_id = ?`, projectID).Scan(&nextOrder)
	res, err := s.db.Exec(`INSERT INTO acts (project_id, title, sort_order) VALUES (?, ?, ?)`, projectID, strings.TrimSpace(title), nextOrder)
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
	_, err := s.db.Exec(`UPDATE acts SET title = ? WHERE id = ?`, strings.TrimSpace(newTitle), actID)
	return err
}

// DeleteAct removes an Act and cascades to its Chapters and Scenes.
func (s *Store) DeleteAct(actID int64) error {
	_, err := s.db.Exec(`DELETE FROM acts WHERE id = ?`, actID)
	return err
}

// MoveAct swaps an Act's sort_order with its neighbor (-1 for up, +1 for down).
func (s *Store) MoveAct(actID int64, direction int) error {
	var projectID int64
	var currentOrder int
	if err := s.db.QueryRow(`SELECT project_id, sort_order FROM acts WHERE id = ?`, actID).Scan(&projectID, &currentOrder); err != nil {
		return err
	}
	var neighborID int64
	var neighborOrder int
	var query string
	if direction < 0 {
		query = `SELECT id, sort_order FROM acts WHERE project_id = ? AND sort_order < ? ORDER BY sort_order DESC LIMIT 1`
	} else {
		query = `SELECT id, sort_order FROM acts WHERE project_id = ? AND sort_order > ? ORDER BY sort_order ASC LIMIT 1`
	}
	if err := s.db.QueryRow(query, projectID, currentOrder).Scan(&neighborID, &neighborOrder); err != nil {
		return nil // Already at boundary
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	_, _ = tx.Exec(`UPDATE acts SET sort_order = ? WHERE id = ?`, neighborOrder, actID)
	_, _ = tx.Exec(`UPDATE acts SET sort_order = ? WHERE id = ?`, currentOrder, neighborID)
	return tx.Commit()
}

// CreateChapter appends a new Chapter to the given Act.
func (s *Store) CreateChapter(actID int64, title string, targetWords int) (*Chapter, error) {
	if targetWords <= 0 {
		targetWords = 3000
	}
	var nextOrder int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM chapters WHERE act_id = ?`, actID).Scan(&nextOrder)
	res, err := s.db.Exec(`INSERT INTO chapters (act_id, title, target_words, sort_order) VALUES (?, ?, ?, ?)`, actID, strings.TrimSpace(title), targetWords, nextOrder)
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
	_, err := s.db.Exec(`UPDATE chapters SET title = ?, target_words = ? WHERE id = ?`, strings.TrimSpace(title), targetWords, chapterID)
	return err
}

// DeleteChapter deletes a Chapter and cascades to its Scenes.
func (s *Store) DeleteChapter(chapterID int64) error {
	_, err := s.db.Exec(`DELETE FROM chapters WHERE id = ?`, chapterID)
	return err
}

// MoveChapter swaps a Chapter's sort_order with its neighbor (-1 for up, +1 for down).
func (s *Store) MoveChapter(chapterID int64, direction int) error {
	var actID int64
	var currentOrder int
	if err := s.db.QueryRow(`SELECT act_id, sort_order FROM chapters WHERE id = ?`, chapterID).Scan(&actID, &currentOrder); err != nil {
		return err
	}
	var neighborID int64
	var neighborOrder int
	var query string
	if direction < 0 {
		query = `SELECT id, sort_order FROM chapters WHERE act_id = ? AND sort_order < ? ORDER BY sort_order DESC LIMIT 1`
	} else {
		query = `SELECT id, sort_order FROM chapters WHERE act_id = ? AND sort_order > ? ORDER BY sort_order ASC LIMIT 1`
	}
	if err := s.db.QueryRow(query, actID, currentOrder).Scan(&neighborID, &neighborOrder); err != nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	_, _ = tx.Exec(`UPDATE chapters SET sort_order = ? WHERE id = ?`, neighborOrder, chapterID)
	_, _ = tx.Exec(`UPDATE chapters SET sort_order = ? WHERE id = ?`, currentOrder, neighborID)
	return tx.Commit()
}

// CreateScene appends a new Scene to the given Chapter.
func (s *Store) CreateScene(chapterID int64, title string, targetWords int) (*Scene, error) {
	if targetWords <= 0 {
		targetWords = 1200
	}
	var nextOrder int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM scenes WHERE chapter_id = ?`, chapterID).Scan(&nextOrder)
	res, err := s.db.Exec(
		`INSERT INTO scenes (chapter_id, title, status, target_words, sort_order) VALUES (?, ?, ?, ?, ?)`,
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
	err := s.db.QueryRow(`
		SELECT id, chapter_id, title, content, side_notes, status, pov_character_id, location_id, target_words, word_count, sort_order, updated_at
		FROM scenes WHERE id = ?`, sceneID,
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
	err := s.db.QueryRow(`SELECT id, act_id, title, target_words, sort_order FROM chapters WHERE id = ?`, chapterID).
		Scan(&c.ID, &c.ActID, &c.Title, &c.TargetWords, &c.SortOrder)
	if err != nil {
		return nil, 0, err
	}
	var totalWords int
	_ = s.db.QueryRow(`SELECT COALESCE(SUM(word_count), 0) FROM scenes WHERE chapter_id = ?`, chapterID).Scan(&totalWords)
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

	_, err = tx.Exec(`
		UPDATE scenes
		SET title = ?, content = ?, side_notes = ?, status = ?, pov_character_id = ?, location_id = ?,
		    target_words = ?, word_count = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		strings.TrimSpace(sc.Title), sc.Content, sc.SideNotes, string(sc.Status),
		sc.POVCharacterID, sc.LocationID, sc.TargetWords, sc.WordCount, sc.ID,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if _, err := tx.Exec(`DELETE FROM scene_characters WHERE scene_id = ?`, sc.ID); err != nil {
		_ = tx.Rollback()
		return err
	}

	for _, charID := range sc.CharacterIDs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO scene_characters (scene_id, character_id) VALUES (?, ?)`, sc.ID, charID); err != nil {
			_ = tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

// DeleteScene removes a Scene from SQLite.
func (s *Store) DeleteScene(sceneID int64) error {
	_, err := s.db.Exec(`DELETE FROM scenes WHERE id = ?`, sceneID)
	return err
}

// MoveScene swaps a Scene's sort_order with its neighbor (-1 for up, +1 for down).
func (s *Store) MoveScene(sceneID int64, direction int) error {
	var chapterID int64
	var currentOrder int
	if err := s.db.QueryRow(`SELECT chapter_id, sort_order FROM scenes WHERE id = ?`, sceneID).Scan(&chapterID, &currentOrder); err != nil {
		return err
	}
	var neighborID int64
	var neighborOrder int
	var query string
	if direction < 0 {
		query = `SELECT id, sort_order FROM scenes WHERE chapter_id = ? AND sort_order < ? ORDER BY sort_order DESC LIMIT 1`
	} else {
		query = `SELECT id, sort_order FROM scenes WHERE chapter_id = ? AND sort_order > ? ORDER BY sort_order ASC LIMIT 1`
	}
	if err := s.db.QueryRow(query, chapterID, currentOrder).Scan(&neighborID, &neighborOrder); err != nil {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	_, _ = tx.Exec(`UPDATE scenes SET sort_order = ? WHERE id = ?`, neighborOrder, sceneID)
	_, _ = tx.Exec(`UPDATE scenes SET sort_order = ? WHERE id = ?`, currentOrder, neighborID)
	return tx.Commit()
}

// GetSceneCharacters returns all character IDs present in a given Scene.
func (s *Store) GetSceneCharacters(sceneID int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT character_id FROM scene_characters WHERE scene_id = ? ORDER BY character_id ASC`, sceneID)
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
	rows, err := s.db.Query(`SELECT id, project_id, name, role, bio FROM characters WHERE project_id = ? ORDER BY name ASC`, projectID)
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
	res, err := s.db.Exec(`INSERT INTO characters (project_id, name, role, bio) VALUES (?, ?, ?, ?)`,
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
	rows, err := s.db.Query(`SELECT id, project_id, name, description FROM locations WHERE project_id = ? ORDER BY name ASC`, projectID)
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
	res, err := s.db.Exec(`INSERT INTO locations (project_id, name, description) VALUES (?, ?, ?)`,
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
	b.WriteString(fmt.Sprintf("# %s\n\n", project.Title))
	if project.Author != "" {
		b.WriteString(fmt.Sprintf("**By %s**  \n", project.Author))
	}
	if project.Genre != "" {
		b.WriteString(fmt.Sprintf("*%s*\n\n", project.Genre))
	}
	b.WriteString("---\n\n")

	for _, act := range acts {
		b.WriteString(fmt.Sprintf("# %s\n\n", act.Title))
		for _, ch := range act.Chapters {
			b.WriteString(fmt.Sprintf("## %s\n\n", ch.Title))
			for sIdx, sc := range ch.Scenes {
				b.WriteString(fmt.Sprintf("### %s\n\n", sc.Title))
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
				b.WriteString(fmt.Sprintf("> *%s*\n\n", strings.Join(metaParts, " · ")))
				b.WriteString(strings.TrimSpace(sc.Content) + "\n\n")
				if sIdx < len(ch.Scenes)-1 {
					b.WriteString("* * *\n\n")
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
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>` + html.EscapeString(project.Title) + `</title>
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
  <h1 class="book-title">` + html.EscapeString(project.Title) + `</h1>
  <div>By ` + html.EscapeString(project.Author) + ` &middot; ` + html.EscapeString(project.Genre) + `</div>
</header>
`)

	for _, act := range acts {
		b.WriteString(fmt.Sprintf(`<h2 class="act-title">%s</h2>`+"\n", html.EscapeString(act.Title)))
		for _, ch := range act.Chapters {
			b.WriteString(fmt.Sprintf(`<h3 class="chapter-title">%s</h3>`+"\n", html.EscapeString(ch.Title)))
			for sIdx, sc := range ch.Scenes {
				b.WriteString(fmt.Sprintf(`<h4 class="scene-title">%s</h4>`+"\n", html.EscapeString(sc.Title)))
				paragraphs := strings.Split(strings.TrimSpace(sc.Content), "\n\n")
				for _, para := range paragraphs {
					clean := strings.TrimSpace(para)
					if clean != "" {
						b.WriteString(fmt.Sprintf("<p>%s</p>\n", html.EscapeString(clean)))
					}
				}
				if sIdx < len(ch.Scenes)-1 {
					b.WriteString(`<hr class="scene-break" />` + "\n")
				}
			}
		}
	}
	b.WriteString("</body>\n</html>\n")
	return b.String(), nil
}
