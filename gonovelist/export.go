package main

import (
	"archive/zip"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ExportFormat định nghĩa các định dạng tệp xuất bản thảo được hỗ trợ.
type ExportFormat string

const (
	ExportFormatTXT      ExportFormat = "txt"
	ExportFormatMarkdown ExportFormat = "markdown"
	ExportFormatHTML     ExportFormat = "html"
	ExportFormatODT      ExportFormat = "odt"
	ExportFormatPDF      ExportFormat = "pdf"
	ExportFormatEPUB     ExportFormat = "epub"
)

// ExportScopeType định nghĩa phạm vi xuất bản thảo (Toàn bộ tác phẩm, Theo Hồi, Chương/Cảnh hiện tại).
type ExportScopeType string

const (
	ExportScopeAllBook     ExportScopeType = "all_book"
	ExportScopeSelectedAct ExportScopeType = "selected_act"
	ExportScopeCurrentNode ExportScopeType = "current_node"
)

// ExportOptions lưu cấu hình xuất bản do người dùng chọn trong hộp thoại.
type ExportOptions struct {
	Format            ExportFormat
	Scope             ExportScopeType
	SelectedActID     int64
	SelectedChapterID int64
	SelectedSceneID   int64
	CurrentNodeKind   string // "act", "chapter", hoặc "scene"
	IncludeSynopsis   bool
	IncludeSceneTitle bool
	OutputPath        string
}

// ExportedChapter chứa thông tin Chương và danh sách Cảnh đã lọc theo phạm vi xuất bản.
type ExportedChapter struct {
	Chapter Chapter
	Scenes  []Scene
}

// ExportedAct chứa thông tin Hồi và danh sách Chương đã lọc theo phạm vi xuất bản.
type ExportedAct struct {
	Act      Act
	Chapters []ExportedChapter
}

// FilteredManuscript đại diện cho cấu trúc bản thảo đã được lọc theo Phạm vi xuất bản.
type FilteredManuscript struct {
	Project     Project
	ScopeLabel  string
	Acts        []ExportedAct
	TotalScenes int
	TotalWords  int
	GeneratedAt time.Time
}

// FormatLabel trả về tên hiển thị Tiếng Việt của định dạng xuất bản.
func FormatLabel(format ExportFormat) string {
	switch format {
	case ExportFormatTXT:
		return "Văn bản thuần túy (.txt)"
	case ExportFormatMarkdown:
		return "Markdown (.md)"
	case ExportFormatHTML:
		return "Trang web HTML (.html)"
	case ExportFormatODT:
		return "OpenDocument Text - LibreOffice (.odt)"
	case ExportFormatPDF:
		return "Tài liệu PDF (.pdf)"
	case ExportFormatEPUB:
		return "Sách điện tử EPUB (.epub)"
	default:
		return "Văn bản (.txt)"
	}
}

// FormatExtension trả về phần mở rộng tệp tương ứng với định dạng.
func FormatExtension(format ExportFormat) string {
	switch format {
	case ExportFormatTXT:
		return ".txt"
	case ExportFormatMarkdown:
		return ".md"
	case ExportFormatHTML:
		return ".html"
	case ExportFormatODT:
		return ".odt"
	case ExportFormatPDF:
		return ".pdf"
	case ExportFormatEPUB:
		return ".epub"
	default:
		return ".txt"
	}
}

// sanitizeFileName chuyển đổi tiêu đề Tiếng Việt thành tên tệp an toàn trên hệ thống.
func sanitizeFileName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "ban_thao"
	}
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_",
		"?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	cleaned := replacer.Replace(trimmed)
	return filepath.Clean(cleaned)
}

// BuildFilteredManuscript truy vấn cơ sở dữ liệu SQLite và lọc cấu trúc Hồi -> Chương -> Cảnh theo phạm vi đã chọn.
func (s *Store) BuildFilteredManuscript(project Project, opts ExportOptions) (*FilteredManuscript, error) {
	ms := &FilteredManuscript{
		Project:     project,
		ScopeLabel:  "Toàn bộ tác phẩm",
		GeneratedAt: time.Now(),
	}

	allActs, err := s.ListActs(project.ID)
	if err != nil {
		return nil, err
	}

	switch opts.Scope {
	case ExportScopeSelectedAct:
		for _, act := range allActs {
			if act.ID == opts.SelectedActID {
				ms.ScopeLabel = fmt.Sprintf("Theo Hồi chỉ định: %s", act.Title)
				expAct, scenesCount, wordsCount, err := s.loadExportedAct(act, 0, 0)
				if err != nil {
					return nil, err
				}
				ms.Acts = append(ms.Acts, expAct)
				ms.TotalScenes += scenesCount
				ms.TotalWords += wordsCount
				break
			}
		}
		if len(ms.Acts) == 0 && len(allActs) > 0 {
			act := allActs[0]
			ms.ScopeLabel = fmt.Sprintf("Theo Hồi chỉ định: %s", act.Title)
			expAct, scenesCount, wordsCount, err := s.loadExportedAct(act, 0, 0)
			if err != nil {
				return nil, err
			}
			ms.Acts = append(ms.Acts, expAct)
			ms.TotalScenes += scenesCount
			ms.TotalWords += wordsCount
		}

	case ExportScopeCurrentNode:
		switch opts.CurrentNodeKind {
		case "act":
			for _, act := range allActs {
				if act.ID == opts.SelectedActID {
					ms.ScopeLabel = fmt.Sprintf("Hồi hiện tại: %s", act.Title)
					expAct, scenesCount, wordsCount, err := s.loadExportedAct(act, 0, 0)
					if err != nil {
						return nil, err
					}
					ms.Acts = append(ms.Acts, expAct)
					ms.TotalScenes += scenesCount
					ms.TotalWords += wordsCount
					break
				}
			}
		case "chapter":
			for _, act := range allActs {
				chapters, err := s.ListChapters(act.ID)
				if err != nil {
					return nil, err
				}
				for _, ch := range chapters {
					if ch.ID == opts.SelectedChapterID {
						ms.ScopeLabel = fmt.Sprintf("Chương hiện tại: %s (%s)", ch.Title, act.Title)
						expAct, scenesCount, wordsCount, err := s.loadExportedAct(act, ch.ID, 0)
						if err != nil {
							return nil, err
						}
						ms.Acts = append(ms.Acts, expAct)
						ms.TotalScenes += scenesCount
						ms.TotalWords += wordsCount
						break
					}
				}
			}
		case "scene":
			for _, act := range allActs {
				chapters, err := s.ListChapters(act.ID)
				if err != nil {
					return nil, err
				}
				for _, ch := range chapters {
					scenes, err := s.ListScenes(ch.ID)
					if err != nil {
						return nil, err
					}
					for _, sc := range scenes {
						if sc.ID == opts.SelectedSceneID {
							ms.ScopeLabel = fmt.Sprintf("Cảnh hiện tại: %s (%s)", sc.Title, ch.Title)
							expAct, scenesCount, wordsCount, err := s.loadExportedAct(act, ch.ID, sc.ID)
							if err != nil {
								return nil, err
							}
							ms.Acts = append(ms.Acts, expAct)
							ms.TotalScenes += scenesCount
							ms.TotalWords += wordsCount
							break
						}
					}
				}
			}
		}

	default: // ExportScopeAllBook
		ms.ScopeLabel = "Toàn bộ tác phẩm"
		for _, act := range allActs {
			expAct, scenesCount, wordsCount, err := s.loadExportedAct(act, 0, 0)
			if err != nil {
				return nil, err
			}
			ms.Acts = append(ms.Acts, expAct)
			ms.TotalScenes += scenesCount
			ms.TotalWords += wordsCount
		}
	}

	return ms, nil
}

func (s *Store) loadExportedAct(act Act, filterChapterID int64, filterSceneID int64) (ExportedAct, int, int, error) {
	expAct := ExportedAct{Act: act}
	chapters, err := s.ListChapters(act.ID)
	if err != nil {
		return expAct, 0, 0, err
	}

	totalScenes := 0
	totalWords := 0

	for _, ch := range chapters {
		if filterChapterID > 0 && ch.ID != filterChapterID {
			continue
		}
		scenes, err := s.ListScenes(ch.ID)
		if err != nil {
			return expAct, 0, 0, err
		}
		var filteredScenes []Scene
		for _, sc := range scenes {
			if filterSceneID > 0 && sc.ID != filterSceneID {
				continue
			}
			filteredScenes = append(filteredScenes, sc)
			totalScenes++
			totalWords += sc.WordCount
		}
		expAct.Chapters = append(expAct.Chapters, ExportedChapter{
			Chapter: ch,
			Scenes:  filteredScenes,
		})
	}
	return expAct, totalScenes, totalWords, nil
}

// ExportManuscriptBytes biên dịch FilteredManuscript ra mảng byte theo định dạng chỉ định.
func (s *Store) ExportManuscriptBytes(project Project, opts ExportOptions) ([]byte, *FilteredManuscript, error) {
	ms, err := s.BuildFilteredManuscript(project, opts)
	if err != nil {
		return nil, nil, err
	}

	var data []byte
	switch opts.Format {
	case ExportFormatTXT:
		data, err = ExportManuscriptTXT(ms, opts)
	case ExportFormatMarkdown:
		data, err = ExportManuscriptMarkdownWithOptions(ms, opts)
	case ExportFormatHTML:
		data, err = ExportManuscriptHTMLWithOptions(ms, opts)
	case ExportFormatODT:
		data, err = ExportManuscriptODT(ms, opts)
	case ExportFormatPDF:
		data, err = ExportManuscriptPDF(ms, opts)
	case ExportFormatEPUB:
		data, err = ExportManuscriptEPUB(ms, opts)
	default:
		data, err = ExportManuscriptTXT(ms, opts)
	}
	if err != nil {
		return nil, nil, err
	}
	return data, ms, nil
}

// ==================== 1. PLAIN TEXT (.txt) EXPORTER ====================

// ExportManuscriptTXT xuất bản thảo ra văn bản thuần túy (.txt) chuẩn UTF-8 không chứa thẻ đánh dấu.
func ExportManuscriptTXT(ms *FilteredManuscript, opts ExportOptions) ([]byte, error) {
	var b strings.Builder
	titleUpper := strings.ToUpper(ms.Project.Title)
	b.WriteString(titleUpper + "\n")
	b.WriteString(strings.Repeat("=", utf8.RuneCountInString(titleUpper)) + "\n\n")

	if ms.Project.Author != "" {
		b.WriteString(fmt.Sprintf("Tác giả: %s\n", ms.Project.Author))
	}
	if ms.Project.Genre != "" {
		b.WriteString(fmt.Sprintf("Thể loại: %s\n", ms.Project.Genre))
	}
	b.WriteString(fmt.Sprintf("Phạm vi xuất bản: %s\n", ms.ScopeLabel))
	b.WriteString(fmt.Sprintf("Tổng số từ: %d từ\n\n", ms.TotalWords))

	if opts.IncludeSynopsis && strings.TrimSpace(ms.Project.Synopsis) != "" {
		b.WriteString("TÓM TẮT TÁC PHẨM:\n")
		b.WriteString(strings.TrimSpace(ms.Project.Synopsis) + "\n\n")
	}

	b.WriteString(strings.Repeat("=", 60) + "\n\n")

	for _, expAct := range ms.Acts {
		b.WriteString(strings.ToUpper(expAct.Act.Title) + "\n")
		b.WriteString(strings.Repeat("-", 60) + "\n\n")

		for _, expCh := range expAct.Chapters {
			b.WriteString(expCh.Chapter.Title + "\n\n")

			for i, sc := range expCh.Scenes {
				if opts.IncludeSceneTitle && strings.TrimSpace(sc.Title) != "" {
					b.WriteString(fmt.Sprintf("[%s]\n\n", sc.Title))
				}
				blocks := ParseRichProseBlocks(sc.Content)
				b.WriteString(RenderRichBlocksPlainText(blocks))
				if i < len(expCh.Scenes)-1 {
					b.WriteString("                    * * *\n\n")
				}
			}
			b.WriteString("\n")
		}
	}

	return []byte(b.String()), nil
}

// ==================== 2. MARKDOWN (.md) EXPORTER ====================

// ExportManuscriptMarkdownWithOptions xuất bản thảo ra Markdown (.md) theo phạm vi tùy chọn.
func ExportManuscriptMarkdownWithOptions(ms *FilteredManuscript, opts ExportOptions) ([]byte, error) {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s\n\n", ms.Project.Title))
	if ms.Project.Author != "" {
		b.WriteString(fmt.Sprintf("**Tác giả:** %s  \n", ms.Project.Author))
	}
	if ms.Project.Genre != "" {
		b.WriteString(fmt.Sprintf("**Thể loại:** %s  \n", ms.Project.Genre))
	}
	b.WriteString(fmt.Sprintf("**Phạm vi xuất bản:** %s  \n", ms.ScopeLabel))
	b.WriteString(fmt.Sprintf("**Tổng số từ:** %d từ\n\n", ms.TotalWords))

	if opts.IncludeSynopsis && strings.TrimSpace(ms.Project.Synopsis) != "" {
		b.WriteString(fmt.Sprintf("> %s\n\n", strings.TrimSpace(ms.Project.Synopsis)))
	}
	b.WriteString("---\n\n")

	for _, expAct := range ms.Acts {
		b.WriteString(fmt.Sprintf("## %s\n\n", expAct.Act.Title))
		for _, expCh := range expAct.Chapters {
			b.WriteString(fmt.Sprintf("### %s\n\n", expCh.Chapter.Title))
			for i, sc := range expCh.Scenes {
				if opts.IncludeSceneTitle && strings.TrimSpace(sc.Title) != "" {
					b.WriteString(fmt.Sprintf("#### %s\n\n", sc.Title))
				}
				if strings.TrimSpace(sc.Content) != "" {
					b.WriteString(NormalizeRichProseToMarkdown(sc.Content) + "\n\n")
				}
				if i < len(expCh.Scenes)-1 {
					b.WriteString("* * *\n\n")
				}
			}
		}
	}
	return []byte(b.String()), nil
}

// ==================== 3. HTML (.html) EXPORTER ====================

// ExportManuscriptHTMLWithOptions xuất bản thảo ra tệp HTML (.html) theo phạm vi tùy chọn.
func ExportManuscriptHTMLWithOptions(ms *FilteredManuscript, opts ExportOptions) ([]byte, error) {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="vi">
<head>
<meta charset="UTF-8">
<title>` + html.EscapeString(ms.Project.Title) + `</title>
<style>
  body { font-family: 'Noto Serif', 'DejaVu Serif', 'Georgia', 'Times New Roman', serif; max-width: 740px; margin: 3rem auto; padding: 0 1.5rem; color: #1C1B18; background: #FAF7F2; line-height: 1.85; }
  h1 { font-size: 2.4rem; margin-bottom: 0.25rem; }
  .meta { color: #57534E; font-style: italic; margin-bottom: 1.5rem; }
  .synopsis { border-left: 3px solid #8B3A2B; padding-left: 1rem; color: #3F3C36; font-style: italic; margin-bottom: 2rem; }
  h2 { margin-top: 3rem; border-bottom: 1px solid #D6D0C4; padding-bottom: 0.4rem; }
  h3 { margin-top: 2rem; color: #3F3C36; }
  h4 { margin-top: 1.5rem; color: #78716C; font-weight: normal; text-transform: uppercase; letter-spacing: 0.08em; font-size: 0.85rem; }
  h5.sub-heading { margin-top: 1.4rem; margin-bottom: 0.6rem; color: #2D2A24; font-size: 1.1rem; font-weight: bold; }
  p { margin: 1.1rem 0; text-indent: 1.5rem; text-align: justify; }
  blockquote { margin: 1.2rem 1.5rem; padding: 0.5rem 1.2rem; border-left: 3px solid #8B3A2B; background: rgba(139, 58, 43, 0.05); font-style: italic; color: #3F3C36; }
  blockquote p { text-indent: 0; margin: 0.4rem 0; }
  u { text-decoration: underline; text-underline-offset: 2px; }
  hr.scene-break { border: none; text-align: center; margin: 2rem 0; }
  hr.scene-break::after { content: "* * *"; color: #78716C; letter-spacing: 0.4em; }
</style>
</head>
<body>
`)
	b.WriteString(fmt.Sprintf("<h1>%s</h1>\n", html.EscapeString(ms.Project.Title)))
	b.WriteString(fmt.Sprintf("<div class=\"meta\">Tác giả: %s &bull; Thể loại: %s &bull; %s (%d từ)</div>\n",
		html.EscapeString(ms.Project.Author),
		html.EscapeString(ms.Project.Genre),
		html.EscapeString(ms.ScopeLabel),
		ms.TotalWords,
	))

	if opts.IncludeSynopsis && strings.TrimSpace(ms.Project.Synopsis) != "" {
		b.WriteString(fmt.Sprintf("<div class=\"synopsis\">%s</div>\n", html.EscapeString(strings.TrimSpace(ms.Project.Synopsis))))
	}

	for _, expAct := range ms.Acts {
		b.WriteString(fmt.Sprintf("<h2>%s</h2>\n", html.EscapeString(expAct.Act.Title)))
		for _, expCh := range expAct.Chapters {
			b.WriteString(fmt.Sprintf("<h3>%s</h3>\n", html.EscapeString(expCh.Chapter.Title)))
			for i, sc := range expCh.Scenes {
				if opts.IncludeSceneTitle && strings.TrimSpace(sc.Title) != "" {
					b.WriteString(fmt.Sprintf("<h4>%s</h4>\n", html.EscapeString(sc.Title)))
				}
				blocks := ParseRichProseBlocks(sc.Content)
				b.WriteString(RenderRichBlocksHTML(blocks, false, "    "))
				if i < len(expCh.Scenes)-1 {
					b.WriteString("<hr class=\"scene-break\">\n")
				}
			}
		}
	}
	b.WriteString("</body>\n</html>")
	return []byte(b.String()), nil
}

// ==================== 4. OPENDOCUMENT TEXT (.odt) EXPORTER ====================

// ExportManuscriptODT tạo tệp chuẩn OASIS OpenDocument Text (.odt) tương thích LibreOffice Writer với đầy đủ Heading và Tiếng Việt UTF-8.
func ExportManuscriptODT(ms *FilteredManuscript, opts ExportOptions) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// 1. Tệp mimetype bắt buộc đứng đầu tiên và không nén (zip.Store) theo chuẩn OASIS ODF 1.2
	mimeHeader := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	}
	mimeWriter, err := zw.CreateHeader(mimeHeader)
	if err != nil {
		return nil, err
	}
	if _, err := mimeWriter.Write([]byte("application/vnd.oasis.opendocument.text")); err != nil {
		return nil, err
	}

	// 2. META-INF/manifest.xml
	manifestXML := `<?xml version="1.0" encoding="UTF-8"?>
<manifest:manifest xmlns:manifest="urn:oasis:names:tc:opendocument:xmlns:manifest:1.0" manifest:version="1.2">
  <manifest:file-entry manifest:full-path="/" manifest:version="1.2" manifest:media-type="application/vnd.oasis.opendocument.text"/>
  <manifest:file-entry manifest:full-path="content.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="styles.xml" manifest:media-type="text/xml"/>
  <manifest:file-entry manifest:full-path="meta.xml" manifest:media-type="text/xml"/>
</manifest:manifest>`
	if err := writeZipFile(zw, "META-INF/manifest.xml", []byte(manifestXML)); err != nil {
		return nil, err
	}

	// 3. meta.xml
	metaXML := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-meta xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:dc="http://purl.org/dc/elements/1.1/"
  xmlns:meta="urn:oasis:names:tc:opendocument:xmlns:meta:1.0" office:version="1.2">
  <office:meta>
    <dc:title>%s</dc:title>
    <dc:creator>%s</dc:creator>
    <dc:language>vi-VN</dc:language>
    <meta:generator>GoNovelist Desktop Export Engine</meta:generator>
    <meta:creation-date>%s</meta:creation-date>
  </office:meta>
</office:document-meta>`,
		escapeXML(ms.Project.Title),
		escapeXML(ms.Project.Author),
		ms.GeneratedAt.Format(time.RFC3339),
	)
	if err := writeZipFile(zw, "meta.xml", []byte(metaXML)); err != nil {
		return nil, err
	}

	// 4. styles.xml (Định nghĩa phông chữ Tiếng Việt & kiểu đoạn văn cho LibreOffice Writer)
	stylesXML := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-styles xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
  xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" office:version="1.2">
  <office:font-face-decls>
    <style:font-face style:name="Noto Serif" svg:font-family="'Noto Serif', 'DejaVu Serif', 'Times New Roman'" xmlns:svg="urn:oasis:names:tc:opendocument:xmlns:svg-compatible:1.0"/>
  </office:font-face-decls>
  <office:styles>
    <style:default-style style:family="paragraph">
      <style:paragraph-properties fo:line-height="150%"/>
      <style:text-properties style:font-name="Noto Serif" fo:font-size="12pt" fo:language="vi" fo:country="VN"/>
    </style:default-style>
    <style:style style:name="DocTitle" style:family="paragraph">
      <style:paragraph-properties fo:text-align="center" fo:margin-top="0.5cm" fo:margin-bottom="0.3cm"/>
      <style:text-properties fo:font-size="24pt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="DocMeta" style:family="paragraph">
      <style:paragraph-properties fo:text-align="center" fo:margin-bottom="0.6cm"/>
      <style:text-properties fo:font-size="11pt" fo:font-style="italic" fo:color="#57534E"/>
    </style:style>
    <style:style style:name="Heading_20_1" style:display-name="Heading 1" style:family="paragraph">
      <style:paragraph-properties fo:margin-top="0.8cm" fo:margin-bottom="0.4cm" fo:keep-with-next="always"/>
      <style:text-properties fo:font-size="18pt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Heading_20_2" style:display-name="Heading 2" style:family="paragraph">
      <style:paragraph-properties fo:margin-top="0.6cm" fo:margin-bottom="0.3cm" fo:keep-with-next="always"/>
      <style:text-properties fo:font-size="14pt" fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="Heading_20_3" style:display-name="Heading 3" style:family="paragraph">
      <style:paragraph-properties fo:margin-top="0.4cm" fo:margin-bottom="0.2cm" fo:keep-with-next="always"/>
      <style:text-properties fo:font-size="11pt" fo:font-weight="bold" fo:color="#57534E"/>
    </style:style>
    <style:style style:name="Text_20_body" style:display-name="Text body" style:family="paragraph">
      <style:paragraph-properties fo:margin-top="0cm" fo:margin-bottom="0.25cm" fo:text-indent="1.0cm" fo:text-align="justify"/>
      <style:text-properties fo:font-size="12pt"/>
    </style:style>
    <style:style style:name="Block_20_Quote" style:display-name="Block Quote" style:family="paragraph">
      <style:paragraph-properties fo:margin-left="1.2cm" fo:margin-right="0.8cm" fo:margin-top="0.2cm" fo:margin-bottom="0.3cm" fo:padding-left="0.35cm" fo:border-left="2pt solid #8B3A2B"/>
      <style:text-properties fo:font-size="11.5pt" fo:font-style="italic" fo:color="#3F3C36"/>
    </style:style>
    <style:style style:name="SceneSeparator" style:family="paragraph">
      <style:paragraph-properties fo:text-align="center" fo:margin-top="0.4cm" fo:margin-bottom="0.4cm"/>
      <style:text-properties fo:font-size="11pt" fo:color="#78716C"/>
    </style:style>
    <style:style style:name="TBold" style:family="text">
      <style:text-properties fo:font-weight="bold"/>
    </style:style>
    <style:style style:name="TItalic" style:family="text">
      <style:text-properties fo:font-style="italic"/>
    </style:style>
    <style:style style:name="TBoldItalic" style:family="text">
      <style:text-properties fo:font-weight="bold" fo:font-style="italic"/>
    </style:style>
    <style:style style:name="TUnderline" style:family="text">
      <style:text-properties style:text-underline-style="solid" style:text-underline-width="auto" style:text-underline-color="font-color"/>
    </style:style>
    <style:style style:name="TBoldUnderline" style:family="text">
      <style:text-properties fo:font-weight="bold" style:text-underline-style="solid" style:text-underline-width="auto" style:text-underline-color="font-color"/>
    </style:style>
    <style:style style:name="TItalicUnderline" style:family="text">
      <style:text-properties fo:font-style="italic" style:text-underline-style="solid" style:text-underline-width="auto" style:text-underline-color="font-color"/>
    </style:style>
    <style:style style:name="TBoldItalicUnderline" style:family="text">
      <style:text-properties fo:font-weight="bold" fo:font-style="italic" style:text-underline-style="solid" style:text-underline-width="auto" style:text-underline-color="font-color"/>
    </style:style>
  </office:styles>
</office:document-styles>`
	if err := writeZipFile(zw, "styles.xml", []byte(stylesXML)); err != nil {
		return nil, err
	}

	// 5. content.xml (Nội dung chính của bản thảo với phân cấp Heading 1 / 2 / 3)
	var content strings.Builder
	content.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:style="urn:oasis:names:tc:opendocument:xmlns:style:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0"
  xmlns:fo="urn:oasis:names:tc:opendocument:xmlns:xsl-fo-compatible:1.0" office:version="1.2">
  <office:body>
    <office:text>
`)
	content.WriteString(fmt.Sprintf(`      <text:p text:style-name="DocTitle">%s</text:p>`+"\n", escapeXML(ms.Project.Title)))
	metaLine := fmt.Sprintf("Tác giả: %s | Thể loại: %s | %s (%d từ)",
		ms.Project.Author, ms.Project.Genre, ms.ScopeLabel, ms.TotalWords)
	content.WriteString(fmt.Sprintf(`      <text:p text:style-name="DocMeta">%s</text:p>`+"\n", escapeXML(metaLine)))

	if opts.IncludeSynopsis && strings.TrimSpace(ms.Project.Synopsis) != "" {
		content.WriteString(fmt.Sprintf(`      <text:p text:style-name="DocMeta">%s</text:p>`+"\n", escapeXML(strings.TrimSpace(ms.Project.Synopsis))))
	}

	for _, expAct := range ms.Acts {
		content.WriteString(fmt.Sprintf(`      <text:h text:style-name="Heading_20_1" text:outline-level="1">%s</text:h>`+"\n",
			escapeXML(expAct.Act.Title)))
		for _, expCh := range expAct.Chapters {
			content.WriteString(fmt.Sprintf(`      <text:h text:style-name="Heading_20_2" text:outline-level="2">%s</text:h>`+"\n",
				escapeXML(expCh.Chapter.Title)))
			for i, sc := range expCh.Scenes {
				if opts.IncludeSceneTitle && strings.TrimSpace(sc.Title) != "" {
					content.WriteString(fmt.Sprintf(`      <text:h text:style-name="Heading_20_3" text:outline-level="3">%s</text:h>`+"\n",
						escapeXML(sc.Title)))
				}
				blocks := ParseRichProseBlocks(sc.Content)
				content.WriteString(RenderRichBlocksODT(blocks))
				if i < len(expCh.Scenes)-1 {
					content.WriteString(`      <text:p text:style-name="SceneSeparator">* * *</text:p>` + "\n")
				}
			}
		}
	}

	content.WriteString(`    </office:text>
  </office:body>
</office:document-content>`)

	if err := writeZipFile(zw, "content.xml", []byte(content.String())); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ==================== 5. EPUB EBOOK (.epub) EXPORTER ====================

// ExportManuscriptEPUB đóng gói bản thảo thành tệp sách điện tử chuẩn EPUB 2/3 (.epub) hỗ trợ đầy đủ Tiếng Việt UTF-8.
func ExportManuscriptEPUB(ms *FilteredManuscript, opts ExportOptions) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// 1. Tệp mimetype bắt buộc đứng đầu tiên và không nén (zip.Store) theo chuẩn IDPF EPUB
	mimeHeader := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	}
	mimeWriter, err := zw.CreateHeader(mimeHeader)
	if err != nil {
		return nil, err
	}
	if _, err := mimeWriter.Write([]byte("application/epub+zip")); err != nil {
		return nil, err
	}

	// 2. META-INF/container.xml
	containerXML := `<?xml version="1.0" encoding="UTF-8"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	if err := writeZipFile(zw, "META-INF/container.xml", []byte(containerXML)); err != nil {
		return nil, err
	}

	// 3. OEBPS/style.css
	cssContent := `body {
  font-family: "Noto Serif", "DejaVu Serif", Georgia, "Times New Roman", serif;
  line-height: 1.75;
  color: #1C1B18;
  margin: 5%;
}
h1.book-title {
  font-size: 2em;
  text-align: center;
  margin-top: 1.5em;
  margin-bottom: 0.3em;
}
.book-meta {
  text-align: center;
  font-style: italic;
  color: #57534E;
  margin-bottom: 2em;
}
.synopsis {
  border-left: 3px solid #8B3A2B;
  padding-left: 1em;
  font-style: italic;
  margin: 1.5em 0;
}
h1.act-title {
  font-size: 1.5em;
  border-bottom: 1px solid #D6D0C4;
  padding-bottom: 0.3em;
  margin-top: 1.5em;
}
h2.chapter-title {
  font-size: 1.3em;
  margin-top: 1.2em;
  margin-bottom: 0.8em;
}
h3.scene-title {
  font-size: 0.95em;
  color: #78716C;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  margin-top: 1.2em;
}
p {
  text-indent: 1.4em;
  margin: 0.6em 0;
  text-align: justify;
}
blockquote {
  margin: 1em 1.4em;
  padding-left: 1em;
  border-left: 3px solid #8B3A2B;
  font-style: italic;
  color: #3F3C36;
}
blockquote p {
  text-indent: 0;
}
h4.sub-heading {
  font-size: 1.05em;
  font-weight: bold;
  margin-top: 1em;
  margin-bottom: 0.4em;
}
.underline-span {
  text-decoration: underline;
}
.scene-break {
  text-align: center;
  letter-spacing: 0.35em;
  color: #78716C;
  margin: 1.5em 0;
}`
	if err := writeZipFile(zw, "OEBPS/style.css", []byte(cssContent)); err != nil {
		return nil, err
	}

	// 4. Trang bìa trong OEBPS/title.xhtml
	var titleXHTML strings.Builder
	titleXHTML.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="vi" lang="vi">
<head>
  <meta charset="UTF-8"/>
  <title>` + escapeXML(ms.Project.Title) + `</title>
  <link rel="stylesheet" type="text/css" href="style.css"/>
</head>
<body>
  <h1 class="book-title">` + escapeXML(ms.Project.Title) + `</h1>
  <div class="book-meta">
    <div>Tác giả: ` + escapeXML(ms.Project.Author) + `</div>
    <div>Thể loại: ` + escapeXML(ms.Project.Genre) + `</div>
    <div>Phạm vi: ` + escapeXML(ms.ScopeLabel) + `</div>
  </div>
`)
	if opts.IncludeSynopsis && strings.TrimSpace(ms.Project.Synopsis) != "" {
		titleXHTML.WriteString(`  <div class="synopsis">` + escapeXML(strings.TrimSpace(ms.Project.Synopsis)) + `</div>` + "\n")
	}
	titleXHTML.WriteString(`</body>
</html>`)
	if err := writeZipFile(zw, "OEBPS/title.xhtml", []byte(titleXHTML.String())); err != nil {
		return nil, err
	}

	type epubChapterSection struct {
		ID       string
		FileName string
		ActTitle string
		ChTitle  string
	}
	var sections []epubChapterSection
	secCounter := 1

	for _, expAct := range ms.Acts {
		for _, expCh := range expAct.Chapters {
			secID := fmt.Sprintf("chap_%03d", secCounter)
			fileName := fmt.Sprintf("chapter_%03d.xhtml", secCounter)
			secCounter++

			var chBuf strings.Builder
			chBuf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xml:lang="vi" lang="vi">
<head>
  <meta charset="UTF-8"/>
  <title>` + escapeXML(expCh.Chapter.Title) + `</title>
  <link rel="stylesheet" type="text/css" href="style.css"/>
</head>
<body>
  <h1 class="act-title">` + escapeXML(expAct.Act.Title) + `</h1>
  <h2 class="chapter-title">` + escapeXML(expCh.Chapter.Title) + `</h2>
`)
			for i, sc := range expCh.Scenes {
				if opts.IncludeSceneTitle && strings.TrimSpace(sc.Title) != "" {
					chBuf.WriteString(`  <h3 class="scene-title">` + escapeXML(sc.Title) + `</h3>` + "\n")
				}
				blocks := ParseRichProseBlocks(sc.Content)
				chBuf.WriteString(RenderRichBlocksHTML(blocks, true, "  "))
				if i < len(expCh.Scenes)-1 {
					chBuf.WriteString(`  <div class="scene-break">* * *</div>` + "\n")
				}
			}
			chBuf.WriteString(`</body>
</html>`)

			if err := writeZipFile(zw, "OEBPS/"+fileName, []byte(chBuf.String())); err != nil {
				return nil, err
			}

			sections = append(sections, epubChapterSection{
				ID:       secID,
				FileName: fileName,
				ActTitle: expAct.Act.Title,
				ChTitle:  expCh.Chapter.Title,
			})
		}
	}

	// 5. OEBPS/nav.xhtml (Mục lục chuẩn EPUB 3)
	var navBuf strings.Builder
	navBuf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops" xml:lang="vi" lang="vi">
<head>
  <meta charset="UTF-8"/>
  <title>Mục lục</title>
  <link rel="stylesheet" type="text/css" href="style.css"/>
</head>
<body>
  <nav epub:type="toc" id="toc">
    <h1>Mục lục</h1>
    <ol>
      <li><a href="title.xhtml">Trang tiêu đề</a></li>
`)
	for _, sec := range sections {
		label := fmt.Sprintf("%s — %s", sec.ActTitle, sec.ChTitle)
		navBuf.WriteString(fmt.Sprintf(`      <li><a href="%s">%s</a></li>`+"\n", sec.FileName, escapeXML(label)))
	}
	navBuf.WriteString(`    </ol>
  </nav>
</body>
</html>`)
	if err := writeZipFile(zw, "OEBPS/nav.xhtml", []byte(navBuf.String())); err != nil {
		return nil, err
	}

	// 6. OEBPS/toc.ncx (Mục lục tương thích ngược EPUB 2)
	bookUID := fmt.Sprintf("urn:gonovelist:book:%d:%d", ms.Project.ID, ms.GeneratedAt.Unix())
	var ncxBuf strings.Builder
	ncxBuf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<ncx xmlns="http://www.daisy.org/z3986/2005/ncx/" version="2005-1">
  <head>
    <meta name="dtb:uid" content="` + bookUID + `"/>
    <meta name="dtb:depth" content="1"/>
    <meta name="dtb:totalPageCount" content="0"/>
    <meta name="dtb:maxPageNumber" content="0"/>
  </head>
  <docTitle><text>` + escapeXML(ms.Project.Title) + `</text></docTitle>
  <navMap>
    <navPoint id="nav_title" playOrder="1">
      <navLabel><text>Trang tiêu đề</text></navLabel>
      <content src="title.xhtml"/>
    </navPoint>
`)
	for idx, sec := range sections {
		label := fmt.Sprintf("%s — %s", sec.ActTitle, sec.ChTitle)
		ncxBuf.WriteString(fmt.Sprintf(`    <navPoint id="nav_%s" playOrder="%d">
      <navLabel><text>%s</text></navLabel>
      <content src="%s"/>
    </navPoint>
`, sec.ID, idx+2, escapeXML(label), sec.FileName))
	}
	ncxBuf.WriteString(`  </navMap>
</ncx>`)
	if err := writeZipFile(zw, "OEBPS/toc.ncx", []byte(ncxBuf.String())); err != nil {
		return nil, err
	}

	// 7. OEBPS/content.opf
	var opfBuf strings.Builder
	opfBuf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="BookId" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:identifier id="BookId">` + bookUID + `</dc:identifier>
    <dc:title>` + escapeXML(ms.Project.Title) + `</dc:title>
    <dc:creator>` + escapeXML(ms.Project.Author) + `</dc:creator>
    <dc:language>vi</dc:language>
    <meta property="dcterms:modified">` + ms.GeneratedAt.UTC().Format("2006-01-02T15:04:05Z") + `</meta>
  </metadata>
  <manifest>
    <item id="style" href="style.css" media-type="text/css"/>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/>
    <item id="title" href="title.xhtml" media-type="application/xhtml+xml"/>
`)
	for _, sec := range sections {
		opfBuf.WriteString(fmt.Sprintf(`    <item id="%s" href="%s" media-type="application/xhtml+xml"/>`+"\n", sec.ID, sec.FileName))
	}
	opfBuf.WriteString(`  </manifest>
  <spine toc="ncx">
    <itemref idref="title"/>
`)
	for _, sec := range sections {
		opfBuf.WriteString(fmt.Sprintf(`    <itemref idref="%s"/>`+"\n", sec.ID))
	}
	opfBuf.WriteString(`  </spine>
</package>`)
	if err := writeZipFile(zw, "OEBPS/content.opf", []byte(opfBuf.String())); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ==================== 6. PDF DOCUMENT (.pdf) EXPORTER (UNICODE TRUETYPE EMBEDDED) ====================

// ttfFontMetrics giữ bảng ánh xạ Unicode -> GlyphID và độ rộng ký tự từ phông chữ TrueType (.ttf) để nhúng vào PDF.
type ttfFontMetrics struct {
	rawTTF       []byte
	unitsPerEm   uint16
	ascent       int16
	descent      int16
	glyphWidths  []uint16
	unicodeToGID map[rune]uint16
}

// loadVietnameseTTFFont tải phông chữ TrueType hỗ trợ Tiếng Việt từ hệ thống hoặc từ phông chữ nhúng sẵn của Fyne.
func loadVietnameseTTFFont() (*ttfFontMetrics, error) {
	candidates := []string{
		os.Getenv("FYNE_FONT"),
		"/usr/share/fonts/TTF/DejaVuSerif.ttf",
		"/usr/share/fonts/TTF/DejaVuSans.ttf",
		"/usr/share/fonts/noto/NotoSerif-Regular.ttf",
		"/usr/share/fonts/noto/NotoSans-Regular.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSerif.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/noto/NotoSerif-Regular.ttf",
		"/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf",
		"/usr/share/fonts/liberation/LiberationSerif-Regular.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSerif-Regular.ttf",
	}

	for _, path := range candidates {
		if path == "" {
			continue
		}
		if data, err := os.ReadFile(path); err == nil && len(data) > 1024 {
			if parsed, err := parseTTFMetrics(data); err == nil {
				return parsed, nil
			}
		}
	}

	// Dự phòng an toàn 100%: Sử dụng phông chữ TrueType Unicode nhúng sẵn trong Fyne v2
	fyneFontBytes := theme.DefaultTextFont().Content()
	return parseTTFMetrics(fyneFontBytes)
}

func parseTTFMetrics(data []byte) (*ttfFontMetrics, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("dữ liệu TTF không hợp lệ")
	}
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	if len(data) < 12+numTables*16 {
		return nil, fmt.Errorf("bảng thư mục TTF bị cắt ngắn")
	}

	tables := make(map[string][2]uint32, numTables)
	for i := 0; i < numTables; i++ {
		pos := 12 + i*16
		tag := string(data[pos : pos+4])
		offset := binary.BigEndian.Uint32(data[pos+8 : pos+12])
		length := binary.BigEndian.Uint32(data[pos+12 : pos+16])
		if int(offset)+int(length) <= len(data) {
			tables[tag] = [2]uint32{offset, length}
		}
	}

	headLoc, ok := tables["head"]
	if !ok || headLoc[1] < 54 {
		return nil, fmt.Errorf("thiếu bảng head trong TTF")
	}
	unitsPerEm := binary.BigEndian.Uint16(data[headLoc[0]+18 : headLoc[0]+20])
	if unitsPerEm == 0 {
		unitsPerEm = 1000
	}

	hheaLoc, ok := tables["hhea"]
	if !ok || hheaLoc[1] < 36 {
		return nil, fmt.Errorf("thiếu bảng hhea trong TTF")
	}
	ascent := int16(binary.BigEndian.Uint16(data[hheaLoc[0]+4 : hheaLoc[0]+6]))
	descent := int16(binary.BigEndian.Uint16(data[hheaLoc[0]+6 : hheaLoc[0]+8]))
	numHMetrics := int(binary.BigEndian.Uint16(data[hheaLoc[0]+34 : hheaLoc[0]+36]))

	numGlyphs := numHMetrics
	if maxpLoc, ok := tables["maxp"]; ok && maxpLoc[1] >= 6 {
		ng := int(binary.BigEndian.Uint16(data[maxpLoc[0]+4 : maxpLoc[0]+6]))
		if ng > numGlyphs {
			numGlyphs = ng
		}
	}

	hmtxLoc, ok := tables["hmtx"]
	if !ok || int(hmtxLoc[1]) < numHMetrics*4 {
		return nil, fmt.Errorf("thiếu bảng hmtx trong TTF")
	}
	widths := make([]uint16, numGlyphs)
	lastWidth := uint16(600)
	for i := 0; i < numGlyphs; i++ {
		if i < numHMetrics {
			pos := int(hmtxLoc[0]) + i*4
			lastWidth = binary.BigEndian.Uint16(data[pos : pos+2])
		}
		widths[i] = lastWidth
	}

	cmapLoc, ok := tables["cmap"]
	if !ok || cmapLoc[1] < 4 {
		return nil, fmt.Errorf("thiếu bảng cmap trong TTF")
	}
	unicodeToGID := parseTTFCmap(data[cmapLoc[0] : cmapLoc[0]+cmapLoc[1]])

	return &ttfFontMetrics{
		rawTTF:       data,
		unitsPerEm:   unitsPerEm,
		ascent:       ascent,
		descent:      descent,
		glyphWidths:  widths,
		unicodeToGID: unicodeToGID,
	}, nil
}

func parseTTFCmap(cmap []byte) map[rune]uint16 {
	res := make(map[rune]uint16, 512)
	if len(cmap) < 4 {
		return res
	}
	numSubtables := int(binary.BigEndian.Uint16(cmap[2:4]))
	if len(cmap) < 4+numSubtables*8 {
		return res
	}

	var format4Offset uint32
	var format12Offset uint32

	for i := 0; i < numSubtables; i++ {
		rec := 4 + i*8
		platformID := binary.BigEndian.Uint16(cmap[rec : rec+2])
		encodingID := binary.BigEndian.Uint16(cmap[rec+2 : rec+4])
		subOffset := binary.BigEndian.Uint32(cmap[rec+4 : rec+8])
		if int(subOffset)+4 > len(cmap) {
			continue
		}
		if platformID == 0 || (platformID == 3 && (encodingID == 1 || encodingID == 10)) {
			format := binary.BigEndian.Uint16(cmap[subOffset : subOffset+2])
			if format == 12 && format12Offset == 0 {
				format12Offset = subOffset
			} else if format == 4 && format4Offset == 0 {
				format4Offset = subOffset
			}
		}
	}

	if format4Offset > 0 {
		parseCmapFormat4(cmap[format4Offset:], res)
	}
	if format12Offset > 0 {
		parseCmapFormat12(cmap[format12Offset:], res)
	}
	return res
}

func parseCmapFormat4(sub []byte, out map[rune]uint16) {
	if len(sub) < 16 {
		return
	}
	segCountX2 := int(binary.BigEndian.Uint16(sub[6:8]))
	segCount := segCountX2 / 2
	endCountPos := 14
	startCountPos := endCountPos + segCountX2 + 2
	idDeltaPos := startCountPos + segCountX2
	idRangeOffsetPos := idDeltaPos + segCountX2
	if len(sub) < idRangeOffsetPos+segCountX2 {
		return
	}

	for i := 0; i < segCount; i++ {
		endCode := int(binary.BigEndian.Uint16(sub[endCountPos+i*2 : endCountPos+i*2+2]))
		startCode := int(binary.BigEndian.Uint16(sub[startCountPos+i*2 : startCountPos+i*2+2]))
		idDelta := int(int16(binary.BigEndian.Uint16(sub[idDeltaPos+i*2 : idDeltaPos+i*2+2])))
		idRangeOffset := int(binary.BigEndian.Uint16(sub[idRangeOffsetPos+i*2 : idRangeOffsetPos+i*2+2]))
		if startCode == 0xFFFF {
			break
		}
		for code := startCode; code <= endCode; code++ {
			var gid uint16
			if idRangeOffset == 0 {
				gid = uint16((code + idDelta) & 0xFFFF)
			} else {
				roPos := idRangeOffsetPos + i*2 + idRangeOffset + (code-startCode)*2
				if roPos+2 <= len(sub) {
					g := int(binary.BigEndian.Uint16(sub[roPos : roPos+2]))
					if g != 0 {
						gid = uint16((g + idDelta) & 0xFFFF)
					}
				}
			}
			if gid != 0 {
				out[rune(code)] = gid
			}
		}
	}
}

func parseCmapFormat12(sub []byte, out map[rune]uint16) {
	if len(sub) < 16 {
		return
	}
	numGroups := int(binary.BigEndian.Uint32(sub[12:16]))
	if len(sub) < 16+numGroups*12 {
		return
	}
	for i := 0; i < numGroups; i++ {
		pos := 16 + i*12
		startCode := binary.BigEndian.Uint32(sub[pos : pos+4])
		endCode := binary.BigEndian.Uint32(sub[pos+4 : pos+8])
		startGID := binary.BigEndian.Uint32(sub[pos+8 : pos+12])
		if endCode > 0x2FFFF {
			continue
		}
		for c := startCode; c <= endCode; c++ {
			gid := uint16((startGID + (c - startCode)) & 0xFFFF)
			if gid != 0 {
				out[rune(c)] = gid
			}
		}
	}
}

func (m *ttfFontMetrics) glyphID(r rune) uint16 {
	if gid, ok := m.unicodeToGID[r]; ok {
		return gid
	}
	if gid, ok := m.unicodeToGID[' ']; ok {
		return gid
	}
	return 0
}

func (m *ttfFontMetrics) pdfWidth(gid uint16) int {
	if int(gid) < len(m.glyphWidths) {
		return int(m.glyphWidths[gid]) * 1000 / int(m.unitsPerEm)
	}
	return 600
}

func (m *ttfFontMetrics) measureStringPt(text string, fontSize float64) float64 {
	total := 0
	for i := 0; i < len(text); {
		r, w := utf8.DecodeRuneInString(text[i:])
		i += w
		total += m.pdfWidth(m.glyphID(r))
	}
	return float64(total) * fontSize / 1000.0
}

func (m *ttfFontMetrics) wrapTextPt(text string, fontSize float64, maxWidthPt float64) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if m.measureStringPt(candidate, fontSize) <= maxWidthPt {
			current = candidate
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	lines = append(lines, current)
	return lines
}

// ExportManuscriptPDF xuất bản thảo ra tệp PDF (.pdf) chuẩn A4 nhúng trực tiếp phông chữ TrueType Unicode Tiếng Việt.
func ExportManuscriptPDF(ms *FilteredManuscript, opts ExportOptions) ([]byte, error) {
	fontMetrics, err := loadVietnameseTTFFont()
	if err != nil {
		return nil, fmt.Errorf("không thể nạp phông chữ TrueType Tiếng Việt cho PDF: %w", err)
	}

	const (
		pageWidth  = 595.28
		pageHeight = 841.89
		marginLeft = 56.0
		marginTop  = 62.0
		marginBot  = 62.0
		usableW    = pageWidth - marginLeft*2
	)

	usedGIDToRune := make(map[uint16]rune)
	encodeHexGIDs := func(text string) string {
		var sb strings.Builder
		sb.WriteByte('<')
		for i := 0; i < len(text); {
			r, w := utf8.DecodeRuneInString(text[i:])
			i += w
			gid := fontMetrics.glyphID(r)
			usedGIDToRune[gid] = r
			sb.WriteString(fmt.Sprintf("%04X", gid))
		}
		sb.WriteByte('>')
		return sb.String()
	}

	var pages []strings.Builder
	curY := pageHeight - marginTop

	startNewPage := func() {
		pages = append(pages, strings.Builder{})
		curY = pageHeight - marginTop
	}
	startNewPage()

	ensureSpace := func(neededPt float64) {
		if curY-neededPt < marginBot {
			startNewPage()
		}
	}

	drawLine := func(text string, fontSize float64, indentPt float64, centered bool, gapAfter float64) {
		lineHeight := fontSize * 1.45
		ensureSpace(lineHeight)
		x := marginLeft + indentPt
		if centered {
			tw := fontMetrics.measureStringPt(text, fontSize)
			if tw < usableW {
				x = marginLeft + (usableW-tw)/2
			}
		}
		curY -= fontSize
		hexStr := encodeHexGIDs(text)
		pageIdx := len(pages) - 1
		pages[pageIdx].WriteString(fmt.Sprintf("BT /F1 %.2f Tf %.2f %.2f Td %s Tj ET\n", fontSize, x, curY, hexStr))
		curY -= (lineHeight - fontSize) + gapAfter
	}

	drawRichBlockPDF := func(block RichBlock, baseFontSize float64) {
		switch block.Kind {
		case RichBlockDivider:
			curY -= 4.0
			drawLine("* * *", 11.0, 0, true, 8.0)
			return
		case RichBlockHeading:
			curY -= 4.0
			ensureSpace(26.0)
			headingSpans := make([]RichSpan, len(block.Spans))
			for i, sp := range block.Spans {
				sp.Bold = true
				headingSpans[i] = sp
			}
			lines := wrapRichSpansPt(fontMetrics, headingSpans, baseFontSize+1.5, usableW)
			for idx, ln := range lines {
				gap := 0.0
				if idx == len(lines)-1 {
					gap = 5.0
				}
				lineHeight := (baseFontSize + 1.5) * 1.45
				ensureSpace(lineHeight)
				curY -= (baseFontSize + 1.5)
				pageIdx := len(pages) - 1
				curX := marginLeft
				for _, seg := range ln {
					hexStr := encodeHexGIDs(seg.Text)
					segW := fontMetrics.measureStringPt(seg.Text, baseFontSize+1.5)
					pages[pageIdx].WriteString(fmt.Sprintf("BT /F1 %.2f Tf 2 Tr 0.35 w %.2f %.2f Td %s Tj 0 Tr ET\n",
						baseFontSize+1.5, curX, curY, hexStr))
					curX += segW
				}
				curY -= (lineHeight - (baseFontSize + 1.5)) + gap
			}
			return
		case RichBlockQuote:
			quoteIndent := 22.0
			quoteSpans := make([]RichSpan, len(block.Spans))
			for i, sp := range block.Spans {
				sp.Italic = true
				quoteSpans[i] = sp
			}
			lines := wrapRichSpansPt(fontMetrics, quoteSpans, baseFontSize, usableW-quoteIndent-10.0)
			for idx, ln := range lines {
				gap := 0.0
				if idx == len(lines)-1 {
					gap = 6.0
				}
				lineHeight := baseFontSize * 1.45
				ensureSpace(lineHeight)
				topBarY := curY - 1.0
				curY -= baseFontSize
				botBarY := curY - 3.0
				pageIdx := len(pages) - 1
				// Vẽ thanh dọc bên trái cho khối Trích dẫn (Blockquote)
				barX := marginLeft + 12.0
				pages[pageIdx].WriteString(fmt.Sprintf("q 1.5 w 0.55 0.23 0.17 RG %.2f %.2f m %.2f %.2f l S Q\n",
					barX, topBarY, barX, botBarY))

				curX := marginLeft + quoteIndent
				for _, seg := range ln {
					hexStr := encodeHexGIDs(seg.Text)
					segW := fontMetrics.measureStringPt(seg.Text, baseFontSize)
					if seg.Bold {
						pages[pageIdx].WriteString(fmt.Sprintf("BT /F1 %.2f Tf 2 Tr 0.32 w 1 0 0.20 1 %.2f %.2f Tm %s Tj 0 Tr ET\n",
							baseFontSize, curX, curY, hexStr))
					} else {
						pages[pageIdx].WriteString(fmt.Sprintf("BT /F1 %.2f Tf 1 0 0.20 1 %.2f %.2f Tm %s Tj ET\n",
							baseFontSize, curX, curY, hexStr))
					}
					if seg.Underline {
						pages[pageIdx].WriteString(fmt.Sprintf("q 0.6 w %.2f %.2f m %.2f %.2f l S Q\n",
							curX, curY-2.0, curX+segW, curY-2.0))
					}
					curX += segW
				}
				curY -= (lineHeight - baseFontSize) + gap
			}
			return
		default: // RichBlockParagraph
			firstLineIndent := 18.0
			lines := wrapRichSpansFirstLinePt(fontMetrics, block.Spans, baseFontSize, usableW, firstLineIndent)
			for idx, ln := range lines {
				indent := 0.0
				if idx == 0 {
					indent = firstLineIndent
				}
				gap := 0.0
				if idx == len(lines)-1 {
					gap = 5.0
				}
				lineHeight := baseFontSize * 1.45
				ensureSpace(lineHeight)
				curY -= baseFontSize
				pageIdx := len(pages) - 1
				curX := marginLeft + indent
				for _, seg := range ln {
					hexStr := encodeHexGIDs(seg.Text)
					segW := fontMetrics.measureStringPt(seg.Text, baseFontSize)
					if seg.Bold && seg.Italic {
						pages[pageIdx].WriteString(fmt.Sprintf("BT /F1 %.2f Tf 2 Tr 0.32 w 1 0 0.20 1 %.2f %.2f Tm %s Tj 0 Tr ET\n",
							baseFontSize, curX, curY, hexStr))
					} else if seg.Bold {
						pages[pageIdx].WriteString(fmt.Sprintf("BT /F1 %.2f Tf 2 Tr 0.32 w %.2f %.2f Td %s Tj 0 Tr ET\n",
							baseFontSize, curX, curY, hexStr))
					} else if seg.Italic {
						pages[pageIdx].WriteString(fmt.Sprintf("BT /F1 %.2f Tf 1 0 0.20 1 %.2f %.2f Tm %s Tj ET\n",
							baseFontSize, curX, curY, hexStr))
					} else {
						pages[pageIdx].WriteString(fmt.Sprintf("BT /F1 %.2f Tf %.2f %.2f Td %s Tj ET\n",
							baseFontSize, curX, curY, hexStr))
					}
					if seg.Underline {
						pages[pageIdx].WriteString(fmt.Sprintf("q 0.6 w %.2f %.2f m %.2f %.2f l S Q\n",
							curX, curY-2.0, curX+segW, curY-2.0))
					}
					curX += segW
				}
				curY -= (lineHeight - baseFontSize) + gap
			}
		}
	}

	drawWrappedBlock := func(text string, fontSize float64, firstLineIndent float64, centered bool, blockGap float64) {
		lines := fontMetrics.wrapTextPt(text, fontSize, usableW-firstLineIndent)
		for idx, ln := range lines {
			indent := 0.0
			if idx == 0 {
				indent = firstLineIndent
			}
			gap := 0.0
			if idx == len(lines)-1 {
				gap = blockGap
			}
			drawLine(ln, fontSize, indent, centered, gap)
		}
	}

	// Trang tiêu đề / Phần mở đầu tài liệu PDF
	drawWrappedBlock(strings.ToUpper(ms.Project.Title), 20.0, 0, true, 8.0)
	if ms.Project.Author != "" {
		drawWrappedBlock("Tác giả: "+ms.Project.Author, 11.5, 0, true, 4.0)
	}
	metaInfo := fmt.Sprintf("Thể loại: %s  •  %s (%d từ)", ms.Project.Genre, ms.ScopeLabel, ms.TotalWords)
	drawWrappedBlock(metaInfo, 10.5, 0, true, 14.0)

	if opts.IncludeSynopsis && strings.TrimSpace(ms.Project.Synopsis) != "" {
		drawWrappedBlock("Tóm tắt: "+strings.TrimSpace(ms.Project.Synopsis), 10.5, 0, false, 16.0)
	}

	// Nội dung Hồi -> Chương -> Cảnh (Bảo toàn đầy đủ định dạng Rich Text: In đậm, In nghiêng, Gạch chân, Trích dẫn)
	for _, expAct := range ms.Acts {
		curY -= 10.0
		ensureSpace(45.0)
		drawWrappedBlock(strings.ToUpper(expAct.Act.Title), 15.0, 0, false, 8.0)

		for _, expCh := range expAct.Chapters {
			curY -= 6.0
			ensureSpace(36.0)
			drawWrappedBlock(expCh.Chapter.Title, 13.0, 0, false, 6.0)

			for i, sc := range expCh.Scenes {
				if opts.IncludeSceneTitle && strings.TrimSpace(sc.Title) != "" {
					ensureSpace(28.0)
					drawWrappedBlock("["+sc.Title+"]", 10.5, 0, false, 4.0)
				}
				richBlocks := ParseRichProseBlocks(sc.Content)
				for _, blk := range richBlocks {
					drawRichBlockPDF(blk, 11.5)
				}
				if i < len(expCh.Scenes)-1 {
					curY -= 4.0
					drawLine("* * *", 11.0, 0, true, 8.0)
				}
			}
		}
	}

	// Đánh số trang ở chân mỗi trang PDF
	totalPages := len(pages)
	for i := 0; i < totalPages; i++ {
		footer := fmt.Sprintf("Trang %d / %d — %s", i+1, totalPages, ms.Project.Title)
		fw := fontMetrics.measureStringPt(footer, 9.0)
		fx := marginLeft + (usableW-fw)/2
		hexFooter := encodeHexGIDs(footer)
		pages[i].WriteString(fmt.Sprintf("BT /F1 9.00 Tf %.2f %.2f Td %s Tj ET\n", fx, 32.0, hexFooter))
	}

	return assemblePDFDocument(fontMetrics, usedGIDToRune, pages)
}

func assemblePDFDocument(fontMetrics *ttfFontMetrics, usedGIDToRune map[uint16]rune, pages []strings.Builder) ([]byte, error) {
	// Nén luồng phông chữ TrueType bằng zlib (FlateDecode)
	var compressedTTF bytes.Buffer
	zw := zlib.NewWriter(&compressedTTF)
	if _, err := zw.Write(fontMetrics.rawTTF); err != nil {
		return nil, err
	}
	_ = zw.Close()

	// Danh sách GID đã dùng để tạo mảng độ rộng /W và bảng ánh xạ /ToUnicode
	gids := make([]int, 0, len(usedGIDToRune))
	for gid := range usedGIDToRune {
		gids = append(gids, int(gid))
	}
	sort.Ints(gids)

	var wArray strings.Builder
	wArray.WriteString("[ ")
	for _, g := range gids {
		wArray.WriteString(fmt.Sprintf("%d [%d] ", g, fontMetrics.pdfWidth(uint16(g))))
	}
	wArray.WriteString("]")

	// Luồng CMap /ToUnicode giúp PDF hỗ trợ tìm kiếm và sao chép văn bản Tiếng Việt UTF-8
	var cmapStream strings.Builder
	cmapStream.WriteString(`/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF>
endcodespacerange
`)
	chunkSize := 100
	for i := 0; i < len(gids); i += chunkSize {
		end := i + chunkSize
		if end > len(gids) {
			end = len(gids)
		}
		cmapStream.WriteString(fmt.Sprintf("%d beginbfchar\n", end-i))
		for _, g := range gids[i:end] {
			r := usedGIDToRune[uint16(g)]
			cmapStream.WriteString(fmt.Sprintf("<%04X> <%04X>\n", g, uint16(r)))
		}
		cmapStream.WriteString("endbfchar\n")
	}
	cmapStream.WriteString(`endcmap
CMapName currentdict /CMap defineresource pop
end
end`)
	cmapBytes := []byte(cmapStream.String())

	// Cấu trúc các đối tượng PDF:
	// 1: Catalog
	// 2: Pages
	// 3: Type0 Font
	// 4: CIDFontType2
	// 5: FontDescriptor
	// 6: FontFile2 (TrueType stream)
	// 7: ToUnicode CMap stream
	// 8..8+2*N-1: Page Object & Content Stream Object cho từng trang
	numPages := len(pages)
	totalObjects := 7 + numPages*2
	offsets := make([]int, totalObjects+1)

	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")

	writeObj := func(id int, body string) {
		offsets[id] = pdf.Len()
		pdf.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", id, body))
	}

	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")

	var kids strings.Builder
	kids.WriteString("[ ")
	for i := 0; i < numPages; i++ {
		pageObjID := 8 + i*2
		kids.WriteString(fmt.Sprintf("%d 0 R ", pageObjID))
	}
	kids.WriteString("]")
	writeObj(2, fmt.Sprintf("<< /Type /Pages /Kids %s /Count %d >>", kids.String(), numPages))

	writeObj(3, "<< /Type /Font /Subtype /Type0 /BaseFont /GoNovelistVN /Encoding /Identity-H /DescendantFonts [4 0 R] /ToUnicode 7 0 R >>")

	writeObj(4, fmt.Sprintf("<< /Type /Font /Subtype /CIDFontType2 /BaseFont /GoNovelistVN /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> /FontDescriptor 5 0 R /CIDToGIDMap /Identity /DW 600 /W %s >>", wArray.String()))

	ascent := int(fontMetrics.ascent) * 1000 / int(fontMetrics.unitsPerEm)
	descent := int(fontMetrics.descent) * 1000 / int(fontMetrics.unitsPerEm)
	writeObj(5, fmt.Sprintf("<< /Type /FontDescriptor /FontName /GoNovelistVN /Flags 32 /FontBBox [-500 %d 1500 %d] /ItalicAngle 0 /Ascent %d /Descent %d /CapHeight 700 /StemV 80 /FontFile2 6 0 R >>",
		descent, ascent, ascent, descent))

	// Đối tượng 6: Luồng dữ liệu TrueType đã nén
	offsets[6] = pdf.Len()
	pdf.WriteString(fmt.Sprintf("6 0 obj\n<< /Length %d /Length1 %d /Filter /FlateDecode >>\nstream\n",
		compressedTTF.Len(), len(fontMetrics.rawTTF)))
	pdf.Write(compressedTTF.Bytes())
	pdf.WriteString("\nendstream\nendobj\n")

	// Đối tượng 7: Luồng ToUnicode CMap
	offsets[7] = pdf.Len()
	pdf.WriteString(fmt.Sprintf("7 0 obj\n<< /Length %d >>\nstream\n", len(cmapBytes)))
	pdf.Write(cmapBytes)
	pdf.WriteString("\nendstream\nendobj\n")

	// Các trang và luồng nội dung trang
	for i := 0; i < numPages; i++ {
		pageObjID := 8 + i*2
		contentObjID := pageObjID + 1
		pageStream := []byte(pages[i].String())

		writeObj(pageObjID, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595.28 841.89] /Resources << /Font << /F1 3 0 R >> >> /Contents %d 0 R >>", contentObjID))

		offsets[contentObjID] = pdf.Len()
		pdf.WriteString(fmt.Sprintf("%d 0 obj\n<< /Length %d >>\nstream\n", contentObjID, len(pageStream)))
		pdf.Write(pageStream)
		pdf.WriteString("endstream\nendobj\n")
	}

	// Bảng tham chiếu chéo xref
	xrefPos := pdf.Len()
	pdf.WriteString(fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", totalObjects+1))
	for i := 1; i <= totalObjects; i++ {
		pdf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	pdf.WriteString(fmt.Sprintf("trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", totalObjects+1, xrefPos))

	return pdf.Bytes(), nil
}

// ==================== DIALOG CẤU HÌNH PHẠM VI & XUẤT BẢN (FYNE V2 UI) ====================

// ShowExportDialog mở hộp thoại cấu hình Phạm vi xuất bản (Toàn bộ tác phẩm, Theo Hồi chỉ định, Chương/Cảnh hiện tại) và lưu tệp.
func (ui *NovelistUI) ShowExportDialog(initialFormat ExportFormat) {
	ui.editorPanel.FlushPendingSave()

	acts, err := ui.store.ListActs(ui.activeProject.ID)
	if err != nil {
		ui.showErrorDialog(err)
		return
	}

	// Xác định mục đang được chọn trên cây phân cấp Hồi -> Chương -> Cảnh
	currentNode, hasCurrentNode := ui.nodeMeta[ui.selectedUID]
	if !hasCurrentNode && ui.editorPanel.activeScene != nil {
		currentNode = HierarchyNode{
			UID:        MakeNodeUID("scene", ui.editorPanel.activeScene.ID),
			Kind:       "scene",
			DatabaseID: ui.editorPanel.activeScene.ID,
			Title:      ui.editorPanel.activeScene.Title,
		}
		hasCurrentNode = true
	}

	// 1. Chọn định dạng xuất bản
	formatOptions := []string{
		FormatLabel(ExportFormatTXT),
		FormatLabel(ExportFormatODT),
		FormatLabel(ExportFormatPDF),
		FormatLabel(ExportFormatEPUB),
		FormatLabel(ExportFormatMarkdown),
		FormatLabel(ExportFormatHTML),
	}
	labelToFormat := map[string]ExportFormat{
		FormatLabel(ExportFormatTXT):      ExportFormatTXT,
		FormatLabel(ExportFormatODT):      ExportFormatODT,
		FormatLabel(ExportFormatPDF):      ExportFormatPDF,
		FormatLabel(ExportFormatEPUB):     ExportFormatEPUB,
		FormatLabel(ExportFormatMarkdown): ExportFormatMarkdown,
		FormatLabel(ExportFormatHTML):     ExportFormatHTML,
	}

	selectedFormat := initialFormat
	formatSelect := widget.NewSelect(formatOptions, nil)
	formatSelect.SetSelected(FormatLabel(selectedFormat))

	// 2. Chọn Phạm vi xuất bản (Granular Export Scope)
	const (
		scopeLabelAllBook     = "Toàn bộ tác phẩm (Tất cả Hồi, Chương và Cảnh)"
		scopeLabelSelectedAct = "Theo Hồi chỉ định (Chọn một Hồi cụ thể)"
		scopeLabelCurrentNode = "Chương/Cảnh hiện tại (Theo mục đang chọn trên cây)"
	)

	actLabels := make([]string, 0, len(acts))
	actLabelToID := make(map[string]int64, len(acts))
	for _, a := range acts {
		lbl := a.Title
		actLabels = append(actLabels, lbl)
		actLabelToID[lbl] = a.ID
	}

	actSelect := widget.NewSelect(actLabels, nil)
	if len(actLabels) > 0 {
		actSelect.SetSelected(actLabels[0])
	}
	actSelect.Disable()

	// Nhãn hiển thị chi tiết Chương/Cảnh hiện tại
	currentNodeDesc := "Chưa chọn mục nào trên cây (sẽ xuất toàn bộ tác phẩm)"
	if hasCurrentNode {
		switch currentNode.Kind {
		case "act":
			currentNodeDesc = fmt.Sprintf("Đang chọn Hồi: %q", currentNode.Title)
		case "chapter":
			currentNodeDesc = fmt.Sprintf("Đang chọn Chương: %q (xuất toàn bộ cảnh trong chương)", currentNode.Title)
		case "scene":
			currentNodeDesc = fmt.Sprintf("Đang chọn Cảnh: %q", currentNode.Title)
		}
	}
	currentNodeInfoLabel := widget.NewLabel(currentNodeDesc)
	currentNodeInfoLabel.Wrapping = fyne.TextWrapWord

	// Cho phép chọn xuất cả Chương chứa Cảnh hiện tại hoặc chỉ riêng Cảnh đang chọn
	currentSubScopeOptions := []string{
		"Xuất Chương chứa mục đang chọn",
		"Chỉ xuất riêng Cảnh đang chọn",
	}
	currentSubScopeSelect := widget.NewSelect(currentSubScopeOptions, nil)
	currentSubScopeSelect.SetSelected(currentSubScopeOptions[0])
	if !hasCurrentNode || currentNode.Kind != "scene" {
		currentSubScopeSelect.Hide()
	}

	// Ô nhập tên tệp đích (hỗ trợ gõ Tiếng Việt hoặc chọn qua hộp thoại tệp)
	pathEntry := NewVietEntry()
	buildSuggestedPath := func(fmtType ExportFormat, scopeChoice string) string {
		base := sanitizeFileName(ui.activeProject.Title)
		suffix := "_toan_bo"
		if scopeChoice == scopeLabelSelectedAct && actSelect.Selected != "" {
			suffix = "_" + sanitizeFileName(actSelect.Selected)
		} else if scopeChoice == scopeLabelCurrentNode && hasCurrentNode {
			suffix = "_" + sanitizeFileName(currentNode.Title)
		}
		return base + suffix + FormatExtension(fmtType)
	}

	scopeRadio := widget.NewRadioGroup([]string{
		scopeLabelAllBook,
		scopeLabelSelectedAct,
		scopeLabelCurrentNode,
	}, nil)
	scopeRadio.SetSelected(scopeLabelAllBook)

	updatePathSuggestion := func() {
		pathEntry.SetText(buildSuggestedPath(selectedFormat, scopeRadio.Selected))
	}

	scopeRadio.OnChanged = func(val string) {
		if val == scopeLabelSelectedAct {
			actSelect.Enable()
		} else {
			actSelect.Disable()
		}
		updatePathSuggestion()
	}
	actSelect.OnChanged = func(_ string) {
		updatePathSuggestion()
	}
	formatSelect.OnChanged = func(lbl string) {
		if f, ok := labelToFormat[lbl]; ok {
			selectedFormat = f
			updatePathSuggestion()
		}
	}
	updatePathSuggestion()

	// Tùy chọn bổ sung
	includeSynopsisCheck := widget.NewCheck("Đính kèm phần Tóm tắt tác phẩm ở đầu bản thảo", nil)
	includeSynopsisCheck.SetChecked(true)

	includeSceneTitleCheck := widget.NewCheck("Hiển thị tiêu đề từng Cảnh trong Chương", nil)
	includeSceneTitleCheck.SetChecked(true)

	// Nút mở hộp thoại chọn nơi lưu tệp có bộ lọc đuôi tệp Tiếng Việt
	browseBtn := widget.NewButtonWithIcon("Chọn nơi lưu tệp...", theme.FolderOpenIcon(), func() {
		ext := FormatExtension(selectedFormat)
		fd := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
			if err != nil {
				ui.showErrorDialog(err)
				return
			}
			if writer == nil {
				return
			}
			selectedPath := writer.URI().Path()
			_ = writer.Close()
			if !strings.HasSuffix(strings.ToLower(selectedPath), ext) {
				selectedPath += ext
			}
			pathEntry.SetText(selectedPath)
		}, ui.window)
		attachWindowRefreshOnClose(fd, ui.window)
		fd.SetFileName(filepath.Base(pathEntry.Text))
		fd.SetFilter(storage.NewExtensionFileFilter([]string{ext}))
		fd.Show()
	})

	formContent := container.NewVBox(
		widget.NewLabelWithStyle("Cấu Hình Xuất Bản Thảo Đa Định Dạng", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewForm(
			widget.NewFormItem("Định dạng xuất bản", formatSelect),
			widget.NewFormItem("Phạm vi xuất bản", scopeRadio),
			widget.NewFormItem("Chọn Hồi chỉ định", actSelect),
			widget.NewFormItem("Mục đang chọn trên cây", container.NewVBox(currentNodeInfoLabel, currentSubScopeSelect)),
		),
		widget.NewSeparator(),
		includeSynopsisCheck,
		includeSceneTitleCheck,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Đường dẫn tệp xuất bản:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, browseBtn, pathEntry),
	)

	var exportPopup dialog.Dialog
	cancelBtn := widget.NewButton("Hủy", func() {
		exportPopup.Hide()
	})
	confirmBtn := widget.NewButtonWithIcon("Xuất bản ngay", theme.DocumentSaveIcon(), func() {
		opts := ExportOptions{
			Format:            selectedFormat,
			Scope:             ExportScopeAllBook,
			IncludeSynopsis:   includeSynopsisCheck.Checked,
			IncludeSceneTitle: includeSceneTitleCheck.Checked,
			OutputPath:        strings.TrimSpace(pathEntry.Text),
		}

		switch scopeRadio.Selected {
		case scopeLabelSelectedAct:
			opts.Scope = ExportScopeSelectedAct
			opts.SelectedActID = actLabelToID[actSelect.Selected]
		case scopeLabelCurrentNode:
			if hasCurrentNode {
				opts.Scope = ExportScopeCurrentNode
				opts.CurrentNodeKind = currentNode.Kind
				switch currentNode.Kind {
				case "act":
					opts.SelectedActID = currentNode.DatabaseID
				case "chapter":
					opts.SelectedChapterID = currentNode.DatabaseID
				case "scene":
					if currentSubScopeSelect.Selected == currentSubScopeOptions[0] && ui.editorPanel.activeScene != nil {
						opts.CurrentNodeKind = "chapter"
						opts.SelectedChapterID = ui.editorPanel.activeScene.ChapterID
					} else {
						opts.SelectedSceneID = currentNode.DatabaseID
					}
				}
			}
		}

		if opts.OutputPath == "" {
			opts.OutputPath = buildSuggestedPath(opts.Format, scopeRadio.Selected)
		}
		ext := FormatExtension(opts.Format)
		if !strings.HasSuffix(strings.ToLower(opts.OutputPath), ext) {
			opts.OutputPath += ext
		}

		data, ms, err := ui.store.ExportManuscriptBytes(ui.activeProject, opts)
		if err != nil {
			ui.showErrorDialog(err)
			return
		}
		if err := os.WriteFile(opts.OutputPath, data, 0644); err != nil {
			ui.showErrorDialog(err)
			return
		}

		exportPopup.Hide()
		if ui.statusFooter != nil {
			ui.statusFooter.SetText(fmt.Sprintf("Đã xuất bản thành công (%s).", ms.ScopeLabel))
		}
		ui.showInformationDialog(
			"Xuất bản thảo thành công",
			fmt.Sprintf(
				"Định dạng: %s\nPhạm vi xuất bản: %s\nSố cảnh đã xuất: %d cảnh (%d từ)\nTệp đích: %s",
				FormatLabel(opts.Format),
				ms.ScopeLabel,
				ms.TotalScenes,
				ms.TotalWords,
				opts.OutputPath,
			),
		)
	})
	confirmBtn.Importance = widget.HighImportance

	footerButtons := container.NewHBox(cancelBtn, confirmBtn)
	dialogBody := container.NewBorder(
		nil,
		container.NewPadded(container.NewCenter(footerButtons)),
		nil,
		nil,
		container.NewPadded(formContent),
	)

	exportPopup = dialog.NewCustomWithoutButtons("Xuất Bản Thảo — "+ui.activeProject.Title, dialogBody, ui.window)
	ui.attachLayoutRefreshOnClose(exportPopup)
	exportPopup.Resize(fyne.NewSize(660, 540))
	exportPopup.Show()
}

// ==================== HELPER UTILITIES ====================

func splitParagraphs(content string) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	normalized := strings.ReplaceAll(trimmed, "\r\n", "\n")
	rawBlocks := strings.Split(normalized, "\n\n")
	var out []string
	for _, blk := range rawBlocks {
		lines := strings.Split(blk, "\n")
		for _, ln := range lines {
			clean := strings.TrimSpace(ln)
			if clean != "" {
				out = append(out, clean)
			}
		}
	}
	return out
}

func escapeXML(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

func writeZipFile(zw *zip.Writer, name string, data []byte) error {
	header := &zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: time.Now(),
	}
	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ==================== BỘ PHÂN TÍCH ĐỊNH DẠNG VĂN BẢN PHONG PHÚ (RICH TEXT AST PARSER) ====================

// RichBlockKind phân loại khối văn bản trong cảnh (Đoạn văn thường, Trích dẫn, Tiêu đề phụ, Ngắt cảnh).
type RichBlockKind string

const (
	RichBlockParagraph RichBlockKind = "paragraph"
	RichBlockQuote     RichBlockKind = "blockquote"
	RichBlockHeading   RichBlockKind = "heading"
	RichBlockDivider   RichBlockKind = "divider"
)

// RichSpan đại diện cho một phân đoạn văn bản nội dòng kèm thuộc tính In đậm, In nghiêng, Gạch chân.
type RichSpan struct {
	Text      string
	Bold      bool
	Italic    bool
	Underline bool
}

// RichBlock đại diện cho một khối đoạn văn đã được phân tích cấu trúc Rich Text.
type RichBlock struct {
	Kind  RichBlockKind
	Raw   string
	Spans []RichSpan
}

// ParseRichProseBlocks phân tích nội dung cảnh thành danh sách các khối RichBlock
// (nhận diện đoạn trích dẫn "> ...", tiêu đề phụ "### ...", dấu ngắt cảnh "* * *" và các thẻ inline).
func ParseRichProseBlocks(content string) []RichBlock {
	rawParagraphs := splitParagraphs(content)
	blocks := make([]RichBlock, 0, len(rawParagraphs))

	for _, p := range rawParagraphs {
		clean := strings.TrimSpace(p)
		if clean == "" {
			continue
		}

		// 1. Dấu ngắt cảnh (* * *, ***, ---)
		if clean == "* * *" || clean == "***" || clean == "---" {
			blocks = append(blocks, RichBlock{
				Kind: RichBlockDivider,
				Raw:  "* * *",
			})
			continue
		}

		// 2. Khối Trích dẫn (Blockquote: bắt đầu bằng "> " hoặc thẻ <blockquote>...</blockquote>)
		if strings.HasPrefix(clean, ">") {
			quoteBody := strings.TrimSpace(strings.TrimPrefix(clean, ">"))
			blocks = append(blocks, RichBlock{
				Kind:  RichBlockQuote,
				Raw:   quoteBody,
				Spans: ParseInlineRichSpans(quoteBody),
			})
			continue
		}
		if strings.HasPrefix(strings.ToLower(clean), "<blockquote>") && strings.HasSuffix(strings.ToLower(clean), "</blockquote>") {
			quoteBody := strings.TrimSpace(clean[len("<blockquote>") : len(clean)-len("</blockquote>")])
			blocks = append(blocks, RichBlock{
				Kind:  RichBlockQuote,
				Raw:   quoteBody,
				Spans: ParseInlineRichSpans(quoteBody),
			})
			continue
		}

		// 3. Tiêu đề phụ trong cảnh (#, ##, ###, ####)
		if strings.HasPrefix(clean, "#") {
			trimmedHashes := strings.TrimLeft(clean, "#")
			if strings.HasPrefix(trimmedHashes, " ") {
				headingBody := strings.TrimSpace(trimmedHashes)
				blocks = append(blocks, RichBlock{
					Kind:  RichBlockHeading,
					Raw:   headingBody,
					Spans: ParseInlineRichSpans(headingBody),
				})
				continue
			}
		}

		// 4. Đoạn văn xuôi thông thường
		blocks = append(blocks, RichBlock{
			Kind:  RichBlockParagraph,
			Raw:   clean,
			Spans: ParseInlineRichSpans(clean),
		})
	}

	return blocks
}

// ParseInlineRichSpans phân tích cú pháp định dạng nội dòng (Inline Rich Text):
//   - In đậm + In nghiêng: ***chữ***
//   - In đậm: **chữ**, __chữ__, <b>chữ</b>, <strong>chữ</strong>
//   - In nghiêng: *chữ*, _chữ_, <i>chữ</i>, <em>chữ</em>
//   - Gạch chân / Ghi chú: <u>chữ</u>, ++chữ++, <ins>chữ</ins>
func ParseInlineRichSpans(input string) []RichSpan {
	if input == "" {
		return nil
	}

	var spans []RichSpan
	var buf strings.Builder
	bold := false
	italic := false
	underline := false

	flushBuf := func() {
		if buf.Len() > 0 {
			spans = append(spans, RichSpan{
				Text:      buf.String(),
				Bold:      bold,
				Italic:    italic,
				Underline: underline,
			})
			buf.Reset()
		}
	}

	i := 0
	n := len(input)
	for i < n {
		rem := input[i:]
		lowerRem := strings.ToLower(rem)

		switch {
		case strings.HasPrefix(rem, "***"):
			flushBuf()
			bold = !bold
			italic = !italic
			i += 3
			continue
		case strings.HasPrefix(rem, "**") || strings.HasPrefix(rem, "__"):
			flushBuf()
			bold = !bold
			i += 2
			continue
		case strings.HasPrefix(rem, "++"):
			flushBuf()
			underline = !underline
			i += 2
			continue
		case strings.HasPrefix(lowerRem, "<b>"):
			flushBuf()
			bold = true
			i += 3
			continue
		case strings.HasPrefix(lowerRem, "</b>"):
			flushBuf()
			bold = false
			i += 4
			continue
		case strings.HasPrefix(lowerRem, "<strong>"):
			flushBuf()
			bold = true
			i += 8
			continue
		case strings.HasPrefix(lowerRem, "</strong>"):
			flushBuf()
			bold = false
			i += 9
			continue
		case strings.HasPrefix(lowerRem, "<i>"):
			flushBuf()
			italic = true
			i += 3
			continue
		case strings.HasPrefix(lowerRem, "</i>"):
			flushBuf()
			italic = false
			i += 4
			continue
		case strings.HasPrefix(lowerRem, "<em>"):
			flushBuf()
			italic = true
			i += 4
			continue
		case strings.HasPrefix(lowerRem, "</em>"):
			flushBuf()
			italic = false
			i += 5
			continue
		case strings.HasPrefix(lowerRem, "<u>"):
			flushBuf()
			underline = true
			i += 3
			continue
		case strings.HasPrefix(lowerRem, "</u>"):
			flushBuf()
			underline = false
			i += 4
			continue
		case strings.HasPrefix(lowerRem, "<ins>"):
			flushBuf()
			underline = true
			i += 5
			continue
		case strings.HasPrefix(lowerRem, "</ins>"):
			flushBuf()
			underline = false
			i += 6
			continue
		case rem[0] == '*' || rem[0] == '_':
			flushBuf()
			italic = !italic
			i++
			continue
		}

		r, width := utf8.DecodeRuneInString(rem)
		buf.WriteRune(r)
		i += width
	}
	flushBuf()

	if len(spans) == 0 {
		return []RichSpan{{Text: input}}
	}
	return spans
}

// ConvertRichProseToFyneMarkdown chuyển đổi văn bản có chứa thẻ <u>, <b>, <i> sang Markdown chuẩn cho widget.RichText của Fyne.
func ConvertRichProseToFyneMarkdown(raw string) string {
	replacer := strings.NewReplacer(
		"<b>", "**", "</b>", "**",
		"<strong>", "**", "</strong>", "**",
		"<i>", "*", "</i>", "*",
		"<em>", "*", "</em>", "*",
		"<u>", "_", "</u>", "_",
	)
	return replacer.Replace(raw)
}

// NormalizeRichProseToMarkdown chuẩn hóa văn bản hỗn hợp (Markdown + thẻ HTML inline) về cú pháp Markdown sạch cho tệp .md.
func NormalizeRichProseToMarkdown(raw string) string {
	blocks := ParseRichProseBlocks(raw)
	var out []string
	for _, blk := range blocks {
		switch blk.Kind {
		case RichBlockDivider:
			out = append(out, "* * *")
		case RichBlockHeading:
			out = append(out, "### "+renderSpansMarkdown(blk.Spans))
		case RichBlockQuote:
			out = append(out, "> "+renderSpansMarkdown(blk.Spans))
		default:
			out = append(out, renderSpansMarkdown(blk.Spans))
		}
	}
	return strings.Join(out, "\n\n")
}

func renderSpansMarkdown(spans []RichSpan) string {
	var b strings.Builder
	for _, sp := range spans {
		txt := sp.Text
		if sp.Underline {
			txt = "<u>" + txt + "</u>"
		}
		if sp.Bold && sp.Italic {
			txt = "***" + txt + "***"
		} else if sp.Bold {
			txt = "**" + txt + "**"
		} else if sp.Italic {
			txt = "*" + txt + "*"
		}
		b.WriteString(txt)
	}
	return b.String()
}

// RenderRichBlocksHTML chuyển đổi danh sách RichBlock sang mã HTML5 (.html) hoặc XHTML (.epub) hợp lệ.
func RenderRichBlocksHTML(blocks []RichBlock, isXHTML bool, indent string) string {
	var b strings.Builder
	for _, blk := range blocks {
		switch blk.Kind {
		case RichBlockDivider:
			if isXHTML {
				b.WriteString(indent + `<div class="scene-break">* * *</div>` + "\n")
			} else {
				b.WriteString(indent + `<hr class="scene-break">` + "\n")
			}
		case RichBlockHeading:
			if isXHTML {
				b.WriteString(indent + `<h4 class="sub-heading">` + renderSpansHTML(blk.Spans, true) + `</h4>` + "\n")
			} else {
				b.WriteString(indent + `<h5 class="sub-heading">` + renderSpansHTML(blk.Spans, false) + `</h5>` + "\n")
			}
		case RichBlockQuote:
			b.WriteString(indent + `<blockquote><p>` + renderSpansHTML(blk.Spans, isXHTML) + `</p></blockquote>` + "\n")
		default:
			b.WriteString(indent + `<p>` + renderSpansHTML(blk.Spans, isXHTML) + `</p>` + "\n")
		}
	}
	return b.String()
}

func renderSpansHTML(spans []RichSpan, isXHTML bool) string {
	var b strings.Builder
	for _, sp := range spans {
		var escaped string
		if isXHTML {
			escaped = escapeXML(sp.Text)
		} else {
			escaped = html.EscapeString(sp.Text)
		}
		if sp.Underline {
			if isXHTML {
				escaped = `<span class="underline-span">` + escaped + `</span>`
			} else {
				escaped = `<u>` + escaped + `</u>`
			}
		}
		if sp.Italic {
			escaped = `<em>` + escaped + `</em>`
		}
		if sp.Bold {
			escaped = `<strong>` + escaped + `</strong>`
		}
		b.WriteString(escaped)
	}
	return b.String()
}

// RenderRichBlocksODT chuyển đổi danh sách RichBlock sang các nút XML OpenDocument Text (<text:p>, <text:h>, <text:span>).
func RenderRichBlocksODT(blocks []RichBlock) string {
	var b strings.Builder
	for _, blk := range blocks {
		switch blk.Kind {
		case RichBlockDivider:
			b.WriteString(`      <text:p text:style-name="SceneSeparator">* * *</text:p>` + "\n")
		case RichBlockHeading:
			b.WriteString(`      <text:h text:style-name="Heading_20_3" text:outline-level="3">` + renderSpansODT(blk.Spans) + `</text:h>` + "\n")
		case RichBlockQuote:
			b.WriteString(`      <text:p text:style-name="Block_20_Quote">` + renderSpansODT(blk.Spans) + `</text:p>` + "\n")
		default:
			b.WriteString(`      <text:p text:style-name="Text_20_body">` + renderSpansODT(blk.Spans) + `</text:p>` + "\n")
		}
	}
	return b.String()
}

func renderSpansODT(spans []RichSpan) string {
	var b strings.Builder
	for _, sp := range spans {
		escaped := escapeXML(sp.Text)
		styleName := ""
		switch {
		case sp.Bold && sp.Italic && sp.Underline:
			styleName = "TBoldItalicUnderline"
		case sp.Bold && sp.Italic:
			styleName = "TBoldItalic"
		case sp.Bold && sp.Underline:
			styleName = "TBoldUnderline"
		case sp.Italic && sp.Underline:
			styleName = "TItalicUnderline"
		case sp.Bold:
			styleName = "TBold"
		case sp.Italic:
			styleName = "TItalic"
		case sp.Underline:
			styleName = "TUnderline"
		}
		if styleName != "" {
			b.WriteString(fmt.Sprintf(`<text:span text:style-name="%s">%s</text:span>`, styleName, escaped))
		} else {
			b.WriteString(escaped)
		}
	}
	return b.String()
}

// RenderRichBlocksPlainText xuất các khối RichBlock ra văn bản thuần (.txt), gỡ bỏ các ký hiệu đánh dấu thô.
func RenderRichBlocksPlainText(blocks []RichBlock) string {
	var b strings.Builder
	for _, blk := range blocks {
		plain := flattenSpansPlainText(blk.Spans)
		switch blk.Kind {
		case RichBlockDivider:
			b.WriteString("                    * * *\n\n")
		case RichBlockHeading:
			b.WriteString("    [" + strings.ToUpper(plain) + "]\n\n")
		case RichBlockQuote:
			b.WriteString("    │ “" + plain + "”\n\n")
		default:
			b.WriteString("    " + plain + "\n\n")
		}
	}
	return b.String()
}

func flattenSpansPlainText(spans []RichSpan) string {
	var b strings.Builder
	for _, sp := range spans {
		b.WriteString(sp.Text)
	}
	return b.String()
}

// wrapRichSpansPt ngắt dòng danh sách RichSpan theo chiều rộng điểm ảnh PDF (pt) mà vẫn giữ nguyên thuộc tính Bold/Italic/Underline của từng từ.
func wrapRichSpansPt(m *ttfFontMetrics, spans []RichSpan, fontSize float64, maxWidthPt float64) [][]RichSpan {
	return wrapRichSpansFirstLinePt(m, spans, fontSize, maxWidthPt, 0)
}

func wrapRichSpansFirstLinePt(m *ttfFontMetrics, spans []RichSpan, fontSize float64, maxWidthPt float64, firstLineIndent float64) [][]RichSpan {
	type styledWord struct {
		word      string
		bold      bool
		italic    bool
		underline bool
	}
	var words []styledWord
	for _, sp := range spans {
		parts := strings.Fields(sp.Text)
		for _, w := range parts {
			words = append(words, styledWord{
				word:      w,
				bold:      sp.Bold,
				italic:    sp.Italic,
				underline: sp.Underline,
			})
		}
	}
	if len(words) == 0 {
		return nil
	}

	var lines [][]RichSpan
	var curLine []styledWord
	var curWidth float64
	spaceW := m.measureStringPt(" ", fontSize)

	flushLine := func() {
		if len(curLine) == 0 {
			return
		}
		var merged []RichSpan
		for idx, sw := range curLine {
			piece := sw.word
			if idx < len(curLine)-1 {
				piece += " "
			}
			if len(merged) > 0 {
				last := &merged[len(merged)-1]
				if last.Bold == sw.bold && last.Italic == sw.italic && last.Underline == sw.underline {
					last.Text += piece
					continue
				}
			}
			merged = append(merged, RichSpan{
				Text:      piece,
				Bold:      sw.bold,
				Italic:    sw.italic,
				Underline: sw.underline,
			})
		}
		lines = append(lines, merged)
		curLine = nil
		curWidth = 0
	}

	for _, sw := range words {
		wPt := m.measureStringPt(sw.word, fontSize)
		limit := maxWidthPt
		if len(lines) == 0 {
			limit = maxWidthPt - firstLineIndent
		}
		addedW := wPt
		if len(curLine) > 0 {
			addedW += spaceW
		}
		if len(curLine) > 0 && curWidth+addedW > limit {
			flushLine()
		}
		curLine = append(curLine, sw)
		curWidth += addedW
	}
	flushLine()

	return lines
}
