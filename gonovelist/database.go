package main

import (
	"database/sql"
	"fmt"
	"html"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Store đóng gói toàn bộ tầng truy xuất dữ liệu SQLite và dịch vụ nghiệp vụ.
type Store struct {
	db *sql.DB
}

// NewStore mở kết nối SQLite cục bộ, kích hoạt khóa ngoại + chế độ WAL và khởi tạo bảng.
func NewStore(dbPath string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("không thể mở sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.Migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.SeedDefaultProjectIfEmpty(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close đóng kết nối cơ sở dữ liệu SQLite.
func (s *Store) Close() error {
	return s.db.Close()
}

// Migrate thực thi các câu lệnh DDL để tạo cấu trúc bảng nếu chưa tồn tại.
func (s *Store) Migrate() error {
	ddl := `
	PRAGMA foreign_keys = ON;

	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author TEXT NOT NULL DEFAULT '',
		genre TEXT NOT NULL DEFAULT '',
		synopsis TEXT NOT NULL DEFAULT '',
		target_words INTEGER NOT NULL DEFAULT 50000,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS acts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		position INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS chapters (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		act_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		position INTEGER NOT NULL DEFAULT 1,
		target_words INTEGER NOT NULL DEFAULT 3000,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (act_id) REFERENCES acts(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS characters (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'Nhân vật chính',
		description TEXT NOT NULL DEFAULT '',
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
		summary TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL DEFAULT '',
		side_notes TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'Ý tưởng',
		pov_character_id INTEGER NULL,
		location_id INTEGER NULL,
		target_words INTEGER NOT NULL DEFAULT 1200,
		word_count INTEGER NOT NULL DEFAULT 0,
		position INTEGER NOT NULL DEFAULT 1,
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
	`
	_, err := s.db.Exec(ddl)
	return err
}

// SeedDefaultProjectIfEmpty tạo một dự án tiểu thuyết mẫu tiếng Việt nếu cơ sở dữ liệu đang trống.
func (s *Store) SeedDefaultProjectIfEmpty() error {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := s.SeedVietnameseSampleProject()
	return err
}

// SeedVietnameseSampleProject khởi tạo một tác phẩm mẫu tiếng Việt hoàn chỉnh.
func (s *Store) SeedVietnameseSampleProject() (*Project, error) {
	proj, err := s.CreateProject(
		"Bản Đồ Thủy Tinh Thành Hội An",
		"Nguyễn Minh Khuê",
		"Tiểu thuyết Lịch sử & Kỳ ảo",
		"Một nghệ nhân phục chế cổ vật tại thương cảng Hội An thế kỷ XIX phát hiện tấm hải đồ khắc ẩn trong thấu kính thủy tinh của ngọn hải đăng cổ.",
		45000,
	)
	if err != nil {
		return nil, err
	}

	elena, err := s.CreateCharacter(proj.ID, "Lê Ngọc Liên", "Nhân vật chính", "Nghệ nhân chế tác và phục chế thấu kính tại phố cổ Hội An.")
	if err != nil {
		return nil, err
	}
	julian, err := s.CreateCharacter(proj.ID, "Trần Đình Bách", "Đồng hành", "Thuyền trưởng tàu buôn từng đi qua vùng biển sương mù Nam Hải.")
	if err != nil {
		return nil, err
	}
	archivist, err := s.CreateCharacter(proj.ID, "Cụ Thủ Từ Họ Phạm", "Người dẫn đường", "Người trông coi kho thư tịch cổ tại hội quán.")
	if err != nil {
		return nil, err
	}

	observatory, err := s.CreateLocation(proj.ID, "Xưởng Thủy Tinh Phố Cổ", "Căn gác gỗ nhìn ra sông Thu Bồn với những lò nung pha lê và bàn mài thấu kính.")
	if err != nil {
		return nil, err
	}
	vault, err := s.CreateLocation(proj.ID, "Thư Các Chùa Cầu", "Căn phòng lưu trữ bản đồ hàng hải và nhật ký thương thuyền trăm năm.")
	if err != nil {
		return nil, err
	}

	act1, err := s.CreateAct(proj.ID, "Hồi I — Vết Rạn Trong Thấu Kính")
	if err != nil {
		return nil, err
	}
	act2, err := s.CreateAct(proj.ID, "Hồi II — Hải Trình Ngoài Sương Mù")
	if err != nil {
		return nil, err
	}

	ch1, err := s.CreateChapter(act1.ID, "Chương 1: Ánh Đèn Dầu Bên Sông Thu Bồn", 3000)
	if err != nil {
		return nil, err
	}
	ch2, err := s.CreateChapter(act1.ID, "Chương 2: Mật Mã Trên Bản Đồ Hàng Hải", 3000)
	if err != nil {
		return nil, err
	}
	ch3, err := s.CreateChapter(act2.ID, "Chương 3: Con Tàu Khởi Hành Lúc Nửa Đêm", 3500)
	if err != nil {
		return nil, err
	}

	scene1Content := `Ngọc Liên nâng phiến thủy tinh cổ thế kỷ mười bảy lên trước ngọn đèn dầu lạc. Bên trong lớp pha lê trong vắt, một vết rạn mảnh như sợi tơ bỗng khúc xạ ánh sáng vàng ấm thành hình dáng một quần đảo chưa từng xuất hiện trên bất kỳ tấm bản đồ hàng hải nào của triều đình.

Ngoài hiên gỗ, tiếng nước sông Thu Bồn vỗ nhẹ vào mạn thuyền buôn dưới màn sương đầu thu.

"Cô nhìn kỹ góc lệch của chùm sáng xem," Trần Đình Bách vừa nói vừa đặt chiếc la bàn đồng cũ kỹ lên mặt bàn gỗ lim. "Đó không phải vết nứt ngẫu nhiên khi làm nguội thủy tinh. Người thợ xưa đã cố tình giấu tọa độ vào độ cong của thấu kính."`

	sc1, err := s.CreateScene(ch1.ID, "Cảnh 1: Thấu Kính Khúc Xạ", 1200)
	if err != nil {
		return nil, err
	}
	sc1.Summary = "Ngọc Liên và Đình Bách phát hiện tấm hải đồ ẩn hiện qua ánh đèn xuyên qua thấu kính cổ."
	sc1.Content = scene1Content
	sc1.SideNotes = "Ghi chú: Nhấn mạnh chi tiết mùi dầu thông và tiếng chuông chùa xa xăm. Bổ sung mô tả chiếc la bàn đồng của Đình Bách."
	sc1.Status = StatusCompleted
	sc1.POVCharacterID = &elena.ID
	sc1.LocationID = &observatory.ID
	sc1.CharacterIDs = []int64{elena.ID, julian.ID}
	if err := s.UpdateScene(sc1); err != nil {
		return nil, err
	}

	scene2Content := `Cụ Thủ Từ họ Phạm chậm rãi mở chiếc rương gỗ trầm hương khóa đồng. Bên trong không phải vàng bạc mà là những cuộn giấy dó đã ngả màu thời gian, ghi chép nhật ký của đội thương thuyền mất tích tám mươi năm trước.

"Mỗi thấu kính được đúc thành một cặp song sinh," cụ trầm giọng nói, ngón tay gầy guộc chỉ vào dòng chữ Hán Nôm viết bằng mực chu sa. "Một phiến đặt trên đỉnh hải đăng Cù Lao Chàm, phiến kia nằm trong tay người hoa tiêu."`

	sc2, err := s.CreateScene(ch1.ID, "Cảnh 2: Cuộn Nhật Ký Bằng Giấy Dó", 1200)
	if err != nil {
		return nil, err
	}
	sc2.Summary = "Cụ Thủ Từ tiết lộ bí mật về cặp thấu kính song sinh và đội thuyền mất tích."
	sc2.Content = scene2Content
	sc2.SideNotes = "Cần kiểm tra lại thuật ngữ hàng hải cổ thế kỷ XIX tại vùng biển miền Trung."
	sc2.Status = StatusDrafting
	sc2.POVCharacterID = &elena.ID
	sc2.LocationID = &vault.ID
	sc2.CharacterIDs = []int64{elena.ID, julian.ID, archivist.ID}
	if err := s.UpdateScene(sc2); err != nil {
		return nil, err
	}

	sc3, err := s.CreateScene(ch2.ID, "Cảnh 1: Đối Chiếu Sao Khuê", 1500)
	if err != nil {
		return nil, err
	}
	sc3.Summary = "Đình Bách tính toán góc phương vị từ bản đồ khúc xạ để tìm cửa biển ẩn."
	sc3.Content = "Đình Bách trải tấm hải đồ da dê lên mặt bàn, dùng thước đo góc bằng đồng đối chiếu từng chấm sáng phản chiếu từ thấu kính..."
	sc3.SideNotes = "Ý tưởng mở rộng: Có người lạ mặt theo dõi xưởng thủy tinh từ bên kia bờ sông."
	sc3.Status = StatusIdea
	sc3.POVCharacterID = &julian.ID
	sc3.LocationID = &observatory.ID
	sc3.CharacterIDs = []int64{elena.ID, julian.ID}
	if err := s.UpdateScene(sc3); err != nil {
		return nil, err
	}

	sc4, err := s.CreateScene(ch3.ID, "Cảnh 1: Nhổ Neo Trong Đêm Sương", 1500)
	if err != nil {
		return nil, err
	}
	sc4.Summary = "Con thuyền rời bến Cửa Đại khi thủy triều lên cao."
	sc4.Content = "Tiếng kéo buồm kẽo kẹt vang lên giữa màn sương đặc quánh. Ngọc Liên ôm chặt chiếc hộp gỗ đựng thấu kính trước ngực..."
	sc4.SideNotes = "Chuẩn bị cao trào cuối Hồi II khi sương mù tách ra để lộ ngọn hải đăng cổ."
	sc4.Status = StatusIdea
	sc4.POVCharacterID = &elena.ID
	sc4.LocationID = &observatory.ID
	sc4.CharacterIDs = []int64{elena.ID, julian.ID}
	if err := s.UpdateScene(sc4); err != nil {
		return nil, err
	}

	return proj, nil
}

// ListProjects trả về danh sách tất cả các dự án tiểu thuyết.
func (s *Store) ListProjects() ([]Project, error) {
	rows, err := s.db.Query(`SELECT id, title, author, genre, synopsis, target_words, created_at, updated_at FROM projects ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Title, &p.Author, &p.Genre, &p.Synopsis, &p.TargetWords, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreateProject tạo mới một dự án tiểu thuyết.
func (s *Store) CreateProject(title, author, genre, synopsis string, targetWords int) (*Project, error) {
	if targetWords <= 0 {
		targetWords = 50000
	}
	res, err := s.db.Exec(
		`INSERT INTO projects (title, author, genre, synopsis, target_words) VALUES (?, ?, ?, ?, ?)`,
		strings.TrimSpace(title), strings.TrimSpace(author), strings.TrimSpace(genre), strings.TrimSpace(synopsis), targetWords,
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
		Synopsis:    synopsis,
		TargetWords: targetWords,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// ListActs trả về các Hồi của một dự án theo thứ tự vị trí.
func (s *Store) ListActs(projectID int64) ([]Act, error) {
	rows, err := s.db.Query(`SELECT id, project_id, title, position, created_at FROM acts WHERE project_id = ? ORDER BY position ASC, id ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Act
	for rows.Next() {
		var a Act
		if err := rows.Scan(&a.ID, &a.ProjectID, &a.Title, &a.Position, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// CreateAct thêm một Hồi mới vào cuối dự án.
func (s *Store) CreateAct(projectID int64, title string) (*Act, error) {
	var nextPos int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(position), 0) + 1 FROM acts WHERE project_id = ?`, projectID).Scan(&nextPos)
	res, err := s.db.Exec(`INSERT INTO acts (project_id, title, position) VALUES (?, ?, ?)`, projectID, strings.TrimSpace(title), nextPos)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Act{ID: id, ProjectID: projectID, Title: title, Position: nextPos, CreatedAt: time.Now()}, nil
}

// RenameAct cập nhật tiêu đề của một Hồi.
func (s *Store) RenameAct(actID int64, newTitle string) error {
	_, err := s.db.Exec(`UPDATE acts SET title = ? WHERE id = ?`, strings.TrimSpace(newTitle), actID)
	return err
}

// DeleteAct xóa một Hồi và toàn bộ các Chương, Cảnh bên trong (nhờ ON DELETE CASCADE).
func (s *Store) DeleteAct(actID int64) error {
	_, err := s.db.Exec(`DELETE FROM acts WHERE id = ?`, actID)
	return err
}

// MoveAct đổi vị trí của Hồi lên hoặc xuống (-1 hoặc +1).
func (s *Store) MoveAct(actID int64, direction int) error {
	var projectID int64
	if err := s.db.QueryRow(`SELECT project_id FROM acts WHERE id = ?`, actID).Scan(&projectID); err != nil {
		return err
	}
	acts, err := s.ListActs(projectID)
	if err != nil {
		return err
	}
	idx := -1
	for i, a := range acts {
		if a.ID == actID {
			idx = i
			break
		}
	}
	targetIdx := idx + direction
	if idx < 0 || targetIdx < 0 || targetIdx >= len(acts) {
		return nil
	}
	_, err = s.db.Exec(`UPDATE acts SET position = ? WHERE id = ?`, acts[targetIdx].Position, acts[idx].ID)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE acts SET position = ? WHERE id = ?`, acts[idx].Position, acts[targetIdx].ID)
	return err
}

// ListChapters trả về các Chương thuộc một Hồi theo thứ tự vị trí.
func (s *Store) ListChapters(actID int64) ([]Chapter, error) {
	rows, err := s.db.Query(`SELECT id, act_id, title, position, target_words, created_at FROM chapters WHERE act_id = ? ORDER BY position ASC, id ASC`, actID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Chapter
	for rows.Next() {
		var c Chapter
		if err := rows.Scan(&c.ID, &c.ActID, &c.Title, &c.Position, &c.TargetWords, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// GetChapter truy vấn một Chương theo ID.
func (s *Store) GetChapter(chapterID int64) (*Chapter, error) {
	var c Chapter
	err := s.db.QueryRow(`SELECT id, act_id, title, position, target_words, created_at FROM chapters WHERE id = ?`, chapterID).
		Scan(&c.ID, &c.ActID, &c.Title, &c.Position, &c.TargetWords, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateChapter thêm một Chương mới vào Hồi.
func (s *Store) CreateChapter(actID int64, title string, targetWords int) (*Chapter, error) {
	if targetWords <= 0 {
		targetWords = 3000
	}
	var nextPos int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(position), 0) + 1 FROM chapters WHERE act_id = ?`, actID).Scan(&nextPos)
	res, err := s.db.Exec(`INSERT INTO chapters (act_id, title, position, target_words) VALUES (?, ?, ?, ?)`, actID, strings.TrimSpace(title), nextPos, targetWords)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Chapter{ID: id, ActID: actID, Title: title, Position: nextPos, TargetWords: targetWords, CreatedAt: time.Now()}, nil
}

// RenameChapter cập nhật tiêu đề Chương.
func (s *Store) RenameChapter(chapterID int64, newTitle string) error {
	_, err := s.db.Exec(`UPDATE chapters SET title = ? WHERE id = ?`, strings.TrimSpace(newTitle), chapterID)
	return err
}

// DeleteChapter xóa một Chương và toàn bộ các Cảnh thuộc Chương đó.
func (s *Store) DeleteChapter(chapterID int64) error {
	_, err := s.db.Exec(`DELETE FROM chapters WHERE id = ?`, chapterID)
	return err
}

// MoveChapter đổi vị trí Chương lên hoặc xuống trong cùng một Hồi.
func (s *Store) MoveChapter(chapterID int64, direction int) error {
	var actID int64
	if err := s.db.QueryRow(`SELECT act_id FROM chapters WHERE id = ?`, chapterID).Scan(&actID); err != nil {
		return err
	}
	chapters, err := s.ListChapters(actID)
	if err != nil {
		return err
	}
	idx := -1
	for i, c := range chapters {
		if c.ID == chapterID {
			idx = i
			break
		}
	}
	targetIdx := idx + direction
	if idx < 0 || targetIdx < 0 || targetIdx >= len(chapters) {
		return nil
	}
	_, err = s.db.Exec(`UPDATE chapters SET position = ? WHERE id = ?`, chapters[targetIdx].Position, chapters[idx].ID)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE chapters SET position = ? WHERE id = ?`, chapters[idx].Position, chapters[targetIdx].ID)
	return err
}

// GetChapterWordProgress trả về tổng số từ hiện tại và mục tiêu số từ của một Chương.
func (s *Store) GetChapterWordProgress(chapterID int64) (int, int, error) {
	var totalWords int
	var targetWords int
	err := s.db.QueryRow(`
		SELECT
			COALESCE((SELECT SUM(word_count) FROM scenes WHERE chapter_id = c.id), 0),
			c.target_words
		FROM chapters c
		WHERE c.id = ?
	`, chapterID).Scan(&totalWords, &targetWords)
	return totalWords, targetWords, err
}

// ListScenes trả về danh sách các Cảnh thuộc một Chương theo thứ tự vị trí.
func (s *Store) ListScenes(chapterID int64) ([]Scene, error) {
	rows, err := s.db.Query(`
		SELECT id, chapter_id, title, summary, content, side_notes, status,
		       pov_character_id, location_id, target_words, word_count, position, updated_at
		FROM scenes
		WHERE chapter_id = ?
		ORDER BY position ASC, id ASC
	`, chapterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Scene
	for rows.Next() {
		var sc Scene
		var povID, locID sql.NullInt64
		var rawStatus string
		if err := rows.Scan(
			&sc.ID, &sc.ChapterID, &sc.Title, &sc.Summary, &sc.Content, &sc.SideNotes, &rawStatus,
			&povID, &locID, &sc.TargetWords, &sc.WordCount, &sc.Position, &sc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sc.Status = NormalizeStatus(SceneStatus(rawStatus))
		if povID.Valid {
			v := povID.Int64
			sc.POVCharacterID = &v
		}
		if locID.Valid {
			v := locID.Int64
			sc.LocationID = &v
		}
		list = append(list, sc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for i := range list {
		charIDs, err := s.getSceneCharacterIDs(list[i].ID)
		if err != nil {
			return nil, err
		}
		list[i].CharacterIDs = charIDs
	}
	return list, nil
}

// GetScene tải đầy đủ thông tin một Cảnh cùng danh sách nhân vật xuất hiện.
func (s *Store) GetScene(sceneID int64) (*Scene, error) {
	var sc Scene
	var povID, locID sql.NullInt64
	var rawStatus string
	err := s.db.QueryRow(`
		SELECT id, chapter_id, title, summary, content, side_notes, status,
		       pov_character_id, location_id, target_words, word_count, position, updated_at
		FROM scenes
		WHERE id = ?
	`, sceneID).Scan(
		&sc.ID, &sc.ChapterID, &sc.Title, &sc.Summary, &sc.Content, &sc.SideNotes, &rawStatus,
		&povID, &locID, &sc.TargetWords, &sc.WordCount, &sc.Position, &sc.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	sc.Status = NormalizeStatus(SceneStatus(rawStatus))
	if povID.Valid {
		v := povID.Int64
		sc.POVCharacterID = &v
	}
	if locID.Valid {
		v := locID.Int64
		sc.LocationID = &v
	}
	charIDs, err := s.getSceneCharacterIDs(sc.ID)
	if err != nil {
		return nil, err
	}
	sc.CharacterIDs = charIDs
	return &sc, nil
}

func (s *Store) getSceneCharacterIDs(sceneID int64) ([]int64, error) {
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

// CreateScene tạo một Cảnh mới vào cuối Chương.
func (s *Store) CreateScene(chapterID int64, title string, targetWords int) (*Scene, error) {
	if targetWords <= 0 {
		targetWords = 1200
	}
	var nextPos int
	_ = s.db.QueryRow(`SELECT COALESCE(MAX(position), 0) + 1 FROM scenes WHERE chapter_id = ?`, chapterID).Scan(&nextPos)

	res, err := s.db.Exec(`
		INSERT INTO scenes (chapter_id, title, status, target_words, position)
		VALUES (?, ?, ?, ?, ?)
	`, chapterID, strings.TrimSpace(title), string(StatusIdea), targetWords, nextPos)
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
		Position:    nextPos,
		UpdatedAt:   time.Now(),
	}, nil
}

// UpdateScene lưu nội dung văn xuôi, ghi chú bên lề, số từ và thẻ siêu dữ liệu vào SQLite.
func (s *Store) UpdateScene(sc *Scene) error {
	sc.WordCount = CountWords(sc.Content)
	sc.Status = NormalizeStatus(sc.Status)
	if sc.TargetWords <= 0 {
		sc.TargetWords = 1200
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.Exec(`
		UPDATE scenes
		SET title = ?,
		    summary = ?,
		    content = ?,
		    side_notes = ?,
		    status = ?,
		    pov_character_id = ?,
		    location_id = ?,
		    target_words = ?,
		    word_count = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`,
		strings.TrimSpace(sc.Title),
		sc.Summary,
		sc.Content,
		sc.SideNotes,
		string(sc.Status),
		sc.POVCharacterID,
		sc.LocationID,
		sc.TargetWords,
		sc.WordCount,
		sc.ID,
	)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM scene_characters WHERE scene_id = ?`, sc.ID); err != nil {
		return err
	}
	for _, cid := range sc.CharacterIDs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO scene_characters (scene_id, character_id) VALUES (?, ?)`, sc.ID, cid); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// RenameScene đổi tiêu đề của Cảnh.
func (s *Store) RenameScene(sceneID int64, newTitle string) error {
	_, err := s.db.Exec(`UPDATE scenes SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, strings.TrimSpace(newTitle), sceneID)
	return err
}

// DeleteScene xóa một Cảnh khỏi Chương.
func (s *Store) DeleteScene(sceneID int64) error {
	_, err := s.db.Exec(`DELETE FROM scenes WHERE id = ?`, sceneID)
	return err
}

// MoveScene đổi thứ tự vị trí của Cảnh lên hoặc xuống trong cùng một Chương.
func (s *Store) MoveScene(sceneID int64, direction int) error {
	var chapterID int64
	if err := s.db.QueryRow(`SELECT chapter_id FROM scenes WHERE id = ?`, sceneID).Scan(&chapterID); err != nil {
		return err
	}
	scenes, err := s.ListScenes(chapterID)
	if err != nil {
		return err
	}
	idx := -1
	for i, sc := range scenes {
		if sc.ID == sceneID {
			idx = i
			break
		}
	}
	targetIdx := idx + direction
	if idx < 0 || targetIdx < 0 || targetIdx >= len(scenes) {
		return nil
	}
	_, err = s.db.Exec(`UPDATE scenes SET position = ? WHERE id = ?`, scenes[targetIdx].Position, scenes[idx].ID)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE scenes SET position = ? WHERE id = ?`, scenes[idx].Position, scenes[targetIdx].ID)
	return err
}

// ListCharacters trả về danh sách tất cả nhân vật thuộc dự án.
func (s *Store) ListCharacters(projectID int64) ([]Character, error) {
	rows, err := s.db.Query(`SELECT id, project_id, name, role, description FROM characters WHERE project_id = ? ORDER BY name ASC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Character
	for rows.Next() {
		var c Character
		if err := rows.Scan(&c.ID, &c.ProjectID, &c.Name, &c.Role, &c.Description); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// CreateCharacter thêm một nhân vật mới vào dự án.
func (s *Store) CreateCharacter(projectID int64, name, role, description string) (*Character, error) {
	res, err := s.db.Exec(`INSERT INTO characters (project_id, name, role, description) VALUES (?, ?, ?, ?)`,
		projectID, strings.TrimSpace(name), strings.TrimSpace(role), strings.TrimSpace(description))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Character{ID: id, ProjectID: projectID, Name: name, Role: role, Description: description}, nil
}

// ListLocations trả về danh sách tất cả bối cảnh / địa điểm của dự án.
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

// CreateLocation thêm một bối cảnh / địa điểm mới vào dự án.
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

// ExportManuscriptMarkdown biên dịch tuần tự toàn bộ Hồi, Chương và Cảnh sang tài liệu Markdown.
func (s *Store) ExportManuscriptMarkdown(project Project) (string, error) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\n\n", project.Title))
	if project.Author != "" {
		b.WriteString(fmt.Sprintf("**Tác giả:** %s  \n", project.Author))
	}
	if project.Genre != "" {
		b.WriteString(fmt.Sprintf("**Thể loại:** %s\n\n", project.Genre))
	}
	if project.Synopsis != "" {
		b.WriteString(fmt.Sprintf("> %s\n\n", project.Synopsis))
	}
	b.WriteString("---\n\n")

	acts, err := s.ListActs(project.ID)
	if err != nil {
		return "", err
	}
	for _, act := range acts {
		b.WriteString(fmt.Sprintf("## %s\n\n", act.Title))
		chapters, err := s.ListChapters(act.ID)
		if err != nil {
			return "", err
		}
		for _, ch := range chapters {
			b.WriteString(fmt.Sprintf("### %s\n\n", ch.Title))
			scenes, err := s.ListScenes(ch.ID)
			if err != nil {
				return "", err
			}
			for i, sc := range scenes {
				b.WriteString(fmt.Sprintf("#### %s\n\n", sc.Title))
				if strings.TrimSpace(sc.Content) != "" {
					b.WriteString(strings.TrimSpace(sc.Content) + "\n\n")
				}
				if i < len(scenes)-1 {
					b.WriteString("* * *\n\n")
				}
			}
		}
	}
	return b.String(), nil
}

// ExportManuscriptHTML biên dịch tuần tự toàn bộ Hồi, Chương và Cảnh sang trang HTML chuẩn in ấn.
func (s *Store) ExportManuscriptHTML(project Project) (string, error) {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="vi">
<head>
<meta charset="UTF-8">
<title>` + html.EscapeString(project.Title) + `</title>
<style>
  body { font-family: 'Georgia', 'Times New Roman', serif; max-width: 740px; margin: 3rem auto; padding: 0 1.5rem; color: #1C1B18; background: #FAF7F2; line-height: 1.8; }
  h1 { font-size: 2.5rem; margin-bottom: 0.25rem; }
  .meta { color: #57534E; font-style: italic; margin-bottom: 2rem; }
  h2 { margin-top: 3rem; border-bottom: 1px solid #D6D0C4; padding-bottom: 0.4rem; }
  h3 { margin-top: 2rem; color: #3F3C36; }
  h4 { margin-top: 1.5rem; color: #78716C; font-weight: normal; text-transform: uppercase; letter-spacing: 0.08em; font-size: 0.85rem; }
  p { margin: 1.1rem 0; text-indent: 1.5rem; }
  hr.scene-break { border: none; text-align: center; margin: 2rem 0; }
  hr.scene-break::after { content: "* * *"; color: #78716C; letter-spacing: 0.4em; }
</style>
</head>
<body>
`)
	b.WriteString(fmt.Sprintf("<h1>%s</h1>\n", html.EscapeString(project.Title)))
	b.WriteString(fmt.Sprintf("<div class=\"meta\">Tác giả: %s &bull; Thể loại: %s</div>\n", html.EscapeString(project.Author), html.EscapeString(project.Genre)))

	acts, err := s.ListActs(project.ID)
	if err != nil {
		return "", err
	}
	for _, act := range acts {
		b.WriteString(fmt.Sprintf("<h2>%s</h2>\n", html.EscapeString(act.Title)))
		chapters, err := s.ListChapters(act.ID)
		if err != nil {
			return "", err
		}
		for _, ch := range chapters {
			b.WriteString(fmt.Sprintf("<h3>%s</h3>\n", html.EscapeString(ch.Title)))
			scenes, err := s.ListScenes(ch.ID)
			if err != nil {
				return "", err
			}
			for i, sc := range scenes {
				b.WriteString(fmt.Sprintf("<h4>%s</h4>\n", html.EscapeString(sc.Title)))
				paragraphs := strings.Split(strings.TrimSpace(sc.Content), "\n\n")
				for _, p := range paragraphs {
					clean := strings.TrimSpace(p)
					if clean != "" {
						b.WriteString(fmt.Sprintf("<p>%s</p>\n", html.EscapeString(clean)))
					}
				}
				if i < len(scenes)-1 {
					b.WriteString("<hr class=\"scene-break\">\n")
				}
			}
		}
	}
	b.WriteString("</body>\n</html>")
	return b.String(), nil
}
