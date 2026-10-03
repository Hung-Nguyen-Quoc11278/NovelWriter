package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
	"time"
)

func TestDOCXFormatMetadata(t *testing.T) {
	if got := FormatExtension(ExportFormatDOCX); got != ".docx" {
		t.Fatalf("FormatExtension(DOCX) = %q, muốn .docx", got)
	}
	if got := FormatLabel(ExportFormatDOCX); !strings.Contains(got, ".docx") {
		t.Fatalf("FormatLabel(DOCX) không hiển thị phần mở rộng: %q", got)
	}
}

func TestExportManuscriptDOCXPackage(t *testing.T) {
	manuscript := &FilteredManuscript{
		Project: Project{
			Title:    "Bản đồ & thấu kính",
			Author:   "Nguyễn An",
			Genre:    "Kỳ ảo",
			Synopsis: "Tóm tắt tiếng Việt.",
		},
		ScopeLabel:  "Toàn bộ tác phẩm",
		TotalScenes: 1,
		TotalWords:  12,
		GeneratedAt: time.Date(2026, time.October, 3, 10, 0, 0, 0, time.UTC),
		Acts: []ExportedAct{{
			Act: Act{Title: "Hồi I — Khởi đầu"},
			Chapters: []ExportedChapter{{
				Chapter: Chapter{Title: "Chương 1: Ánh đèn"},
				Scenes: []Scene{{
					Title:   "Cảnh 1",
					Content: "Một **đoạn đậm** và *nghiêng*.\n\n> Lời trích dẫn.\n\n### Tiêu đề phụ\n\n<u>Gạch chân</u>\n\n* * *",
				}},
			}},
		}},
	}
	data, err := ExportManuscriptDOCX(manuscript, ExportOptions{IncludeSynopsis: true, IncludeSceneTitle: true})
	if err != nil {
		t.Fatalf("ExportManuscriptDOCX() lỗi: %v", err)
	}

	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("tệp DOCX không phải ZIP hợp lệ: %v", err)
	}
	parts := make(map[string]string, len(archive.File))
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatalf("không mở được thành phần %s: %v", file.Name, err)
		}
		content, readErr := io.ReadAll(reader)
		closeErr := reader.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("không đọc được thành phần %s: %v %v", file.Name, readErr, closeErr)
		}
		parts[file.Name] = string(content)
	}

	requiredParts := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"docProps/core.xml",
		"word/document.xml",
		"word/styles.xml",
		"word/_rels/document.xml.rels",
	}
	for _, name := range requiredParts {
		content, ok := parts[name]
		if !ok {
			t.Errorf("thiếu thành phần DOCX bắt buộc %q", name)
			continue
		}
		decoder := xml.NewDecoder(strings.NewReader(content))
		for {
			if _, err := decoder.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Errorf("XML trong %s không hợp lệ: %v", name, err)
				break
			}
		}
	}

	document := parts["word/document.xml"]
	for _, expected := range []string{
		"Bản đồ &amp; thấu kính",
		"Hồi I — Khởi đầu",
		"Chương 1: Ánh đèn",
		"Tóm tắt tiếng Việt.",
		"<w:b/>",
		"<w:i/>",
		`<w:u w:val="single"/>`,
		"Lời trích dẫn.",
		"Tiêu đề phụ",
		"* * *",
	} {
		if !strings.Contains(document, expected) {
			t.Errorf("document.xml thiếu nội dung/định dạng %q", expected)
		}
	}
}

func TestExportManuscriptBytesDispatchesDOCX(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	projects, err := store.ListProjects()
	if err != nil || len(projects) == 0 {
		t.Fatalf("không nạp được tác phẩm mẫu: %v", err)
	}

	data, _, err := store.ExportManuscriptBytes(projects[0], ExportOptions{
		Format: ExportFormatDOCX,
		Scope:  ExportScopeAllBook,
	})
	if err != nil {
		t.Fatalf("dispatcher không xuất DOCX: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("PK")) {
		t.Fatal("dispatcher DOCX không trả về gói ZIP")
	}
}
