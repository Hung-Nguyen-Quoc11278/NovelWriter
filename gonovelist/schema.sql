-- Sơ đồ Cơ sở dữ liệu SQLite của GoNovelist (Phiên bản Mở rộng Thế giới & Hệ thống Thẻ Đa năng)
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

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

-- Bảng Vật phẩm (Props)
CREATE TABLE IF NOT EXISTS props (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    category TEXT NOT NULL DEFAULT 'Cổ vật',
    description TEXT NOT NULL DEFAULT '',
    significance TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (book_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Bảng Sự kiện (Events) theo dòng thời gian
CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id INTEGER NOT NULL,
    title TEXT NOT NULL,
    timeline_order INTEGER NOT NULL DEFAULT 1,
    description TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (book_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Hệ thống Thẻ đa năng có phân loại màu sắc (Universal Color-Coded Tagging System)
CREATE TABLE IF NOT EXISTS tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    book_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '#3498db',
    FOREIGN KEY (book_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Bảng ánh xạ đa hình thực thể - thẻ (character / location / prop / event)
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

-- Các bảng ánh xạ Cảnh (Scene Mapping Tables)
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

CREATE INDEX IF NOT EXISTS idx_acts_project_pos ON acts(project_id, position);
CREATE INDEX IF NOT EXISTS idx_chapters_act_pos ON chapters(act_id, position);
CREATE INDEX IF NOT EXISTS idx_scenes_chapter_pos ON scenes(chapter_id, position);
CREATE INDEX IF NOT EXISTS idx_props_book ON props(book_id);
CREATE INDEX IF NOT EXISTS idx_events_book_order ON events(book_id, timeline_order);
CREATE INDEX IF NOT EXISTS idx_tags_book ON tags(book_id);
CREATE INDEX IF NOT EXISTS idx_entity_tags_lookup ON entity_tags(entity_type, entity_id);
