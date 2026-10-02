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

// Migrate thực thi các câu lệnh DDL để tạo và nâng cấp cấu trúc bảng (bao gồm Props, Events, Tags).
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

	CREATE TABLE IF NOT EXISTS props (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		book_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT 'Cổ vật',
		description TEXT NOT NULL DEFAULT '',
		significance TEXT NOT NULL DEFAULT '',
		FOREIGN KEY (book_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		book_id INTEGER NOT NULL,
		title TEXT NOT NULL,
		timeline_order INTEGER NOT NULL DEFAULT 1,
		description TEXT NOT NULL DEFAULT '',
		FOREIGN KEY (book_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		book_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		color TEXT NOT NULL DEFAULT '#3498db',
		FOREIGN KEY (book_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS entity_tags (
		entity_type TEXT NOT NULL,
		entity_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (entity_type, entity_id, tag_id),
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
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

	CREATE TABLE IF NOT EXISTS scene_props (
		scene_id INTEGER NOT NULL,
		prop_id INTEGER NOT NULL,
		PRIMARY KEY (scene_id, prop_id),
		FOREIGN KEY (scene_id) REFERENCES scenes(id) ON DELETE CASCADE,
		FOREIGN KEY (prop_id) REFERENCES props(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS scene_events (
		scene_id INTEGER NOT NULL,
		event_id INTEGER NOT NULL,
		PRIMARY KEY (scene_id, event_id),
		FOREIGN KEY (scene_id) REFERENCES scenes(id) ON DELETE CASCADE,
		FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE
	);
	`
	if _, err := s.db.Exec(ddl); err != nil {
		return err
	}

	// Migration an toàn cho các cơ sở dữ liệu cũ chưa có cột color trong bảng tags
	if err := s.ensureTagColorColumn(); err != nil {
		return err
	}
	return nil
}

// ensureTagColorColumn kiểm tra bảng tags hiện có và tự động thêm cột color TEXT DEFAULT '#3498db' nếu chưa tồn tại.
func (s *Store) ensureTagColorColumn() error {
	rows, err := s.db.Query(`PRAGMA table_info(tags)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasColorCol := false
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err == nil {
			if strings.EqualFold(name, "color") {
				hasColorCol = true
				break
			}
		}
	}

	if !hasColorCol {
		if _, err := s.db.Exec(`ALTER TABLE tags ADD COLUMN color TEXT DEFAULT '#3498db'`); err != nil {
			return fmt.Errorf("không thể nâng cấp cột color cho bảng tags: %w", err)
		}
	}

	_, _ = s.db.Exec(`UPDATE tags SET color = '#3498db' WHERE color IS NULL OR TRIM(color) = ''`)
	return nil
}

// SeedDefaultProjectIfEmpty tạo một dự án mẫu tiếng Việt nếu cơ sở dữ liệu trống,
// hoặc bổ sung Vật phẩm / Sự kiện / Thẻ mẫu cho dự án hiện có nếu các bảng mới đang trống.
func (s *Store) SeedDefaultProjectIfEmpty() error {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		_, err := s.SeedVietnameseSampleProject()
		return err
	}

	// Nếu DB cũ đã có project nhưng chưa có dữ liệu mẫu cho bảng tags/props/events thì tự động khởi tạo
	var tagCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM tags`).Scan(&tagCount); err == nil && tagCount == 0 {
		projects, _ := s.ListProjects()
		if len(projects) > 0 {
			_ = s.seedWorldBuildingExtrasForProject(projects[0].ID)
		}
	}
	return nil
}

func (s *Store) seedWorldBuildingExtrasForProject(bookID int64) error {
	tagCoVat, err := s.CreateTag(bookID, "Cổ vật", "#f39c12")
	if err != nil {
		return err
	}
	tagKhuVucCam, err := s.CreateTag(bookID, "Khu vực cấm", "#e74c3c")
	if err != nil {
		return err
	}
	tagThienGioi, err := s.CreateTag(bookID, "Thiên giới", "#9b59b6")
	if err != nil {
		return err
	}
	tagHoangGia, err := s.CreateTag(bookID, "Bí mật triều đình", "#3498db")
	if err != nil {
		return err
	}

	prop1, err := s.CreateProp(
		bookID,
		"Thấu Kính Hải Đăng Cổ",
		"Báu vật quang học",
		"Phiến pha lê thế kỷ XVII có vết rạn ẩn chứa hải đồ ra đảo sương mù.",
		"Chìa khóa duy nhất giải mã luồng lạch qua rạn đá ngầm Nam Hải.",
	)
	if err == nil {
		_ = s.SetEntityTags(EntityProp, prop1.ID, []int64{tagCoVat.ID, tagHoangGia.ID})
	}

	prop2, err := s.CreateProp(
		bookID,
		"La Bàn Đồng Khắc Chữ Chu Sa",
		"Khí cụ hàng hải",
		"Chiếc la bàn đồng cổ của thuyền trưởng Trần Đình Bách có kim chỉ hướng lệch theo từ trường đảo ngầm.",
		"Giúp định vị phương vị Sao Khuê trong đêm sương.",
	)
	if err == nil {
		_ = s.SetEntityTags(EntityProp, prop2.ID, []int64{tagCoVat.ID})
	}

	ev1, err := s.CreateEvent(
		bookID,
		"Vụ Đắm Đội Thương Thuyền Năm Cảnh Hưng",
		1,
		"Đội thuyền buôn chở cặp thấu kính song sinh đột ngột mất tích giữa vùng biển sương mù 80 năm trước.",
	)
	if err == nil {
		_ = s.SetEntityTags(EntityEvent, ev1.ID, []int64{tagHoangGia.ID, tagKhuVucCam.ID})
	}

	ev2, err := s.CreateEvent(
		bookID,
		"Đêm Thủy Triều Đỏ Rằm Tháng Tám",
		2,
		"Thời điểm duy nhất trong năm khi rạn đá ngầm hạ thấp, mở lối vào ngọn hải đăng cổ.",
	)
	if err == nil {
		_ = s.SetEntityTags(EntityEvent, ev2.ID, []int64{tagThienGioi.ID, tagKhuVucCam.ID})
	}

	// Gắn thẻ mẫu cho nhân vật và địa điểm đầu tiên (nếu có)
	chars, _ := s.ListCharacters(bookID)
	if len(chars) > 0 {
		_ = s.SetEntityTags(EntityCharacter, chars[0].ID, []int64{tagHoangGia.ID})
	}
	locs, _ := s.ListLocations(bookID)
	if len(locs) > 0 {
		_ = s.SetEntityTags(EntityLocation, locs[0].ID, []int64{tagKhuVucCam.ID})
	}
	return nil
}

// SeedVietnameseSampleProject khởi tạo một tác phẩm mẫu tiếng Việt hoàn chỉnh kèm Nhân vật, Địa điểm, Vật phẩm, Sự kiện và Thẻ.
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

	tagCoVat, _ := s.CreateTag(proj.ID, "Cổ vật", "#f39c12")
	tagKhuVucCam, _ := s.CreateTag(proj.ID, "Khu vực cấm", "#e74c3c")
	tagThienGioi, _ := s.CreateTag(proj.ID, "Thiên giới", "#9b59b6")
	tagHoangGia, _ := s.CreateTag(proj.ID, "Bí mật triều đình", "#3498db")

	elena, err := s.CreateCharacter(proj.ID, "Lê Ngọc Liên", "Nhân vật chính", "Nghệ nhân chế tác và phục chế thấu kính tại phố cổ Hội An.")
	if err != nil {
		return nil, err
	}
	_ = s.SetEntityTags(EntityCharacter, elena.ID, []int64{tagCoVat.ID, tagHoangGia.ID})

	julian, err := s.CreateCharacter(proj.ID, "Trần Đình Bách", "Đồng hành", "Thuyền trưởng tàu buôn từng đi qua vùng biển sương mù Nam Hải.")
	if err != nil {
		return nil, err
	}
	_ = s.SetEntityTags(EntityCharacter, julian.ID, []int64{tagKhuVucCam.ID})

	archivist, err := s.CreateCharacter(proj.ID, "Cụ Thủ Từ Họ Phạm", "Người dẫn đường", "Người trông coi kho thư tịch cổ tại hội quán.")
	if err != nil {
		return nil, err
	}
	_ = s.SetEntityTags(EntityCharacter, archivist.ID, []int64{tagHoangGia.ID, tagThienGioi.ID})

	observatory, err := s.CreateLocation(proj.ID, "Xưởng Thủy Tinh Phố Cổ", "Căn gác gỗ nhìn ra sông Thu Bồn với những lò nung pha lê và bàn mài thấu kính.")
	if err != nil {
		return nil, err
	}
	_ = s.SetEntityTags(EntityLocation, observatory.ID, []int64{tagCoVat.ID})

	vault, err := s.CreateLocation(proj.ID, "Thư Các Chùa Cầu", "Căn phòng lưu trữ bản đồ hàng hải và nhật ký thương thuyền trăm năm.")
	if err != nil {
		return nil, err
	}
	_ = s.SetEntityTags(EntityLocation, vault.ID, []int64{tagKhuVucCam.ID, tagHoangGia.ID})

	propLens, err := s.CreateProp(
		proj.ID,
		"Thấu Kính Hải Đăng Cổ",
		"Báu vật quang học",
		"Phiến pha lê thế kỷ XVII có vết rạn ẩn chứa hải đồ ra đảo sương mù.",
		"Chìa khóa duy nhất giải mã luồng lạch qua rạn đá ngầm Nam Hải.",
	)
	if err != nil {
		return nil, err
	}
	_ = s.SetEntityTags(EntityProp, propLens.ID, []int64{tagCoVat.ID, tagThienGioi.ID})

	propCompass, err := s.CreateProp(
		proj.ID,
		"La Bàn Đồng Khắc Chữ Chu Sa",
		"Khí cụ hàng hải",
		"Chiếc la bàn đồng cổ của thuyền trưởng Trần Đình Bách.",
		"Định vị tọa độ đảo ẩn khi kết hợp với chùm sáng từ thấu kính.",
	)
	if err != nil {
		return nil, err
	}
	_ = s.SetEntityTags(EntityProp, propCompass.ID, []int64{tagCoVat.ID})

	evShipwreck, err := s.CreateEvent(
		proj.ID,
		"Vụ Mất Tích Đội Thương Thuyền 80 Năm Trước",
		1,
		"Đội thuyền chở cặp thấu kính song sinh biến mất trong màn sương ngoài khơi Cù Lao Chàm.",
	)
	if err != nil {
		return nil, err
	}
	_ = s.SetEntityTags(EntityEvent, evShipwreck.ID, []int64{tagHoangGia.ID, tagKhuVucCam.ID})

	evEquinox, err := s.CreateEvent(
		proj.ID,
		"Đêm Thủy Triều Thấp Rằm Tháng Tám",
		2,
		"Thời khắc rạn đá ngầm lộ diện và ngọn hải đăng cổ phát tín hiệu.",
	)
	if err != nil {
		return nil, err
	}
	_ = s.SetEntityTags(EntityEvent, evEquinox.ID, []int64{tagThienGioi.ID})

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
	sc1.SideNotes = "Ghi chú: Nhấn mạnh chi tiết mùi dầu thông và tiếng chuông chùa xa xăm."
	sc1.Status = StatusCompleted
	sc1.POVCharacterID = &elena.ID
	sc1.LocationID = &observatory.ID
	sc1.CharacterIDs = []int64{elena.ID, julian.ID}
	sc1.PropIDs = []int64{propLens.ID, propCompass.ID}
	sc1.EventIDs = []int64{evShipwreck.ID}
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
	sc2.PropIDs = []int64{propLens.ID}
	sc2.EventIDs = []int64{evShipwreck.ID, evEquinox.ID}
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
	sc3.PropIDs = []int64{propCompass.ID}
	sc3.EventIDs = []int64{evEquinox.ID}
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
	sc4.PropIDs = []int64{propLens.ID, propCompass.ID}
	sc4.EventIDs = []int64{evEquinox.ID}
	if err := s.UpdateScene(sc4); err != nil {
		return nil, err
	}

	return proj, nil
}

// ==================== PROJECT / ACT / CHAPTER / SCENE CRUD ====================

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

func (s *Store) RenameAct(actID int64, newTitle string) error {
	_, err := s.db.Exec(`UPDATE acts SET title = ? WHERE id = ?`, strings.TrimSpace(newTitle), actID)
	return err
}

func (s *Store) DeleteAct(actID int64) error {
	_, err := s.db.Exec(`DELETE FROM acts WHERE id = ?`, actID)
	return err
}

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

func (s *Store) GetChapter(chapterID int64) (*Chapter, error) {
	var c Chapter
	err := s.db.QueryRow(`SELECT id, act_id, title, position, target_words, created_at FROM chapters WHERE id = ?`, chapterID).
		Scan(&c.ID, &c.ActID, &c.Title, &c.Position, &c.TargetWords, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

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

func (s *Store) RenameChapter(chapterID int64, newTitle string) error {
	_, err := s.db.Exec(`UPDATE chapters SET title = ? WHERE id = ?`, strings.TrimSpace(newTitle), chapterID)
	return err
}

func (s *Store) DeleteChapter(chapterID int64) error {
	_, err := s.db.Exec(`DELETE FROM chapters WHERE id = ?`, chapterID)
	return err
}

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
		list[i].CharacterIDs, _ = s.getSceneCharacterIDs(list[i].ID)
		list[i].PropIDs, _ = s.getScenePropIDs(list[i].ID)
		list[i].EventIDs, _ = s.getSceneEventIDs(list[i].ID)
	}
	return list, nil
}

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
	sc.CharacterIDs, _ = s.getSceneCharacterIDs(sc.ID)
	sc.PropIDs, _ = s.getScenePropIDs(sc.ID)
	sc.EventIDs, _ = s.getSceneEventIDs(sc.ID)
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
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

func (s *Store) getScenePropIDs(sceneID int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT prop_id FROM scene_props WHERE scene_id = ? ORDER BY prop_id ASC`, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

func (s *Store) getSceneEventIDs(sceneID int64) ([]int64, error) {
	rows, err := s.db.Query(`SELECT event_id FROM scene_events WHERE scene_id = ? ORDER BY event_id ASC`, sceneID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

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

// UpdateScene lưu nội dung văn xuôi, ghi chú bên lề, số từ và các bảng ánh xạ (Nhân vật, Vật phẩm, Sự kiện) vào SQLite.
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

	// Cập nhật ánh xạ Nhân vật trong cảnh
	if _, err := tx.Exec(`DELETE FROM scene_characters WHERE scene_id = ?`, sc.ID); err != nil {
		return err
	}
	for _, cid := range sc.CharacterIDs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO scene_characters (scene_id, character_id) VALUES (?, ?)`, sc.ID, cid); err != nil {
			return err
		}
	}

	// Cập nhật ánh xạ Vật phẩm trong cảnh (scene_props)
	if _, err := tx.Exec(`DELETE FROM scene_props WHERE scene_id = ?`, sc.ID); err != nil {
		return err
	}
	for _, pid := range sc.PropIDs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO scene_props (scene_id, prop_id) VALUES (?, ?)`, sc.ID, pid); err != nil {
			return err
		}
	}

	// Cập nhật ánh xạ Sự kiện trong cảnh (scene_events)
	if _, err := tx.Exec(`DELETE FROM scene_events WHERE scene_id = ?`, sc.ID); err != nil {
		return err
	}
	for _, eid := range sc.EventIDs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO scene_events (scene_id, event_id) VALUES (?, ?)`, sc.ID, eid); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) RenameScene(sceneID int64, newTitle string) error {
	_, err := s.db.Exec(`UPDATE scenes SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, strings.TrimSpace(newTitle), sceneID)
	return err
}

func (s *Store) DeleteScene(sceneID int64) error {
	_, err := s.db.Exec(`DELETE FROM scenes WHERE id = ?`, sceneID)
	return err
}

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

// ==================== UNIVERSAL TAGGING SYSTEM (tags & entity_tags) ====================

// ListTags trả về danh sách tất cả các Thẻ (kèm mã màu Hex) của một cuốn sách (book_id).
func (s *Store) ListTags(bookID int64) ([]Tag, error) {
	rows, err := s.db.Query(`SELECT id, book_id, name, COALESCE(color, '#3498db') FROM tags WHERE book_id = ? ORDER BY name ASC`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.BookID, &t.Name, &t.Color); err != nil {
			return nil, err
		}
		t.Color = NormalizeHexColor(t.Color)
		list = append(list, t)
	}
	return list, rows.Err()
}

// CreateTag tạo một Thẻ mới với mã màu tùy chọn (mặc định '#3498db' nếu không truyền màu).
func (s *Store) CreateTag(bookID int64, name string, optionalColor ...string) (*Tag, error) {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil, fmt.Errorf("tên thẻ không được để trống")
	}
	hexColor := DefaultTagColor
	if len(optionalColor) > 0 {
		hexColor = NormalizeHexColor(optionalColor[0])
	}

	existing, _ := s.ListTags(bookID)
	for _, t := range existing {
		if strings.EqualFold(t.Name, clean) {
			if len(optionalColor) > 0 && t.Color != hexColor {
				_ = s.UpdateTag(t.ID, clean, hexColor)
				t.Color = hexColor
			}
			return &t, nil
		}
	}
	res, err := s.db.Exec(`INSERT INTO tags (book_id, name, color) VALUES (?, ?, ?)`, bookID, clean, hexColor)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Tag{ID: id, BookID: bookID, Name: clean, Color: hexColor}, nil
}

// UpdateTag cập nhật tên và mã màu Hex của một Thẻ đã tồn tại.
func (s *Store) UpdateTag(tagID int64, name string, color string) error {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return fmt.Errorf("tên thẻ không được để trống")
	}
	hexColor := NormalizeHexColor(color)
	_, err := s.db.Exec(`UPDATE tags SET name = ?, color = ? WHERE id = ?`, clean, hexColor, tagID)
	return err
}

// DeleteTag xóa một Thẻ và toàn bộ liên kết trong entity_tags.
func (s *Store) DeleteTag(tagID int64) error {
	_, err := s.db.Exec(`DELETE FROM tags WHERE id = ?`, tagID)
	return err
}

// GetEntityTags trả về danh sách các Thẻ (kèm mã màu Hex) được gắn cho một thực thể cụ thể (character/location/prop/event).
func (s *Store) GetEntityTags(entityType EntityType, entityID int64) ([]Tag, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.book_id, t.name, COALESCE(t.color, '#3498db')
		FROM tags t
		INNER JOIN entity_tags et ON et.tag_id = t.id
		WHERE et.entity_type = ? AND et.entity_id = ?
		ORDER BY t.name ASC
	`, string(entityType), entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.BookID, &t.Name, &t.Color); err != nil {
			return nil, err
		}
		t.Color = NormalizeHexColor(t.Color)
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// SetEntityTags cập nhật danh sách ID thẻ được gắn cho một thực thể trong bảng entity_tags.
func (s *Store) SetEntityTags(entityType EntityType, entityID int64, tagIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM entity_tags WHERE entity_type = ? AND entity_id = ?`, string(entityType), entityID); err != nil {
		return err
	}
	for _, tid := range tagIDs {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO entity_tags (entity_type, entity_id, tag_id) VALUES (?, ?, ?)`,
			string(entityType), entityID, tid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ==================== CHARACTERS CRUD + TAGS ====================

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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Tags, _ = s.GetEntityTags(EntityCharacter, list[i].ID)
	}
	return list, nil
}

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

func (s *Store) UpdateCharacter(c Character, tagIDs []int64) error {
	_, err := s.db.Exec(`UPDATE characters SET name = ?, role = ?, description = ? WHERE id = ?`,
		strings.TrimSpace(c.Name), strings.TrimSpace(c.Role), strings.TrimSpace(c.Description), c.ID)
	if err != nil {
		return err
	}
	return s.SetEntityTags(EntityCharacter, c.ID, tagIDs)
}

func (s *Store) DeleteCharacter(charID int64) error {
	_ = s.SetEntityTags(EntityCharacter, charID, nil)
	_, err := s.db.Exec(`DELETE FROM characters WHERE id = ?`, charID)
	return err
}

// ==================== LOCATIONS CRUD + TAGS ====================

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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Tags, _ = s.GetEntityTags(EntityLocation, list[i].ID)
	}
	return list, nil
}

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

func (s *Store) UpdateLocation(l Location, tagIDs []int64) error {
	_, err := s.db.Exec(`UPDATE locations SET name = ?, description = ? WHERE id = ?`,
		strings.TrimSpace(l.Name), strings.TrimSpace(l.Description), l.ID)
	if err != nil {
		return err
	}
	return s.SetEntityTags(EntityLocation, l.ID, tagIDs)
}

func (s *Store) DeleteLocation(locID int64) error {
	_ = s.SetEntityTags(EntityLocation, locID, nil)
	_, err := s.db.Exec(`DELETE FROM locations WHERE id = ?`, locID)
	return err
}

// ==================== PROPS (VẬT PHẨM) CRUD + TAGS ====================

func (s *Store) ListProps(bookID int64) ([]Prop, error) {
	rows, err := s.db.Query(`SELECT id, book_id, name, category, description, significance FROM props WHERE book_id = ? ORDER BY name ASC`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Prop
	for rows.Next() {
		var p Prop
		if err := rows.Scan(&p.ID, &p.BookID, &p.Name, &p.Category, &p.Description, &p.Significance); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Tags, _ = s.GetEntityTags(EntityProp, list[i].ID)
	}
	return list, nil
}

func (s *Store) CreateProp(bookID int64, name, category, description, significance string) (*Prop, error) {
	if strings.TrimSpace(category) == "" {
		category = "Cổ vật"
	}
	res, err := s.db.Exec(`
		INSERT INTO props (book_id, name, category, description, significance)
		VALUES (?, ?, ?, ?, ?)
	`, bookID, strings.TrimSpace(name), strings.TrimSpace(category), strings.TrimSpace(description), strings.TrimSpace(significance))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Prop{
		ID:           id,
		BookID:       bookID,
		Name:         name,
		Category:     category,
		Description:  description,
		Significance: significance,
	}, nil
}

func (s *Store) UpdateProp(p Prop, tagIDs []int64) error {
	_, err := s.db.Exec(`
		UPDATE props
		SET name = ?, category = ?, description = ?, significance = ?
		WHERE id = ?
	`, strings.TrimSpace(p.Name), strings.TrimSpace(p.Category), strings.TrimSpace(p.Description), strings.TrimSpace(p.Significance), p.ID)
	if err != nil {
		return err
	}
	return s.SetEntityTags(EntityProp, p.ID, tagIDs)
}

func (s *Store) DeleteProp(propID int64) error {
	_ = s.SetEntityTags(EntityProp, propID, nil)
	_, err := s.db.Exec(`DELETE FROM props WHERE id = ?`, propID)
	return err
}

// ==================== EVENTS (SỰ KIỆN DÒNG THỜI GIAN) CRUD + TAGS ====================

func (s *Store) ListEvents(bookID int64) ([]Event, error) {
	rows, err := s.db.Query(`SELECT id, book_id, title, timeline_order, description FROM events WHERE book_id = ? ORDER BY timeline_order ASC, id ASC`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Event
	for rows.Next() {
		var ev Event
		if err := rows.Scan(&ev.ID, &ev.BookID, &ev.Title, &ev.TimelineOrder, &ev.Description); err != nil {
			return nil, err
		}
		list = append(list, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Tags, _ = s.GetEntityTags(EntityEvent, list[i].ID)
	}
	return list, nil
}

func (s *Store) CreateEvent(bookID int64, title string, timelineOrder int, description string) (*Event, error) {
	if timelineOrder <= 0 {
		_ = s.db.QueryRow(`SELECT COALESCE(MAX(timeline_order), 0) + 1 FROM events WHERE book_id = ?`, bookID).Scan(&timelineOrder)
	}
	res, err := s.db.Exec(`
		INSERT INTO events (book_id, title, timeline_order, description)
		VALUES (?, ?, ?, ?)
	`, bookID, strings.TrimSpace(title), timelineOrder, strings.TrimSpace(description))
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &Event{
		ID:            id,
		BookID:        bookID,
		Title:         title,
		TimelineOrder: timelineOrder,
		Description:   description,
	}, nil
}

func (s *Store) UpdateEvent(ev Event, tagIDs []int64) error {
	_, err := s.db.Exec(`
		UPDATE events
		SET title = ?, timeline_order = ?, description = ?
		WHERE id = ?
	`, strings.TrimSpace(ev.Title), ev.TimelineOrder, strings.TrimSpace(ev.Description), ev.ID)
	if err != nil {
		return err
	}
	return s.SetEntityTags(EntityEvent, ev.ID, tagIDs)
}

func (s *Store) DeleteEvent(eventID int64) error {
	_ = s.SetEntityTags(EntityEvent, eventID, nil)
	_, err := s.db.Exec(`DELETE FROM events WHERE id = ?`, eventID)
	return err
}

// ==================== MANUSCRIPT EXPORTERS ====================

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
