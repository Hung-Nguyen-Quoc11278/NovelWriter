package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFormatEdgeTTSAdjustment(t *testing.T) {
	tests := []struct {
		value int
		want  string
	}{
		{value: -125, want: "-100%"},
		{value: -15, want: "-15%"},
		{value: 0, want: "+0%"},
		{value: 20, want: "+20%"},
		{value: 125, want: "+100%"},
	}

	for _, test := range tests {
		if got := formatEdgeTTSAdjustment(test.value); got != test.want {
			t.Errorf("formatEdgeTTSAdjustment(%d) = %q, muốn %q", test.value, got, test.want)
		}
	}
}

func TestFormatEdgeTTSProcessErrorIncludesBoundedDiagnostic(t *testing.T) {
	diagnostic := strings.Repeat("lỗi mạng ", 200)
	err := formatEdgeTTSProcessError("exit status 1", []byte(diagnostic))
	if !strings.Contains(err.Error(), "Chi tiết kỹ thuật:") {
		t.Fatalf("lỗi chưa có phần chẩn đoán: %v", err)
	}
	if len([]rune(err.Error())) > 1400 {
		t.Fatalf("thông báo lỗi vượt giới hạn dự kiến")
	}
}

func TestEdgeTTSSynthesisIntegration(t *testing.T) {
	if os.Getenv("GONOVELIST_RUN_EDGE_TTS_INTEGRATION") != "1" {
		t.Skip("đặt GONOVELIST_RUN_EDGE_TTS_INTEGRATION=1 để chạy kiểm tra dịch vụ Edge-TTS thật")
	}
	command, err := resolveEdgeTTSCommand()
	if err != nil {
		t.Fatalf("không phân giải được Edge-TTS: %v", err)
	}
	outputPath := t.TempDir() + string(os.PathSeparator) + "kiem-tra.mp3"
	if err := synthesizeSpeechChunk(command, "vi-VN-HoaiMyNeural", "Xin chào.", outputPath, 0, 0); err != nil {
		t.Fatalf("tổng hợp kiểm tra thất bại: %v", err)
	}
	info, err := os.Stat(outputPath)
	if err != nil || info.Size() == 0 {
		t.Fatalf("không tạo được MP3 kiểm tra: %v", err)
	}
}

func TestEdgeTTSArguments(t *testing.T) {
	got := edgeTTSArguments(
		[]string{"-m", "edge_tts"},
		"vi-VN-HoaiMyNeural",
		"Xin chào, thế giới!",
		"/tmp/đầu ra.mp3",
		15,
		-5,
	)
	want := []string{
		"-m", "edge_tts",
		"--voice", "vi-VN-HoaiMyNeural",
		"--rate", "+15%",
		"--volume", "-5%",
		"--text", "Xin chào, thế giới!",
		"--write-media", "/tmp/đầu ra.mp3",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("edgeTTSArguments() = %#v, muốn %#v", got, want)
	}
}

func TestVietnameseVoicePresetsMatchAvailableEdgeTTSVoices(t *testing.T) {
	want := []string{
		"vi-VN-HoaiMyNeural",
		"vi-VN-NamMinhNeural",
	}
	if len(VietnameseVoicePresets) != len(want) {
		t.Fatalf("số voice tiếng Việt khả dụng = %d, muốn %d", len(VietnameseVoicePresets), len(want))
	}
	for index, voice := range VietnameseVoicePresets {
		if voice.ID != want[index] {
			t.Errorf("voice[%d] = %q, muốn %q", index, voice.ID, want[index])
		}
		if got := ResolveVoiceIDFromLabel(voice.Label); got != voice.ID {
			t.Errorf("ResolveVoiceIDFromLabel(%q) = %q, muốn %q", voice.Label, got, voice.ID)
		}
	}
}

func TestSplitTextIntoTTSChunksLimitsLongVietnameseText(t *testing.T) {
	text := strings.Repeat("ế", 17)
	chunks := SplitTextIntoTTSChunks(text, 5)
	var rebuilt strings.Builder
	for _, chunk := range chunks {
		if count := utf8.RuneCountInString(chunk); count > 5 {
			t.Fatalf("đoạn có %d rune, vượt giới hạn 5", count)
		}
		rebuilt.WriteString(chunk)
	}
	if rebuilt.String() != text {
		t.Fatalf("ghép lại các đoạn không khớp văn bản gốc")
	}
}
