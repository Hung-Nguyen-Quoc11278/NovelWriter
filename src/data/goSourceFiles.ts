import mainGoRaw from '../../gonovelist/main.go?raw';
import modelsGoRaw from '../../gonovelist/models.go?raw';
import databaseGoRaw from '../../gonovelist/database.go?raw';
import uiMainGoRaw from '../../gonovelist/ui_main.go?raw';
import uiEditorGoRaw from '../../gonovelist/ui_editor.go?raw';
import schemaSqlRaw from '../../gonovelist/schema.sql?raw';
import goModRaw from '../../gonovelist/go.mod?raw';

export interface GoSourceFile {
  filename: string;
  path: string;
  layer:
    | 'Điểm Khởi Chạy (Entry Point)'
    | 'Tầng Mô Hình (Domain Layer)'
    | 'Tầng Dữ Liệu & Dịch Vụ (SQLite)'
    | 'Tầng Giao Diện (Fyne v2 UI)'
    | 'Lược Đồ CSDL (SQLite DDL)'
    | 'Cấu Hình Module';
  summary: string;
  code: string;
}

export const PROJECT_TREE_LAYOUT = `gonovelist/
├── go.mod          # Định nghĩa module Go 1.22+ (fyne.io/fyne/v2 v2.5.3, modernc.org/sqlite)
├── schema.sql      # Lược đồ cơ sở dữ liệu quan hệ SQLite (ràng buộc khóa ngoại & chế độ WAL)
├── main.go         # Điểm khởi chạy ứng dụng, khởi tạo thư mục dữ liệu & cơ sở dữ liệu
├── models.go       # Các struct miền dữ liệu (Project, Act, Chapter, Scene, Character, Location) & bộ đếm từ Unicode
├── database.go     # Tầng truy xuất SQLite, tự động tạo bảng, dữ liệu mẫu tiếng Việt & xuất Markdown/HTML
├── ui_main.go      # Cửa sổ chính Fyne v2, cây phân cấp Hồi -> Chương -> Cảnh, thanh trình đơn & hộp thoại
└── ui_editor.go    # Trình soạn thảo văn xuôi, tự động lưu chống dội (750ms debounce), thanh tiến độ & siêu dữ liệu`;

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
      'Khởi tạo ứng dụng Fyne v2 tiếng Việt, mở cơ sở dữ liệu SQLite tại ~/.gonovelist/gonovelist.db và đảm bảo lưu bản thảo trước khi đóng cửa sổ.',
    code: mainGoRaw,
  },
  {
    filename: 'models.go',
    path: 'gonovelist/models.go',
    layer: 'Tầng Mô Hình (Domain Layer)',
    summary:
      'Định nghĩa các thực thể cốt lõi (Project, Act, Chapter, Scene, Character, Location), chuẩn hóa trạng thái tiếng Việt và hàm đếm từ hỗ trợ đầy đủ tiếng Việt Unicode.',
    code: modelsGoRaw,
  },
  {
    filename: 'database.go',
    path: 'gonovelist/database.go',
    layer: 'Tầng Dữ Liệu & Dịch Vụ (SQLite)',
    summary:
      'Quản lý lưu trữ SQLite, tự động khởi tạo bảng, tạo tác phẩm mẫu tiếng Việt ("Bản Đồ Thủy Tinh Thành Hội An"), sắp xếp thứ tự Hồi/Chương/Cảnh và xuất bản thảo Markdown/HTML.',
    code: databaseGoRaw,
  },
  {
    filename: 'ui_main.go',
    path: 'gonovelist/ui_main.go',
    layer: 'Tầng Giao Diện (Fyne v2 UI)',
    summary:
      'Điều phối bố cục cửa sổ chính tiếng Việt, cây phân cấp Hồi/Chương/Cảnh (widget.Tree), thanh trình đơn, tạo tác phẩm mẫu tiếng Việt và xuất bản thảo.',
    code: uiMainGoRaw,
  },
  {
    filename: 'ui_editor.go',
    path: 'gonovelist/ui_editor.go',
    layer: 'Tầng Giao Diện (Fyne v2 UI)',
    summary:
      'Khung soạn thảo văn xuôi tiếng Việt, bộ tự động lưu chống dội 750ms, thanh tiến độ từ theo Cảnh/Chương và bảng Ngữ cảnh (Trạng thái, POV, Bối cảnh, Nhân vật, Ghi chú).',
    code: uiEditorGoRaw,
  },
  {
    filename: 'schema.sql',
    path: 'gonovelist/schema.sql',
    layer: 'Lược Đồ CSDL (SQLite DDL)',
    summary:
      'Định nghĩa các bảng SQLite với khóa ngoại xóa dây chuyền (ON DELETE CASCADE) cho Tác phẩm, Hồi, Chương, Cảnh, Nhân vật và Bối cảnh.',
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
