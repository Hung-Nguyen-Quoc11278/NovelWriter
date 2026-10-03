# GoNovelist

GoNovelist là ứng dụng desktop viết tiểu thuyết bằng tiếng Việt, xây dựng với Go, Fyne v2 và SQLite. Ứng dụng quản lý bản thảo theo cấu trúc **Tác phẩm → Hồi → Chương → Cảnh**, có trình soạn thảo, quản lý thế giới nhân vật/địa điểm/vật phẩm/sự kiện và xuất bản sang nhiều định dạng.

> Hướng dẫn này dành cho ứng dụng desktop trong thư mục [`gonovelist/`](gonovelist/). Thư mục `src/` ở root là giao diện minh họa riêng của repository.

## Tính năng

- Soạn thảo cảnh, tóm tắt, tiêu đề và ghi chú; tự động lưu nội dung vào SQLite.
- Quản lý nhân vật, địa điểm, vật phẩm, sự kiện và thẻ theo từng danh mục.
- Nhập tiếng Việt qua IME hệ thống như Fcitx5/IBus; có thể bật bộ gõ Telex nội bộ trong Cài đặt.
- Điều chỉnh cỡ chữ bằng nút `A-`, `A+`, `Mặc định` hoặc phím tắt `Ctrl` + `-`, `Ctrl` + `+`, `Ctrl` + `0`.
- Chọn chủ đề Sáng, Tối hoặc Sepia.
- Xuất bản thảo thành `.txt`, `.md`, `.html`, `.odt`, `.docx`, `.pdf` hoặc `.epub`.
- Xuất audio MP3 bằng Edge-TTS, chọn phạm vi và giọng đọc tiếng Việt.

## Yêu cầu

- Go 1.22 trở lên.
- Trên Linux: trình biên dịch C, `pkg-config`, thư viện phát triển OpenGL và X11 cần cho Fyne/GLFW. Ví dụ trên Ubuntu/Debian:

```sh
sudo apt install build-essential pkg-config libgl1-mesa-dev xorg-dev
```

Tên gói có thể khác trên các bản phân phối Linux khác. macOS cần Xcode Command Line Tools; Windows cần toolchain C tương thích với Fyne.

## Chạy ứng dụng

Mở terminal tại thư mục Go:

```sh
cd gonovelist
go mod download
go run .
```

Biên dịch ứng dụng:

```sh
go build -o GoNovelist .
```

Trên Windows, lệnh build tạo `GoNovelist.exe`:

```powershell
go build -o GoNovelist.exe .
```

Ứng dụng tự tạo thư mục dữ liệu `~/.gonovelist/` trên Linux/macOS hoặc thư mục người dùng tương ứng trên Windows. Cơ sở dữ liệu chính là `gonovelist.db` trong thư mục này. Hãy đóng ứng dụng trước khi sao lưu cơ sở dữ liệu.

## Bắt đầu viết

1. Khởi chạy GoNovelist. Nếu chưa có tác phẩm, chọn **Tệp → Tạo tác phẩm mẫu Tiếng Việt** để nạp dữ liệu minh họa, hoặc **Tệp → Tác phẩm mới...** để tạo tác phẩm.
2. Dùng các nút **+ Hồi**, **+ Chương**, **+ Cảnh** hoặc menu **Cấu trúc** để xây dựng dàn ý.
3. Chọn một cảnh trong cây bên trái rồi nhập bản thảo ở vùng soạn thảo trung tâm. Tiêu đề, tóm tắt và nội dung được tự động lưu.
4. Dùng thanh công cụ để định dạng đoạn văn, chèn trích dẫn, tiêu đề phụ hoặc dấu ngắt cảnh. Nút **Xem trước Định dạng** chuyển đổi giữa bản thảo thô và bản xem trước.
5. Mở **Thế giới & Thẻ → Mở Trung Tâm Thế Giới...** để quản lý nhân vật, địa điểm, vật phẩm, sự kiện và thẻ.
6. Dùng menu **Chế độ xem** để bật/tắt chế độ tập trung hoặc thay đổi cỡ chữ. Có thể kéo các thanh chia để điều chỉnh độ rộng cây tác phẩm, trình soạn thảo và bảng ngữ cảnh.

## Xuất bản thảo

Chọn một lệnh trong menu **Tệp**, hoặc dùng **Xuất bản ra...**. Hộp thoại cho phép chọn định dạng, phạm vi, có đính kèm tóm tắt hay không, có xuất tiêu đề cảnh hay không và đường dẫn lưu.

Các phạm vi hiện có:

- Toàn bộ tác phẩm.
- Một Hồi được chọn.
- Chương/Cảnh đang chọn trên cây.

Định dạng Word `.docx` là gói Office Open XML, có cấu trúc tiêu đề và giữ định dạng đậm, nghiêng, gạch chân, trích dẫn, tiêu đề phụ và ngắt cảnh. `.odt` phù hợp với LibreOffice; `.pdf` nhúng phông chữ TrueType để hỗ trợ tiếng Việt.

## Xuất audio MP3

Chọn **Tệp → Xuất bản ra Audio MP3 (Edge-TTS)...** hoặc nút **Xuất Audio** trong trình soạn thảo. Chọn phạm vi, giọng đọc, tốc độ, âm lượng, xem/chỉnh nội dung và đường dẫn MP3. Tác vụ chạy nền và hiển thị tiến độ theo từng phần.

Hiện Edge-TTS trả về hai giọng tiếng Việt được ứng dụng hỗ trợ:

- Hoài Mỹ, nữ miền Nam (`vi-VN-HoaiMyNeural`).
- Nam Minh, nam miền Nam (`vi-VN-NamMinhNeural`).

Edge-TTS cần kết nối Internet. Nếu không tìm thấy công cụ, ứng dụng thử lần lượt `edge-tts` trong `PATH`, cấu hình/runtime được chỉ định, runtime tại `assets/python` hoặc thư mục cấu hình, rồi Python hệ thống có module `edge_tts`.

### Thiết lập cho môi trường phát triển

Các chức năng soạn thảo/xuất bản không cần Python. Nếu muốn thử xuất audio khi chạy từ source, tạo môi trường riêng rồi trỏ GoNovelist tới môi trường đó:

```sh
python3 -m venv "$HOME/.gonovelist-python"
"$HOME/.gonovelist-python/bin/python" -m pip install edge-tts
export GONOVELIST_PYTHON_HOME="$HOME/.gonovelist-python"
go run .
```

Hoặc cung cấp trực tiếp đường dẫn lệnh `edge-tts` đã cài sẵn:

```sh
export GONOVELIST_EDGE_TTS="/đường/dẫn/đến/edge-tts"
go run .
```

Runtime Python được phát triển cục bộ không được Git theo dõi. Khi phát hành bản cài đặt thực sự độc lập, đơn vị đóng gói cần đưa runtime Python và gói `edge-tts` phù hợp hệ điều hành/kiến trúc vào `assets/python` cạnh ứng dụng; nếu không, người dùng cần cung cấp một trong các phương án hệ thống nêu trên.

## Kiểm thử

Từ thư mục `gonovelist/`:

```sh
go test ./...
go test -race ./...
go vet ./...
```

Có thể chạy kiểm tra tổng hợp thật qua dịch vụ Edge-TTS (cần Internet và runtime đã cấu hình):

```sh
GONOVELIST_RUN_EDGE_TTS_INTEGRATION=1 go test -run '^TestEdgeTTSSynthesisIntegration$' -count=1
```

## Giấy phép (License)
Dự án này được phát hành dưới các điều khoản của giấy phép [GNU General Public License v3.0](LICENSE).
