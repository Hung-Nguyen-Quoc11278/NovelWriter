package main

import (
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
