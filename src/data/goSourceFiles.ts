import mainGoRaw from '../../gonovelist/main.go?raw';
import vietInputGoRaw from '../../gonovelist/vietnamese_input.go?raw';
import modelsGoRaw from '../../gonovelist/models.go?raw';
import databaseGoRaw from '../../gonovelist/database.go?raw';
import uiWorldBuildingGoRaw from '../../gonovelist/ui_worldbuilding.go?raw';
import uiMainGoRaw from '../../gonovelist/ui_main.go?raw';
import uiEditorGoRaw from '../../gonovelist/ui_editor.go?raw';
import schemaSqlRaw from '../../gonovelist/schema.sql?raw';
import goModRaw from '../../gonovelist/go.mod?raw';

export interface GoSourceFile {
  filename: string;
  path: string;
  layer:
    | 'Điểm Khởi Chạy (Entry Point)'
    | 'Bộ Gõ & Phông Chữ Tiếng Việt'
    | 'Tầng Mô Hình (Domain Layer)'
    | 'Tầng Dữ Liệu & Dịch Vụ (SQLite)'
    | 'Tầng Giao Diện (Fyne v2 UI)'
    | 'Lược Đồ CSDL (SQLite DDL)'
    | 'Cấu Hình Module';
  summary: string;
  code: string;
}

export const PROJECT_TREE_LAYOUT = `gonovelist/
├── go.mod                 # Định nghĩa module Go 1.22+ (fyne.io/fyne/v2 v2.5.3, modernc.org/sqlite)
├── schema.sql             # Lược đồ SQLite (projects, acts, chapters, scenes, characters, locations, props, events, tags, entity_tags)
├── main.go                # Điểm khởi chạy ứng dụng, tự động nạp phông chữ Tiếng Việt & khởi tạo SQLite
├── vietnamese_input.go    # Bộ gõ Tiếng Việt Telex tích hợp cho Fyne widget.Entry & tự động dò tìm phông chữ Unicode
├── models.go              # Các struct miền dữ liệu (Project, Act, Chapter, Scene, Character, Location, Prop, Event, Tag)
├── database.go            # Tầng truy xuất SQLite, миграции tự động, quản lý Thẻ đa hình (entity_tags) & xuất bản thảo
├── ui_worldbuilding.go    # Trung tâm Xây dựng Thế giới đa tab (Nhân vật, Địa điểm, Vật phẩm, Sự kiện, Quản lý Thẻ & Lọc theo thẻ)
├── ui_main.go             # Cửa sổ chính Fyne v2, cây phân cấp Hồi -> Chương -> Cảnh, nút mở Trung tâm Thế giới & bật/tắt Telex
└── ui_editor.go           # Trình soạn thảo văn xuôi Tiếng Việt, auto-save 750ms & thanh bên Ngữ cảnh Cảnh (Nhân vật, Vật phẩm, Sự kiện)`;

export const BUILD_COMMANDS = `# 1. Tạo thư mục dự án và khởi tạo module Go (Yêu cầu Go 1.22+)
mkdir -p gonovelist && cd gonovelist
go mod init gonovelist

# 2. Cài đặt Fyne v2 và trình điều khiển SQLite thuần Go
go get fyne.io/fyne/v2@v2.5.3
go get modernc.org/sqlite@v1.34.4
go mod tidy

# 3. Chạy trực tiếp ứng dụng GoNovelist giao diện Tiếng Việt
go run .

# 4. Biên dịch thành tệp thực thi Desktop gốc (Native Binary)
# Trên Linux / macOS:
go build -ldflags="-s -w" -o gonovelist .

# Trên Windows (ẩn cửa sổ dòng lệnh console):
go build -ldflags="-s -w -H=windowsgui" -o GoNovelist.exe .`;

export const GO_SOURCE_FILES: GoSourceFile[] = [
  {
    filename: 'main.go',
    path: 'gonovelist/main.go',
    layer: 'Điểm Khởi Chạy (Entry Point)',
    summary:
      'Tự động cấu hình phông chữ Unicode Tiếng Việt, khởi tạo ứng dụng Fyne v2 và mở cơ sở dữ liệu SQLite tại ~/.gonovelist/gonovelist.db.',
    code: mainGoRaw,
  },
  {
    filename: 'vietnamese_input.go',
    path: 'gonovelist/vietnamese_input.go',
    layer: 'Bộ Gõ & Phông Chữ Tiếng Việt',
    summary:
      'Bộ gõ Tiếng Việt Telex tích hợp trực tiếp vào widget.Entry của Fyne (hỗ trợ aa->â, aw->ă, dd->đ, ee->ê, oo->ô, ow->ơ, uw->ư và 5 dấu s/f/r/x/j/z) kèm tự động nạp phông chữ hệ thống.',
    code: vietInputGoRaw,
  },
  {
    filename: 'models.go',
    path: 'gonovelist/models.go',
    layer: 'Tầng Mô Hình (Domain Layer)',
    summary:
      'Định nghĩa các thực thể cốt lõi và mở rộng: Character, Location, Prop (Vật phẩm), Event (Sự kiện), Tag (Thẻ đa năng), EntityType và Scene (PropIDs, EventIDs).',
    code: modelsGoRaw,
  },
  {
    filename: 'database.go',
    path: 'gonovelist/database.go',
    layer: 'Tầng Dữ Liệu & Dịch Vụ (SQLite)',
    summary:
      'Cập nhật migration SQLite cho bảng props, events, tags, entity_tags, scene_props, scene_events; cung cấp CRUD đầy đủ và dữ liệu mẫu Tiếng Việt.',
    code: databaseGoRaw,
  },
  {
    filename: 'ui_worldbuilding.go',
    path: 'gonovelist/ui_worldbuilding.go',
    layer: 'Tầng Giao Diện (Fyne v2 UI)',
    summary:
      'Trung tâm Xây dựng Thế giới đa tab (container.NewAppTabs): Nhân vật, Địa điểm, Vật phẩm, Sự kiện và Quản lý Thẻ kèm bộ lọc thông minh theo Thẻ.',
    code: uiWorldBuildingGoRaw,
  },
  {
    filename: 'ui_editor.go',
    path: 'gonovelist/ui_editor.go',
    layer: 'Tầng Giao Diện (Fyne v2 UI)',
    summary:
      'Khung soạn thảo văn xuôi Tiếng Việt và thanh bên Ngữ cảnh Cảnh mở rộng (chọn Nhân vật, Vật phẩm trong cảnh, Sự kiện trong cảnh lưu thẳng vào SQLite).',
    code: uiEditorGoRaw,
  },
  {
    filename: 'ui_main.go',
    path: 'gonovelist/ui_main.go',
    layer: 'Tầng Giao Diện (Fyne v2 UI)',
    summary:
      'Cửa sổ chính Fyne v2 tích hợp nút "Quản lý Thế giới & Thẻ", công tắc bật/tắt bộ gõ Tiếng Việt Telex và cây phân cấp Hồi -> Chương -> Cảnh.',
    code: uiMainGoRaw,
  },
  {
    filename: 'schema.sql',
    path: 'gonovelist/schema.sql',
    layer: 'Lược Đồ CSDL (SQLite DDL)',
    summary:
      'Lược đồ SQLite hoàn chỉnh bao gồm các bảng mới: props, events, tags, entity_tags, scene_props và scene_events.',
    code: schemaSqlRaw,
  },
  {
    filename: 'go.mod',
    path: 'gonovelist/go.mod',
    layer: 'Cấu Hình Module',
    summary:
      'Khai báo module Go 1.22+ sử dụng Fyne v2.5.3 và trình điều khiển SQLite modernc.org/sqlite.',
    code: goModRaw,
  },
];
