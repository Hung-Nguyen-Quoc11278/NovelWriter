package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// VietnameseVoicePreset định nghĩa thông số một mẫu giọng đọc Neural Tiếng Việt trong Edge-TTS.
type VietnameseVoicePreset struct {
	ID          string // Mã định danh kỹ thuật (VD: "vi-VN-HoaiMyNeural")
	Label       string // Tên hiển thị Tiếng Việt trên giao diện (VD: "Hoài Mỹ (Nữ - Miền Nam)")
	Gender      string // Giới tính (Nữ / Nam)
	Description string // Mô tả đặc trưng giọng đọc
}

// Danh sách các giọng đọc Tiếng Việt chất lượng cao của Edge-TTS
var VietnameseVoicePresets = []VietnameseVoicePreset{
	{
		ID:          "vi-VN-HoaiMyNeural",
		Label:       "Hoài Mỹ (Nữ - Miền Nam)",
		Gender:      "Nữ",
		Description: "Giọng nữ miền Nam dịu dàng, tự nhiên, giàu cảm xúc — rất thích hợp cho tiểu thuyết tình cảm & tự sự.",
	},
	{
		ID:          "vi-VN-NamMinhNeural",
		Label:       "Nam Minh (Nam - Miền Nam)",
		Gender:      "Nam",
		Description: "Giọng nam miền Nam ấm áp, truyền cảm, rõ chữ — rất thích hợp cho tiểu thuyết lịch sử & phiêu lưu.",
	},
	{
		ID:          "vi-VN-HoangMaiNeural",
		Label:       "Hoàng Mai (Nữ - Miền Bắc)",
		Gender:      "Nữ",
		Description: "Giọng nữ Hà Nội chuẩn mực, thanh lịch, phát âm tròn vành rõ chữ — rất thích hợp cho truyện văn học kinh điển & ký sự.",
	},
	{
		ID:          "vi-VN-NamKhanhNeural",
		Label:       "Nam Khánh (Nam - Miền Bắc)",
		Gender:      "Nam",
		Description: "Giọng nam miền Bắc đĩnh đạc, đầm ấm, quyền uy và cuốn hút — thích hợp cho truyện kỳ ảo, hành động & trinh thám.",
	},
	{
		ID:          "vi-VN-ThuTrangNeural",
		Label:       "Thu Trang (Nữ - Miền Trung)",
		Gender:      "Nữ",
		Description: "Giọng nữ miền Trung nhẹ nhàng, mộc mạc, tha thiết đậm chất thơ — thích hợp cho truyện đồng quê, hồi ức & chiêm nghiệm.",
	},
}

// AudioExportScope định nghĩa phạm vi phân cấp nội dung cần xuất sang âm thanh.
type AudioExportScope string

const (
	AudioScopeCurrentScene   AudioExportScope = "Cảnh hiện tại"
	AudioScopeCurrentChapter AudioExportScope = "Chương hiện tại"
	AudioScopeCurrentAct     AudioExportScope = "Hồi hiện tại"
	AudioScopeFullNovel      AudioExportScope = "Toàn bộ tác phẩm"
)

// AllAudioExportScopes trả về danh sách các phạm vi xuất bản audio cho widget.Select.
var AllAudioExportScopes = []string{
	string(AudioScopeCurrentScene),
	string(AudioScopeCurrentChapter),
	string(AudioScopeCurrentAct),
	string(AudioScopeFullNovel),
}

// AllVietnameseVoiceLabels trả về danh sách các nhãn hiển thị cho widget.Select.
func AllVietnameseVoiceLabels() []string {
	labels := make([]string, len(VietnameseVoicePresets))
	for i, v := range VietnameseVoicePresets {
		labels[i] = v.Label
	}
	return labels
}

// ResolveVoiceIDFromLabel ánh xạ từ nhãn Tiếng Việt sang Voice ID của Edge-TTS.
func ResolveVoiceIDFromLabel(label string) string {
	for _, v := range VietnameseVoicePresets {
		if v.Label == label || v.ID == label {
			return v.ID
		}
	}
	return "vi-VN-HoaiMyNeural"
}

// ResolveVoiceDescriptionFromLabel trả về mô tả của giọng đọc đang chọn.
func ResolveVoiceDescriptionFromLabel(label string) string {
	for _, v := range VietnameseVoicePresets {
		if v.Label == label || v.ID == label {
			return v.Description
		}
	}
	return ""
}

// CleanProseForSpeech loại bỏ các ký tự định dạng thô (Markdown, thẻ HTML, dấu ngắt cảnh)
// để Edge-TTS đọc tự nhiên và liền mạch như một người kể chuyện thực thụ.
func CleanProseForSpeech(raw string) string {
	text := raw
	// Xóa thẻ HTML nếu có
	reHTML := regexp.MustCompile(`<[^>]*>`)
	text = reHTML.ReplaceAllString(text, "")

	// Xóa định dạng in đậm / in nghiêng markdown: **text** hoặc *text*
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "_", "")

	// Thay thế các dòng ngắt cảnh * * * hoặc ― bằng dấu ngắt câu nhẹ
	reDivider := regexp.MustCompile(`(?m)^\s*(\*\s*\*\s*\*|―+|--+)\s*$`)
	text = reDivider.ReplaceAllString(text, "\n\n")

	// Xóa ký tự trích dẫn > ở đầu dòng
	reQuote := regexp.MustCompile(`(?m)^\s*>\s*`)
	text = reQuote.ReplaceAllString(text, "")

	// Rút gọn các dòng trống liên tiếp
	reMultiLines := regexp.MustCompile(`\n{3,}`)
	text = reMultiLines.ReplaceAllString(text, "\n\n")

	return strings.TrimSpace(text)
}

// ==================== PURE GO EDGE-TTS WEBSOCKET CLIENT ====================

const (
	edgeTTSHost      = "speech.platform.bing.com"
	edgeTTSPort      = "443"
	edgeTTSToken     = "6A5AA1D4EAFF4E9FB37E23D68491D6F4"
	edgeTTSUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0"
	edgeTTSOrigin    = "chrome-extension://jdiccldimpdaibmpdkjnbmckianbfold"

	// Các mã Opcode WebSocket chuẩn theo RFC 6455
	wsOpcodeContinuation byte = 0x0
	wsOpcodeText         byte = 0x1
	wsOpcodeBinary       byte = 0x2
	wsOpcodeClose        byte = 0x8
	wsOpcodePing         byte = 0x9
	wsOpcodePong         byte = 0xA
)

func writeWSFrame(conn net.Conn, opcode byte, payload []byte) error {
	var header []byte
	b0 := byte(0x80) | (opcode & 0x0F) // FIN = 1
	n := len(payload)
	if n <= 125 {
		header = []byte{b0, 0x80 | byte(n)}
	} else if n <= 65535 {
		header = make([]byte, 4)
		header[0] = b0
		header[1] = 0x80 | 126
		binary.BigEndian.PutUint16(header[2:4], uint16(n))
	} else {
		header = make([]byte, 10)
		header[0] = b0
		header[1] = 0x80 | 127
		binary.BigEndian.PutUint64(header[2:10], uint64(n))
	}

	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return fmt.Errorf("lỗi sinh khóa ngẫu nhiên che dữ liệu WebSocket: %w", err)
	}
	header = append(header, mask[:]...)

	maskedPayload := make([]byte, n)
	for i := 0; i < n; i++ {
		maskedPayload[i] = payload[i] ^ mask[i%4]
	}

	if _, err := conn.Write(header); err != nil {
		return err
	}
	if n > 0 {
		_, err := conn.Write(maskedPayload)
		return err
	}
	return nil
}

func readWSFrame(reader *bufio.Reader) (byte, []byte, error) {
	b0, err := reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	opcode := b0 & 0x0F

	b1, err := reader.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	isMasked := (b1 & 0x80) != 0
	len7 := int(b1 & 0x7F)

	var payloadLen uint64
	if len7 <= 125 {
		payloadLen = uint64(len7)
	} else if len7 == 126 {
		var l uint16
		if err := binary.Read(reader, binary.BigEndian, &l); err != nil {
			return 0, nil, err
		}
		payloadLen = uint64(l)
	} else if len7 == 127 {
		var l uint64
		if err := binary.Read(reader, binary.BigEndian, &l); err != nil {
			return 0, nil, err
		}
		payloadLen = l
	}

	// Giới hạn an toàn chống tràn bộ nhớ với gói tin lỗi
	if payloadLen > 10*1024*1024 {
		return 0, nil, fmt.Errorf("kích thước khung dữ liệu WebSocket vượt quá 10MB: %d bytes", payloadLen)
	}

	var mask [4]byte
	if isMasked {
		if _, err := io.ReadFull(reader, mask[:]); err != nil {
			return 0, nil, err
		}
	}

	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(reader, payload); err != nil {
			return 0, nil, err
		}
		if isMasked {
			for i := 0; i < int(payloadLen); i++ {
				payload[i] ^= mask[i%4]
			}
		}
	}

	return opcode, payload, nil
}

func sendWSClose(conn net.Conn) error {
	// Gửi mã đóng 1000 (Normal Closure)
	payload := []byte{0x03, 0xE8}
	return writeWSFrame(conn, wsOpcodeClose, payload)
}

func xmlEscape(s string) string {
	var buf strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			buf.WriteString("&amp;")
		case '<':
			buf.WriteString("&lt;")
		case '>':
			buf.WriteString("&gt;")
		case '"':
			buf.WriteString("&quot;")
		case '\'':
			buf.WriteString("&apos;")
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

func randomHex(byteLen int) string {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%032x", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}

func edgeTimestamp() string {
	return time.Now().UTC().Format("Mon Jan 02 2006 15:04:05") + " GMT+0000 (Coordinated Universal Time)"
}

// SplitTextIntoTTSChunks phân đoạn văn bản theo ranh giới câu và đoạn văn để tránh vượt giới hạn kích thước SSML của Edge-TTS.
func SplitTextIntoTTSChunks(text string, maxRunes int) []string {
	if maxRunes <= 0 {
		maxRunes = 2500
	}
	clean := strings.TrimSpace(text)
	if clean == "" {
		return nil
	}
	if utf8.RuneCountInString(clean) <= maxRunes {
		return []string{clean}
	}

	paragraphs := strings.Split(clean, "\n")
	var chunks []string
	var currentChunk strings.Builder

	for _, p := range paragraphs {
		trimmedP := strings.TrimSpace(p)
		if trimmedP == "" {
			continue
		}

		pRunes := utf8.RuneCountInString(trimmedP)
		curRunes := utf8.RuneCountInString(currentChunk.String())

		if pRunes > maxRunes {
			sentences := splitParagraphIntoSentences(trimmedP)
			for _, s := range sentences {
				sRunes := utf8.RuneCountInString(s)
				if curRunes+sRunes+1 > maxRunes && curRunes > 0 {
					chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
					currentChunk.Reset()
					curRunes = 0
				}
				if currentChunk.Len() > 0 {
					currentChunk.WriteString(" ")
				}
				currentChunk.WriteString(s)
				curRunes += sRunes + 1
			}
			continue
		}

		if curRunes+pRunes+2 > maxRunes && curRunes > 0 {
			chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
			currentChunk.Reset()
		}

		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(trimmedP)
	}

	if currentChunk.Len() > 0 {
		trimmed := strings.TrimSpace(currentChunk.String())
		if trimmed != "" {
			chunks = append(chunks, trimmed)
		}
	}

	return chunks
}

func splitParagraphIntoSentences(p string) []string {
	re := regexp.MustCompile(`[^.!?…]+[.!?…]*`)
	matches := re.FindAllString(p, -1)
	if len(matches) == 0 {
		return []string{p}
	}
	var res []string
	for _, m := range matches {
		t := strings.TrimSpace(m)
		if t != "" {
			res = append(res, t)
		}
	}
	return res
}

// SynthesizeSpeechChunk gửi một phân đoạn văn bản qua WebSocket Edge-TTS và trả về mảng byte MP3 hoàn chỉnh.
func SynthesizeSpeechChunk(voiceID, chunkText string) ([]byte, error) {
	if strings.TrimSpace(chunkText) == "" {
		return nil, nil
	}
	if voiceID == "" {
		voiceID = "vi-VN-HoaiMyNeural"
	}

	connID := randomHex(16)
	reqPath := fmt.Sprintf("/consumer/speech/synthesize/readaloud/edge/v1?TrustedClientToken=%s&ConnectionId=%s", edgeTTSToken, connID)

	dialer := &net.Dialer{
		Timeout: 15 * time.Second,
	}
	tlsConfig := &tls.Config{
		ServerName: edgeTTSHost,
	}

	rawConn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(edgeTTSHost, edgeTTSPort), tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("lỗi kết nối mạng: không thể thiết lập kết nối an toàn TLS tới máy chủ Edge-TTS: %w", err)
	}
	defer rawConn.Close()

	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("lỗi sinh khóa ngẫu nhiên Sec-WebSocket-Key: %w", err)
	}
	secKey := base64.StdEncoding.EncodeToString(keyBytes)

	reqStr := fmt.Sprintf(
		"GET %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Key: %s\r\n"+
			"Sec-WebSocket-Version: 13\r\n"+
			"Origin: %s\r\n"+
			"Pragma: no-cache\r\n"+
			"Cache-Control: no-cache\r\n"+
			"User-Agent: %s\r\n"+
			"Accept-Encoding: gzip, deflate, br, zstd\r\n"+
			"Accept-Language: vi-VN,vi;q=0.9,en-US;q=0.8,en;q=0.7\r\n\r\n",
		reqPath, edgeTTSHost, secKey, edgeTTSOrigin, edgeTTSUserAgent,
	)

	_ = rawConn.SetDeadline(time.Now().Add(15 * time.Second))
	if _, err := rawConn.Write([]byte(reqStr)); err != nil {
		return nil, fmt.Errorf("lỗi gửi bản tin bắt tay WebSocket: %w", err)
	}

	reader := bufio.NewReader(rawConn)
	u, _ := url.Parse(reqPath)
	resp, err := http.ReadResponse(reader, &http.Request{Method: "GET", URL: u})
	if err != nil {
		return nil, fmt.Errorf("lỗi đọc phản hồi bắt tay từ máy chủ giọng đọc Edge-TTS: %w", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return nil, fmt.Errorf("máy chủ Edge-TTS từ chối kết nối WebSocket (Mã lỗi HTTP %d %s)", resp.StatusCode, resp.Status)
	}

	// 1. Gửi cấu hình giọng đọc (speech.config)
	configMsg := fmt.Sprintf(
		"X-Timestamp:%s\r\n"+
			"Content-Type:application/json; charset=utf-8\r\n"+
			"Path:speech.config\r\n\r\n"+
			`{"context":{"synthesis":{"audio":{"metadataoptions":{"sentenceBoundaryEnabled":"false","wordBoundaryEnabled":"false"},"outputFormat":"audio-24khz-48kbitrate-mono-mp3"}}}}`,
		edgeTimestamp(),
	)
	if err := writeWSFrame(rawConn, wsOpcodeText, []byte(configMsg)); err != nil {
		return nil, fmt.Errorf("lỗi gửi bản tin cấu hình giọng đọc Edge-TTS: %w", err)
	}

	// 2. Gửi văn bản SSML
	reqID := randomHex(16)
	ssmlPayload := fmt.Sprintf(
		"<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='vi-VN'>"+
			"<voice name='%s'>"+
			"<prosody pitch='+0Hz' rate='+0%%' volume='+0%%'>%s</prosody>"+
			"</voice></speak>",
		xmlEscape(voiceID),
		xmlEscape(chunkText),
	)
	ssmlMsg := fmt.Sprintf(
		"X-RequestId:%s\r\n"+
			"Content-Type:application/ssml+xml\r\n"+
			"X-Timestamp:%s\r\n"+
			"Path:ssml\r\n\r\n"+
			"%s",
		reqID,
		edgeTimestamp(),
		ssmlPayload,
	)
	if err := writeWSFrame(rawConn, wsOpcodeText, []byte(ssmlMsg)); err != nil {
		return nil, fmt.Errorf("lỗi gửi bản tin SSML tới máy chủ giọng đọc: %w", err)
	}

	// 3. Lắng nghe và trích xuất các gói dữ liệu MP3 nhị phân
	var audioBuf bytes.Buffer
	for {
		_ = rawConn.SetReadDeadline(time.Now().Add(40 * time.Second))
		opcode, payload, err := readWSFrame(reader)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return nil, fmt.Errorf("quá thời gian chờ phản hồi từ máy chủ giọng đọc Edge-TTS (Timeout)")
			}
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("lỗi khi nhận luồng âm thanh WebSocket: %w", err)
		}

		switch opcode {
		case wsOpcodeText:
			textStr := string(payload)
			if strings.Contains(textStr, "Path:turn.end") {
				// Hoàn thành tổng hợp cho phân đoạn này
				_ = sendWSClose(rawConn)
				return audioBuf.Bytes(), nil
			}
		case wsOpcodeBinary:
			if len(payload) >= 2 {
				hLen := int(binary.BigEndian.Uint16(payload[0:2]))
				if len(payload) > 2+hLen {
					audioChunk := payload[2+hLen:]
					audioBuf.Write(audioChunk)
				}
			}
		case wsOpcodePing:
			_ = writeWSFrame(rawConn, wsOpcodePong, payload)
		case wsOpcodeClose:
			_ = sendWSClose(rawConn)
			return audioBuf.Bytes(), nil
		}
	}

	_ = sendWSClose(rawConn)
	if audioBuf.Len() == 0 {
		return nil, fmt.Errorf("máy chủ giọng đọc không trả về dữ liệu âm thanh")
	}
	return audioBuf.Bytes(), nil
}

// ExportTextToAudioWithProgress chuyển đổi văn bản sang định dạng âm thanh MP3 thuần túy bằng Go qua WebSocket,
// hỗ trợ chia nhỏ các đoạn văn dài (chunking) và báo cáo tiến trình liên tục cho giao diện Fyne.
func ExportTextToAudioWithProgress(voiceID, textContent, outputPath string, onProgress func(current, total int, msg string)) error {
	cleanText := strings.TrimSpace(textContent)
	if cleanText == "" {
		return fmt.Errorf("nội dung văn bản để xuất âm thanh không được để trống")
	}
	if voiceID == "" {
		voiceID = "vi-VN-HoaiMyNeural"
	}
	if outputPath == "" {
		return fmt.Errorf("đường dẫn tệp âm thanh đích không được để trống")
	}

	if !strings.HasSuffix(strings.ToLower(outputPath), ".mp3") {
		outputPath += ".mp3"
	}

	outDir := filepath.Dir(outputPath)
	if outDir != "" && outDir != "." {
		_ = os.MkdirAll(outDir, 0755)
	}

	// Chia văn bản thành các phân đoạn an toàn theo giới hạn ký tự của Edge-TTS (~2500 ký tự)
	chunks := SplitTextIntoTTSChunks(cleanText, 2500)
	if len(chunks) == 0 {
		return fmt.Errorf("không tìm thấy đoạn văn bản hợp lệ để xuất âm thanh")
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("không thể tạo tệp âm thanh đích '%s': %w", outputPath, err)
	}
	defer outFile.Close()

	totalChunks := len(chunks)
	for i, chunk := range chunks {
		partNum := i + 1
		if onProgress != nil {
			onProgress(partNum, totalChunks, fmt.Sprintf("Đang kết nối dịch vụ giọng đọc và xử lý phần %d/%d...", partNum, totalChunks))
		}

		var audioBytes []byte
		var synthErr error
		// Cơ chế tự động thử lại 1 lần nếu gặp lỗi mạng chập chờn
		for attempt := 1; attempt <= 2; attempt++ {
			audioBytes, synthErr = SynthesizeSpeechChunk(voiceID, chunk)
			if synthErr == nil && len(audioBytes) > 0 {
				break
			}
			time.Sleep(300 * time.Millisecond)
		}

		if synthErr != nil {
			return fmt.Errorf("lỗi khi tổng hợp âm thanh phần %d/%d: %w", partNum, totalChunks, synthErr)
		}
		if len(audioBytes) == 0 {
			return fmt.Errorf("phần %d/%d không có dữ liệu âm thanh trả về", partNum, totalChunks)
		}

		if _, err := outFile.Write(audioBytes); err != nil {
			return fmt.Errorf("lỗi khi ghi luồng âm thanh vào tệp: %w", err)
		}

		if partNum < totalChunks {
			time.Sleep(100 * time.Millisecond)
		}
	}

	_ = outFile.Sync()
	fi, err := outFile.Stat()
	if err != nil || fi.Size() == 0 {
		return fmt.Errorf("tệp âm thanh chưa được tạo hoặc dung lượng bằng 0 byte")
	}

	if onProgress != nil {
		onProgress(totalChunks, totalChunks, "Đã hoàn thành xuất file âm thanh!")
	}
	return nil
}

// ExportTextToAudioWithEdgeTTS là hàm tương thích ngược, thực hiện xuất âm thanh thuần Go qua WebSocket.
func ExportTextToAudioWithEdgeTTS(voiceID, textContent, outputPath string) error {
	return ExportTextToAudioWithProgress(voiceID, textContent, outputPath, nil)
}

// resolveCurrentSceneChapterAct xác định ngữ cảnh thực tế của Cảnh, Chương và Hồi đang được chọn.
func (ui *NovelistUI) resolveCurrentSceneChapterAct() (*Scene, *Chapter, *Act) {
	var curScene *Scene
	var curChapter *Chapter
	var curAct *Act

	// 1. Thử lấy từ editorPanel nếu đang mở cảnh
	if ui.editorPanel != nil && ui.editorPanel.activeScene != nil {
		curScene = ui.editorPanel.activeScene
		if curScene.ChapterID > 0 {
			curChapter, _ = ui.store.GetChapter(curScene.ChapterID)
			if curChapter != nil && curChapter.ActID > 0 {
				curAct, _ = ui.store.GetAct(curChapter.ActID)
			}
		}
	}

	// 2. Nếu chưa có cảnh hoặc đang chọn nút khác trên cây phân cấp
	if curScene == nil && ui.selectedUID != "" {
		kind, id, err := ParseNodeUID(ui.selectedUID)
		if err == nil {
			switch kind {
			case "scene":
				curScene, _ = ui.store.GetScene(id)
				if curScene != nil && curScene.ChapterID > 0 {
					curChapter, _ = ui.store.GetChapter(curScene.ChapterID)
					if curChapter != nil && curChapter.ActID > 0 {
						curAct, _ = ui.store.GetAct(curChapter.ActID)
					}
				}
			case "chapter":
				curChapter, _ = ui.store.GetChapter(id)
				if curChapter != nil {
					if curChapter.ActID > 0 {
						curAct, _ = ui.store.GetAct(curChapter.ActID)
					}
					scenes, _ := ui.store.ListScenes(curChapter.ID)
					if len(scenes) > 0 {
						curScene = &scenes[0]
					}
				}
			case "act":
				curAct, _ = ui.store.GetAct(id)
				if curAct != nil {
					chaps, _ := ui.store.ListChapters(curAct.ID)
					if len(chaps) > 0 {
						curChapter = &chaps[0]
						scenes, _ := ui.store.ListScenes(curChapter.ID)
						if len(scenes) > 0 {
							curScene = &scenes[0]
						}
					}
				}
			}
		}
	}

	// 3. Dự phòng nếu chưa có gì được chọn: lấy Hồi, Chương và Cảnh đầu tiên của tác phẩm
	if curAct == nil {
		allActs, _ := ui.store.ListActs(ui.activeProject.ID)
		if len(allActs) > 0 {
			curAct = &allActs[0]
			chaps, _ := ui.store.ListChapters(curAct.ID)
			if len(chaps) > 0 {
				curChapter = &chaps[0]
				scenes, _ := ui.store.ListScenes(curChapter.ID)
				if len(scenes) > 0 {
					curScene = &scenes[0]
				}
			}
		}
	}

	return curScene, curChapter, curAct
}

// CompileAudioTextForScope duyệt cơ sở dữ liệu SQLite theo phân cấp Hồi -> Chương -> Cảnh
// và tổng hợp nội dung văn bản sạch phù hợp cho giọng đọc Edge-TTS.
func (ui *NovelistUI) CompileAudioTextForScope(scope AudioExportScope) (compiledText string, scopeTitle string, totalScenes int, summaryInfo string) {
	curScene, curChapter, curAct := ui.resolveCurrentSceneChapterAct()
	var sb strings.Builder

	switch scope {
	case AudioScopeCurrentScene:
		scTitle := "Cảnh"
		scContent := ""
		if curScene != nil {
			scTitle = curScene.Title
			scContent = curScene.Content
			if ui.editorPanel != nil && ui.editorPanel.activeScene != nil && ui.editorPanel.activeScene.ID == curScene.ID {
				if ui.editorPanel.proseEntry != nil && strings.TrimSpace(ui.editorPanel.proseEntry.Text) != "" {
					scContent = ui.editorPanel.proseEntry.Text
				}
			}
		}
		sb.WriteString(scTitle + ".\n\n")
		sb.WriteString(CleanProseForSpeech(scContent))
		scopeTitle = fmt.Sprintf("%s_%s", ui.activeProject.Title, scTitle)
		summaryInfo = fmt.Sprintf("Cảnh đơn: %s", scTitle)
		return sb.String(), scopeTitle, 1, summaryInfo

	case AudioScopeCurrentChapter:
		if curChapter == nil {
			return ui.CompileAudioTextForScope(AudioScopeFullNovel)
		}
		scopeTitle = fmt.Sprintf("%s_%s", ui.activeProject.Title, curChapter.Title)
		sb.WriteString(curChapter.Title + ".\n\n")

		scenes, _ := ui.store.ListScenes(curChapter.ID)
		count := 0
		for _, sc := range scenes {
			text := sc.Content
			if ui.editorPanel != nil && ui.editorPanel.activeScene != nil && ui.editorPanel.activeScene.ID == sc.ID {
				if ui.editorPanel.proseEntry != nil && strings.TrimSpace(ui.editorPanel.proseEntry.Text) != "" {
					text = ui.editorPanel.proseEntry.Text
				}
			}
			clean := CleanProseForSpeech(text)
			if clean == "" {
				continue
			}
			count++
			if sc.Title != "" {
				sb.WriteString(sc.Title + ".\n\n")
			}
			sb.WriteString(clean)
			sb.WriteString("\n\n---\n\n")
		}
		summaryInfo = fmt.Sprintf("Chương: %s (%d cảnh)", curChapter.Title, count)
		return sb.String(), scopeTitle, count, summaryInfo

	case AudioScopeCurrentAct:
		if curAct == nil {
			return ui.CompileAudioTextForScope(AudioScopeFullNovel)
		}
		scopeTitle = fmt.Sprintf("%s_%s", ui.activeProject.Title, curAct.Title)
		sb.WriteString(curAct.Title + ".\n\n")

		chaps, _ := ui.store.ListChapters(curAct.ID)
		count := 0
		for _, ch := range chaps {
			sb.WriteString(ch.Title + ".\n\n")
			scenes, _ := ui.store.ListScenes(ch.ID)
			for _, sc := range scenes {
				text := sc.Content
				if ui.editorPanel != nil && ui.editorPanel.activeScene != nil && ui.editorPanel.activeScene.ID == sc.ID {
					if ui.editorPanel.proseEntry != nil && strings.TrimSpace(ui.editorPanel.proseEntry.Text) != "" {
						text = ui.editorPanel.proseEntry.Text
					}
				}
				clean := CleanProseForSpeech(text)
				if clean == "" {
					continue
				}
				count++
				if sc.Title != "" {
					sb.WriteString(sc.Title + ".\n\n")
				}
				sb.WriteString(clean)
				sb.WriteString("\n\n---\n\n")
			}
		}
		summaryInfo = fmt.Sprintf("Hồi: %s (%d chương, %d cảnh)", curAct.Title, len(chaps), count)
		return sb.String(), scopeTitle, count, summaryInfo

	case AudioScopeFullNovel:
		fallthrough
	default:
		scopeTitle = fmt.Sprintf("%s_Toan_Bo_Tac_Pham", ui.activeProject.Title)
		sb.WriteString(ui.activeProject.Title + ".\n")
		if ui.activeProject.Author != "" {
			sb.WriteString("Tác giả: " + ui.activeProject.Author + ".\n")
		}
		if strings.TrimSpace(ui.activeProject.Synopsis) != "" {
			sb.WriteString("Tóm tắt tác phẩm:\n" + CleanProseForSpeech(ui.activeProject.Synopsis) + "\n\n")
		}
		sb.WriteString("\n====================\n\n")

		allActs, _ := ui.store.ListActs(ui.activeProject.ID)
		count := 0
		for _, act := range allActs {
			sb.WriteString(act.Title + ".\n\n")
			chaps, _ := ui.store.ListChapters(act.ID)
			for _, ch := range chaps {
				sb.WriteString(ch.Title + ".\n\n")
				scenes, _ := ui.store.ListScenes(ch.ID)
				for _, sc := range scenes {
					text := sc.Content
					if ui.editorPanel != nil && ui.editorPanel.activeScene != nil && ui.editorPanel.activeScene.ID == sc.ID {
						if ui.editorPanel.proseEntry != nil && strings.TrimSpace(ui.editorPanel.proseEntry.Text) != "" {
							text = ui.editorPanel.proseEntry.Text
						}
					}
					clean := CleanProseForSpeech(text)
					if clean == "" {
						continue
					}
					count++
					if sc.Title != "" {
						sb.WriteString(sc.Title + ".\n\n")
					}
					sb.WriteString(clean)
					sb.WriteString("\n\n---\n\n")
				}
			}
		}
		summaryInfo = fmt.Sprintf("Toàn bộ tác phẩm: %s (%d hồi, %d cảnh)", ui.activeProject.Title, len(allActs), count)
		return sb.String(), scopeTitle, count, summaryInfo
	}
}

// ShowAudioExportDialog mở hộp thoại trực quan hỗ trợ đa giọng đọc Neural Tiếng Việt
// và lựa chọn phạm vi phân cấp linh hoạt (Cảnh, Chương, Hồi, Toàn bộ tác phẩm) với WebSocket thuần Go.
func (ui *NovelistUI) ShowAudioExportDialog() {
	if ui.window == nil {
		return
	}

	// Xác định phạm vi ban đầu: ưu tiên Cảnh hiện tại nếu đang mở, ngược lại chọn Toàn bộ tác phẩm
	initialScope := AudioScopeCurrentScene
	if ui.editorPanel == nil || ui.editorPanel.activeScene == nil {
		initialScope = AudioScopeFullNovel
	}

	initialContent, initialTitle, _, initialSummary := ui.CompileAudioTextForScope(initialScope)

	// 1. Selector Phạm vi xuất bản (Audio Export Scope)
	scopeSelect := widget.NewSelect(AllAudioExportScopes, nil)
	scopeSelect.SetSelected(string(initialScope))

	scopeInfoLabel := widget.NewLabel(fmt.Sprintf("📌 %s", initialSummary))
	scopeInfoLabel.Wrapping = fyne.TextWrapWord
	scopeInfoLabel.TextStyle = fyne.TextStyle{Italic: true}

	// 2. Selector Giọng đọc Neural Tiếng Việt (Hoài Mỹ, Nam Minh, Hoàng Mai, Nam Khánh, Thu Trang)
	voiceLabels := AllVietnameseVoiceLabels()
	selectedVoiceID := VietnameseVoicePresets[0].ID
	voiceSelect := widget.NewSelect(voiceLabels, nil)
	voiceSelect.SetSelected(VietnameseVoicePresets[0].Label)

	voiceDescLabel := widget.NewLabel(VietnameseVoicePresets[0].Description)
	voiceDescLabel.Wrapping = fyne.TextWrapWord
	voiceDescLabel.TextStyle = fyne.TextStyle{Italic: true}

	voiceSelect.OnChanged = func(chosen string) {
		selectedVoiceID = ResolveVoiceIDFromLabel(chosen)
		voiceDescLabel.SetText(ResolveVoiceDescriptionFromLabel(chosen))
	}

	// 3. Khung soạn thảo / xem trước văn bản cần chuyển thành audio
	contentEntry := NewVietMultiLineEntry()
	contentEntry.SetPlaceHolder("Nội dung văn bản được biên dịch tự động theo phạm vi đã chọn...")
	contentEntry.SetText(initialContent)
	contentEntry.SetMinRowsVisible(7)

	statsLabel := widget.NewLabel("")
	updateStats := func(text string) {
		words := len(strings.Fields(text))
		chars := utf8.RuneCountInString(text)
		statsLabel.SetText(fmt.Sprintf("📊 Thống kê: %d từ  •  %d ký tự  •  Thời lượng ước tính: khoảng %.1f phút nghe (chuẩn 160 từ/phút)",
			words, chars, float64(words)/160.0))
	}
	updateStats(initialContent)

	contentEntry.SetOnChangedCallback(func(text string) {
		updateStats(text)
	})

	// 4. Đường dẫn lưu tệp MP3 đích
	suggestedFileName := sanitizeFileName(initialTitle) + ".mp3"
	pathEntry := NewVietEntry()
	pathEntry.SetText(suggestedFileName)

	// Lắng nghe thay đổi Phạm vi xuất bản: Tự động tổng hợp lại nội dung và cập nhật tên file
	scopeSelect.OnChanged = func(chosen string) {
		sc := AudioExportScope(chosen)
		compiled, title, totalScenes, summary := ui.CompileAudioTextForScope(sc)
		contentEntry.SetText(compiled)
		updateStats(compiled)
		suggestedFileName = sanitizeFileName(title) + ".mp3"
		pathEntry.SetText(suggestedFileName)
		scopeInfoLabel.SetText(fmt.Sprintf("📌 %s (tổng cộng %d cảnh được biên dịch)", summary, totalScenes))
	}

	browseBtn := widget.NewButtonWithIcon("Chọn nơi lưu...", theme.FolderOpenIcon(), func() {
		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, ui.window)
				return
			}
			if writer == nil {
				return
			}
			p := writer.URI().Path()
			_ = writer.Close()
			if !strings.HasSuffix(strings.ToLower(p), ".mp3") {
				p += ".mp3"
			}
			pathEntry.SetText(p)
		}, ui.window)
		fd.SetFileName(filepath.Base(pathEntry.Text))
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".mp3"}))
		fd.Show()
	})

	// 5. Trạng thái và thanh tiến trình
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Hide()

	statusLabel := widget.NewLabelWithStyle("Sẵn sàng xuất audio bằng giọng đọc Neural Tiếng Việt (Thuần Go WebSocket).", fyne.TextAlignLeading, fyne.TextStyle{Italic: true})

	reloadScopeBtn := widget.NewButtonWithIcon("Nạp lại theo phạm vi đang chọn", theme.ViewRefreshIcon(), func() {
		sc := AudioExportScope(scopeSelect.Selected)
		compiled, title, totalScenes, summary := ui.CompileAudioTextForScope(sc)
		contentEntry.SetText(compiled)
		updateStats(compiled)
		suggestedFileName = sanitizeFileName(title) + ".mp3"
		pathEntry.SetText(suggestedFileName)
		scopeInfoLabel.SetText(fmt.Sprintf("📌 %s (tổng cộng %d cảnh được biên dịch)", summary, totalScenes))
	})
	reloadScopeBtn.Importance = widget.LowImportance

	// 6. Xây dựng bố cục Dialog
	headerBox := container.NewVBox(
		widget.NewLabelWithStyle("🎧 XUẤT BẢN FILE AUDIO (MP3) — ĐA GIỌNG ĐỌC AI & PHẠM VI LINH HOẠT", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Sử dụng giao thức WebSocket trực tiếp tới dịch vụ giọng đọc Neural — Thuần Go 100% không phụ thuộc Python, với đầy đủ giọng điệu 3 miền Bắc - Trung - Nam."),
		widget.NewSeparator(),
	)

	scopeCard := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabelWithStyle("Phạm vi xuất bản:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, scopeSelect),
		scopeInfoLabel,
	)

	voiceCard := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabelWithStyle("Chọn giọng đọc:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, voiceSelect),
		voiceDescLabel,
	)

	contentHeader := container.NewBorder(
		nil, nil,
		widget.NewLabelWithStyle("Nội dung văn bản biên dịch để đọc thành tiếng:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		reloadScopeBtn,
	)

	pathCard := container.NewVBox(
		widget.NewLabelWithStyle("Đường dẫn lưu tệp MP3:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, browseBtn, pathEntry),
	)

	var audioDialog dialog.Dialog

	cancelBtn := widget.NewButton("Đóng", func() {
		if audioDialog != nil {
			audioDialog.Hide()
		}
	})

	exportBtn := widget.NewButtonWithIcon("Xuất file Audio", theme.MediaPlayIcon(), nil)
	exportBtn.Importance = widget.HighImportance

	exportBtn.OnTapped = func() {
		textToRead := strings.TrimSpace(contentEntry.Text)
		if textToRead == "" {
			dialog.ShowInformation("Thông báo", "Vui lòng nhập nội dung văn bản để chuyển đổi sang âm thanh.", ui.window)
			return
		}

		outPath := strings.TrimSpace(pathEntry.Text)
		if outPath == "" {
			outPath = suggestedFileName
		}
		if !strings.HasSuffix(strings.ToLower(outPath), ".mp3") {
			outPath += ".mp3"
		}

		wordCount := len(strings.Fields(textToRead))

		// Khóa giao diện và hiển thị tiến trình tổng hợp
		exportBtn.Disable()
		cancelBtn.Disable()
		scopeSelect.Disable()
		voiceSelect.Disable()
		progressBar.Show()

		statusLabel.SetText(fmt.Sprintf("⏳ Đang kết nối dịch vụ giọng đọc Edge-TTS (%s — %d từ)...", scopeSelect.Selected, wordCount))

		voiceChosen := selectedVoiceID
		go func() {
			err := ExportTextToAudioWithProgress(voiceChosen, textToRead, outPath, func(curr, total int, msg string) {
				statusLabel.SetText(fmt.Sprintf("⏳ %s", msg))
			})

			exportBtn.Enable()
			cancelBtn.Enable()
			scopeSelect.Enable()
			voiceSelect.Enable()
			progressBar.Hide()

			if err != nil {
				statusLabel.SetText("❌ Có lỗi xảy ra khi tạo audio.")
				dialog.ShowError(err, ui.window)
				return
			}

			PlayUIClickSound()
			statusLabel.SetText("✅ Xuất audio thành công!")

			fi, _ := os.Stat(outPath)
			fileSizeMB := float64(0)
			if fi != nil {
				fileSizeMB = float64(fi.Size()) / (1024 * 1024)
			}

			if ui.statusFooter != nil {
				ui.statusFooter.SetText(fmt.Sprintf("Đã xuất audio: %s (%.2f MB)", filepath.Base(outPath), fileSizeMB))
			}

			dialog.ShowInformation(
				"Xuất Audio Thành Công!",
				fmt.Sprintf("Đã xuất file âm thanh thành công (Thuần Go WebSocket)!\n\n• Tệp đích: %s\n• Phạm vi xuất bản: %s\n• Giọng đọc: %s\n• Dung lượng: %.2f MB\n• Số từ: %d từ\n• Thời lượng ước tính: %.1f phút nghe",
					outPath,
					scopeSelect.Selected,
					voiceSelect.Selected,
					fileSizeMB,
					wordCount,
					float64(wordCount)/160.0,
				),
				ui.window,
			)
		}()
	}

	actionsRow := container.NewHBox(
		cancelBtn,
		layout.NewSpacer(),
		exportBtn,
	)

	dialogBody := container.NewVScroll(container.NewVBox(
		headerBox,
		scopeCard,
		widget.NewSeparator(),
		voiceCard,
		widget.NewSeparator(),
		contentHeader,
		contentEntry,
		statsLabel,
		widget.NewSeparator(),
		pathCard,
		widget.NewSeparator(),
		progressBar,
		statusLabel,
		widget.NewSeparator(),
		actionsRow,
	))

	audioDialog = dialog.NewCustomWithoutButtons("Xuất Bản Audio (Edge-TTS)", dialogBody, ui.window)
	audioDialog.Resize(fyne.NewSize(780, 680))
	audioDialog.Show()
}
