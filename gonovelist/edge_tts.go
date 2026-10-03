package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type edgeTTSCommand struct {
	path string
	args []string
}

func resolveEdgeTTSCommand() (edgeTTSCommand, error) {
	if path, err := exec.LookPath("edge-tts"); err == nil {
		return edgeTTSCommand{path: path}, nil
	}

	if configuredPath := strings.TrimSpace(os.Getenv("GONOVELIST_EDGE_TTS")); configuredPath != "" {
		if path, err := exec.LookPath(configuredPath); err == nil {
			return edgeTTSCommand{path: path}, nil
		}
	}

	for _, root := range bundledPythonRoots() {
		for _, pythonPath := range pythonExecutables(root) {
			if hasEdgeTTSModule(pythonPath) {
				return edgeTTSCommand{path: pythonPath, args: []string{"-m", "edge_tts"}}, nil
			}
		}
		for _, executablePath := range edgeTTSExecutables(root) {
			if path, err := exec.LookPath(executablePath); err == nil {
				return edgeTTSCommand{path: path}, nil
			}
		}
	}

	for _, name := range systemPythonNames() {
		pythonPath, err := exec.LookPath(name)
		if err == nil && hasEdgeTTSModule(pythonPath) {
			return edgeTTSCommand{path: pythonPath, args: []string{"-m", "edge_tts"}}, nil
		}
	}

	return edgeTTSCommand{}, fmt.Errorf("không tìm thấy công cụ Edge-TTS. Gói phát hành cần kèm runtime Python và gói edge-tts trong thư mục assets/python; ứng dụng sẽ tự sử dụng bộ công cụ này khi khởi chạy")
}

func bundledPythonRoots() []string {
	var roots []string
	addRoot := func(path string) {
		if path == "" {
			return
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return
		}
		for _, existing := range roots {
			if existing == abs {
				return
			}
		}
		roots = append(roots, abs)
	}

	if executablePath, err := os.Executable(); err == nil {
		executableDir := filepath.Dir(executablePath)
		addRoot(filepath.Join(executableDir, "assets", "python"))
		addRoot(filepath.Join(executableDir, "python"))
		addRoot(filepath.Join(executableDir, "..", "Resources", "assets", "python"))
	}
	addRoot(filepath.Join("assets", "python"))
	if configDir, err := os.UserConfigDir(); err == nil {
		addRoot(filepath.Join(configDir, "GoNovelist", "python"))
	}
	if configuredRoot := strings.TrimSpace(os.Getenv("GONOVELIST_PYTHON_HOME")); configuredRoot != "" {
		addRoot(configuredRoot)
	}
	return roots
}

func pythonExecutables(root string) []string {
	paths := []string{
		filepath.Join(root, "bin", "python3"),
		filepath.Join(root, "bin", "python"),
		filepath.Join(root, "python.exe"),
		filepath.Join(root, "Scripts", "python.exe"),
	}
	if runtime.GOOS == "windows" {
		paths = append([]string{filepath.Join(root, "python.exe")}, paths...)
	}
	return paths
}

func edgeTTSExecutables(root string) []string {
	if runtime.GOOS == "windows" {
		return []string{
			filepath.Join(root, "Scripts", "edge-tts.exe"),
			filepath.Join(root, "Scripts", "edge-tts"),
		}
	}
	return []string{
		filepath.Join(root, "bin", "edge-tts"),
		filepath.Join(root, "Scripts", "edge-tts"),
	}
}

func systemPythonNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"python.exe", "python"}
	}
	return []string{"python3", "python"}
}

func hasEdgeTTSModule(pythonPath string) bool {
	cmd := exec.Command(pythonPath, "-c", "import edge_tts")
	return cmd.Run() == nil
}

func formatEdgeTTSAdjustment(value int) string {
	if value < -100 {
		value = -100
	}
	if value > 100 {
		value = 100
	}
	if value >= 0 {
		return fmt.Sprintf("+%d%%", value)
	}
	return fmt.Sprintf("%d%%", value)
}

func edgeTTSArguments(commandArgs []string, voiceID, text, outputPath string, rate, volume int) []string {
	args := append([]string(nil), commandArgs...)
	args = append(args,
		"--voice", voiceID,
		"--rate", formatEdgeTTSAdjustment(rate),
		"--volume", formatEdgeTTSAdjustment(volume),
		"--text", text,
		"--write-media", outputPath,
	)
	return args
}

func synthesizeSpeechChunk(command edgeTTSCommand, voiceID, chunkText, outputPath string, rate, volume int) error {
	args := edgeTTSArguments(command.args, voiceID, chunkText, outputPath, rate, volume)
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		_ = os.Remove(outputPath)
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		cmd := exec.CommandContext(ctx, command.path, args...)
		output, err := cmd.CombinedOutput()
		contextErr := ctx.Err()
		cancel()

		if err == nil {
			info, statErr := os.Stat(outputPath)
			if statErr == nil && info.Size() > 0 {
				return nil
			}
			lastErr = fmt.Errorf("Edge-TTS không tạo được dữ liệu âm thanh cho phần văn bản")
		} else if contextErr == context.DeadlineExceeded {
			lastErr = formatEdgeTTSProcessError("quá thời gian chờ tổng hợp phần văn bản", output)
		} else {
			lastErr = formatEdgeTTSProcessError(err.Error(), output)
		}

		if attempt < 2 {
			time.Sleep(500 * time.Millisecond)
		}
	}
	return fmt.Errorf("không thể tổng hợp sau 2 lần thử: %w", lastErr)
}

func formatEdgeTTSProcessError(reason string, output []byte) error {
	diagnostic := strings.TrimSpace(string(output))
	if diagnostic != "" {
		const maxDiagnosticRunes = 1200
		runes := []rune(diagnostic)
		if len(runes) > maxDiagnosticRunes {
			diagnostic = string(runes[:maxDiagnosticRunes]) + "…"
		}
		return fmt.Errorf("Edge-TTS không thể tổng hợp phần văn bản (%s). Chi tiết kỹ thuật: %s", reason, diagnostic)
	}
	return fmt.Errorf("Edge-TTS không thể tổng hợp phần văn bản (%s)", reason)
}

func ExportTextToAudioWithProgress(voiceID, textContent, outputPath string, onProgress func(current, total int, msg string)) error {
	return ExportTextToAudioWithOptionsProgress(voiceID, textContent, outputPath, 0, 0, onProgress)
}

func ExportTextToAudioWithOptionsProgress(voiceID, textContent, outputPath string, rate, volume int, onProgress func(current, total int, msg string)) error {
	cleanText := strings.TrimSpace(textContent)
	if cleanText == "" {
		return fmt.Errorf("nội dung văn bản để xuất âm thanh không được để trống")
	}
	if outputPath == "" {
		return fmt.Errorf("đường dẫn tệp âm thanh đích không được để trống")
	}
	if voiceID == "" {
		voiceID = "vi-VN-HoaiMyNeural"
	}
	if !strings.HasSuffix(strings.ToLower(outputPath), ".mp3") {
		outputPath += ".mp3"
	}

	chunks := SplitTextIntoTTSChunks(cleanText, 2500)
	if len(chunks) == 0 {
		return fmt.Errorf("không tìm thấy đoạn văn bản hợp lệ để xuất âm thanh")
	}
	if onProgress != nil {
		onProgress(0, len(chunks), "Đang khởi tạo công cụ giọng đọc...")
	}
	command, err := resolveEdgeTTSCommand()
	if err != nil {
		return err
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("không thể tạo thư mục lưu tệp âm thanh")
	}
	assembledFile, err := os.CreateTemp(outputDir, ".gonovelist-audio-*.mp3")
	if err != nil {
		return fmt.Errorf("không thể tạo tệp âm thanh tạm")
	}
	assembledPath := assembledFile.Name()
	defer os.Remove(assembledPath)
	defer assembledFile.Close()

	for index, chunk := range chunks {
		partNumber := index + 1
		if onProgress != nil {
			onProgress(partNumber-1, len(chunks), fmt.Sprintf("Đang tổng hợp audio phần %d/%d...", partNumber, len(chunks)))
		}

		chunkFile, err := os.CreateTemp(outputDir, ".gonovelist-part-*.mp3")
		if err != nil {
			return fmt.Errorf("không thể tạo tệp tạm cho phần %d/%d", partNumber, len(chunks))
		}
		chunkPath := chunkFile.Name()
		if err := chunkFile.Close(); err != nil {
			os.Remove(chunkPath)
			return fmt.Errorf("không thể chuẩn bị tệp tạm cho phần %d/%d", partNumber, len(chunks))
		}
		if err := os.Remove(chunkPath); err != nil {
			return fmt.Errorf("không thể chuẩn bị tệp tạm cho phần %d/%d", partNumber, len(chunks))
		}

		synthErr := synthesizeSpeechChunk(command, voiceID, chunk, chunkPath, rate, volume)
		if synthErr != nil {
			os.Remove(chunkPath)
			return fmt.Errorf("lỗi khi tổng hợp audio phần %d/%d: %w", partNumber, len(chunks), synthErr)
		}
		chunkInput, err := os.Open(chunkPath)
		if err != nil {
			os.Remove(chunkPath)
			return fmt.Errorf("không thể đọc tệp audio phần %d/%d", partNumber, len(chunks))
		}
		_, copyErr := io.Copy(assembledFile, chunkInput)
		closeErr := chunkInput.Close()
		os.Remove(chunkPath)
		if copyErr != nil || closeErr != nil {
			return fmt.Errorf("không thể ghép audio phần %d/%d", partNumber, len(chunks))
		}
	}

	if err := assembledFile.Sync(); err != nil {
		return fmt.Errorf("không thể hoàn tất tệp audio tạm")
	}
	if err := assembledFile.Close(); err != nil {
		return fmt.Errorf("không thể đóng tệp audio tạm")
	}

	outputFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("không thể mở tệp đích để lưu audio")
	}
	assembledInput, err := os.Open(assembledPath)
	if err != nil {
		outputFile.Close()
		return fmt.Errorf("không thể đọc dữ liệu audio đã tổng hợp")
	}
	_, copyErr := io.Copy(outputFile, assembledInput)
	inputCloseErr := assembledInput.Close()
	syncErr := outputFile.Sync()
	outputCloseErr := outputFile.Close()
	if copyErr != nil || inputCloseErr != nil || syncErr != nil || outputCloseErr != nil {
		return fmt.Errorf("không thể ghi hoàn chỉnh tệp audio đích")
	}

	info, err := os.Stat(outputPath)
	if err != nil || info.Size() == 0 {
		return fmt.Errorf("tệp audio chưa được tạo hoặc không có dữ liệu")
	}
	if onProgress != nil {
		onProgress(len(chunks), len(chunks), "Xuất file thành công!")
	}
	return nil
}

func ExportTextToAudioWithEdgeTTS(voiceID, textContent, outputPath string) error {
	return ExportTextToAudioWithProgress(voiceID, textContent, outputPath, nil)
}
