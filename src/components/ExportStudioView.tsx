import React, { useMemo, useState } from 'react';
import {
  Check,
  Copy,
  Download,
  FileText,
  Filter,
  Headphones,
  Volume2,
  Play,
  Square,
  Loader2,
  Sparkles,
  Terminal,
} from 'lucide-react';
import { Act, Chapter, Project, Scene } from '../types/novelist';
import { compileProjectSQLiteDump } from '../data/initialNovelData';

interface ExportStudioViewProps {
  project: Project;
}

export interface VietnameseVoicePreset {
  id: string;
  label: string;
  gender: string;
  description: string;
}

export const VIETNAMESE_VOICE_PRESETS: VietnameseVoicePreset[] = [
  {
    id: 'vi-VN-HoaiMyNeural',
    label: 'Hoài Mỹ (Nữ - Miền Nam)',
    gender: 'Nữ',
    description:
      'Giọng nữ miền Nam dịu dàng, tự nhiên, giàu cảm xúc — rất thích hợp cho tiểu thuyết tình cảm & tự sự.',
  },
  {
    id: 'vi-VN-NamMinhNeural',
    label: 'Nam Minh (Nam - Miền Nam)',
    gender: 'Nam',
    description:
      'Giọng nam miền Nam ấm áp, truyền cảm, rõ chữ — rất thích hợp cho tiểu thuyết lịch sử & phiêu lưu.',
  },
];

function cleanProseForSpeech(raw: string): string {
  let text = raw;
  text = text.replace(/<[^>]*>/g, '');
  text = text.replace(/\*\*/g, '').replace(/\*/g, '').replace(/_/g, '');
  text = text.replace(/^\s*(\*\s*\*\s*\*|―+|--+)\s*$/gm, '\n\n');
  text = text.replace(/^\s*>\s*/gm, '');
  text = text.replace(/\n{3,}/g, '\n\n');
  return text.trim();
}

type ExportFormat =
  | 'rendered'
  | 'audio'
  | 'txt'
  | 'odt'
  | 'pdf'
  | 'epub'
  | 'markdown'
  | 'html'
  | 'sql';

type ExportScope = 'all_book' | 'selected_act' | 'current_node';

export const ExportStudioView: React.FC<ExportStudioViewProps> = ({
  project,
}) => {
  const [format, setFormat] = useState<ExportFormat>('rendered');
  const [scope, setScope] = useState<ExportScope>('all_book');
  const [selectedActId, setSelectedActId] = useState<number>(
    () => project.acts[0]?.id || 0
  );
  const [selectedNodeKey, setSelectedNodeKey] = useState<string>(() => {
    const firstCh = project.acts[0]?.chapters[0];
    return firstCh ? `chapter:${firstCh.id}` : '';
  });
  const [includeSynopsis, setIncludeSynopsis] = useState(true);
  const [includeSceneTitle, setIncludeSceneTitle] = useState(true);
  const [copied, setCopied] = useState(false);

  // Cấu hình âm thanh AI (Edge-TTS Tiếng Việt)
  const [selectedVoiceId, setSelectedVoiceId] = useState<string>(
    VIETNAMESE_VOICE_PRESETS[0].id
  );
  const [customAudioText, setCustomAudioText] = useState<string>('');
  const [isExportingAudio, setIsExportingAudio] = useState(false);
  const [audioSuccessMsg, setAudioSuccessMsg] = useState<string | null>(null);
  const [isPlayingWebSpeech, setIsPlayingWebSpeech] = useState(false);

  // Lọc cấu trúc Hồi -> Chương -> Cảnh theo Phạm vi xuất bản (mô phỏng BuildFilteredManuscript trong export.go)
  const filteredManuscript = useMemo(() => {
    let acts: Act[] = [];
    let scopeLabel = 'Toàn bộ tác phẩm';

    if (scope === 'selected_act') {
      const targetAct =
        project.acts.find((a) => a.id === selectedActId) || project.acts[0];
      if (targetAct) {
        acts = [targetAct];
        scopeLabel = `Theo Hồi chỉ định: ${targetAct.title}`;
      }
    } else if (scope === 'current_node') {
      const [kind, rawId] = selectedNodeKey.split(':');
      const id = Number(rawId);
      if (kind === 'chapter') {
        for (const a of project.acts) {
          const ch = a.chapters.find((c) => c.id === id);
          if (ch) {
            acts = [{ ...a, chapters: [ch] }];
            scopeLabel = `Chương hiện tại: ${ch.title} (${a.title})`;
            break;
          }
        }
      } else if (kind === 'scene') {
        for (const a of project.acts) {
          for (const ch of a.chapters) {
            const sc = ch.scenes.find((s) => s.id === id);
            if (sc) {
              acts = [
                {
                  ...a,
                  chapters: [{ ...ch, scenes: [sc] }],
                },
              ];
              scopeLabel = `Cảnh hiện tại: ${sc.title} (${ch.title})`;
              break;
            }
          }
        }
      }
    } else {
      acts = project.acts;
      scopeLabel = 'Toàn bộ tác phẩm';
    }

    const totalChapters = acts.reduce((acc, a) => acc + a.chapters.length, 0);
    const totalScenes = acts.reduce(
      (acc, a) =>
        acc + a.chapters.reduce((cAcc, c) => cAcc + c.scenes.length, 0),
      0
    );
    const totalWords = acts.reduce(
      (acc, a) =>
        acc +
        a.chapters.reduce(
          (cAcc, c) =>
            cAcc + c.scenes.reduce((sAcc, s) => sAcc + s.wordCount, 0),
          0
        ),
      0
    );

    return {
      acts,
      scopeLabel,
      totalChapters,
      totalScenes,
      totalWords,
    };
  }, [project, scope, selectedActId, selectedNodeKey]);

  // Bộ biên dịch Plain Text (.txt)
  const txtOutput = useMemo(() => {
    const lines: string[] = [];
    const titleUpper = project.title.toUpperCase();
    lines.push(titleUpper);
    lines.push('='.repeat(Math.max(20, titleUpper.length)));
    lines.push('');
    lines.push(`Tác giả: ${project.author || 'Khuyết danh'}`);
    lines.push(`Thể loại: ${project.genre}`);
    lines.push(`Phạm vi xuất bản: ${filteredManuscript.scopeLabel}`);
    lines.push(
      `Tổng số từ: ${filteredManuscript.totalWords.toLocaleString('vi-VN')} từ`
    );
    lines.push('');
    if (includeSynopsis && project.synopsis) {
      lines.push('TÓM TẮT TÁC PHẨM:');
      lines.push(project.synopsis.trim());
      lines.push('');
    }
    lines.push('='.repeat(60));
    lines.push('');

    for (const act of filteredManuscript.acts) {
      lines.push(act.title.toUpperCase());
      lines.push('-'.repeat(60));
      lines.push('');
      for (const ch of act.chapters) {
        lines.push(ch.title);
        lines.push('');
        ch.scenes.forEach((sc: Scene, idx: number) => {
          if (includeSceneTitle && sc.title) {
            lines.push(`[${sc.title}]`);
            lines.push('');
          }
          sc.content
            .split(/\n\s*\n/)
            .map((p) => p.trim())
            .filter(Boolean)
            .forEach((p) => {
              lines.push(`    ${p}`);
              lines.push('');
            });
          if (idx < ch.scenes.length - 1) {
            lines.push('                    * * *');
            lines.push('');
          }
        });
      }
    }
    return lines.join('\n');
  }, [project, filteredManuscript, includeSynopsis, includeSceneTitle]);

  // Bộ biên dịch Markdown (.md)
  const markdownOutput = useMemo(() => {
    const lines: string[] = [];
    lines.push(`# ${project.title}\n`);
    lines.push(`**Tác giả:** ${project.author || 'Khuyết danh'}  `);
    lines.push(`**Thể loại:** ${project.genre}  `);
    lines.push(`**Phạm vi xuất bản:** ${filteredManuscript.scopeLabel}  `);
    lines.push(
      `**Tổng số từ:** ${filteredManuscript.totalWords.toLocaleString('vi-VN')} từ\n`
    );
    if (includeSynopsis && project.synopsis) {
      lines.push(`> ${project.synopsis.trim()}\n`);
    }
    lines.push(`---\n`);
    for (const act of filteredManuscript.acts) {
      lines.push(`## ${act.title}\n`);
      for (const ch of act.chapters) {
        lines.push(`### ${ch.title}\n`);
        ch.scenes.forEach((sc, idx) => {
          if (includeSceneTitle && sc.title) {
            lines.push(`#### ${sc.title}\n`);
          }
          if (sc.content.trim()) {
            lines.push(`${sc.content.trim()}\n`);
          }
          if (idx < ch.scenes.length - 1) {
            lines.push(`* * *\n`);
          }
        });
      }
    }
    return lines.join('\n');
  }, [project, filteredManuscript, includeSynopsis, includeSceneTitle]);

  // Bộ biên dịch HTML (.html)
  const htmlOutput = useMemo(() => {
    const lines: string[] = [];
    lines.push(`<!DOCTYPE html>
<html lang="vi">
<head>
<meta charset="UTF-8">
<title>${project.title}</title>
<style>
  body { font-family: 'Noto Serif', 'DejaVu Serif', Georgia, serif; max-width: 740px; margin: 3rem auto; padding: 0 1.5rem; color: #1C1B18; background: #FAF7F2; line-height: 1.85; }
  h1 { font-size: 2.4rem; margin-bottom: 0.25rem; }
  .meta { color: #57534E; font-style: italic; margin-bottom: 1.5rem; }
  h2 { margin-top: 3rem; border-bottom: 1px solid #D6D0C4; padding-bottom: 0.4rem; }
  h3 { margin-top: 2rem; color: #3F3C36; }
  h4 { margin-top: 1.5rem; color: #78716C; text-transform: uppercase; letter-spacing: 0.08em; font-size: 0.85rem; }
  p { margin: 1.1rem 0; text-indent: 1.5rem; text-align: justify; }
</style>
</head>
<body>`);
    lines.push(`<h1>${project.title}</h1>`);
    lines.push(
      `<div class="meta">Tác giả: ${project.author} &bull; Thể loại: ${project.genre} &bull; ${filteredManuscript.scopeLabel} (${filteredManuscript.totalWords} từ)</div>`
    );
    if (includeSynopsis && project.synopsis) {
      lines.push(`<blockquote>${project.synopsis}</blockquote>`);
    }
    for (const act of filteredManuscript.acts) {
      lines.push(`<h2>${act.title}</h2>`);
      for (const ch of act.chapters) {
        lines.push(`<h3>${ch.title}</h3>`);
        ch.scenes.forEach((sc, idx) => {
          if (includeSceneTitle && sc.title) {
            lines.push(`<h4>${sc.title}</h4>`);
          }
          sc.content
            .split(/\n\s*\n/)
            .map((p) => p.trim())
            .filter(Boolean)
            .forEach((p) => {
              lines.push(`<p>${p}</p>`);
            });
          if (idx < ch.scenes.length - 1) {
            lines.push(`<hr/>`);
          }
        });
      }
    }
    lines.push(`</body>\n</html>`);
    return lines.join('\n');
  }, [project, filteredManuscript, includeSynopsis, includeSceneTitle]);

  // Bộ biên dịch OpenDocument Text XML (.odt content.xml preview)
  const odtOutput = useMemo(() => {
    const lines: string[] = [];
    lines.push(`<?xml version="1.0" encoding="UTF-8"?>
<!-- Tệp .odt thực tế trong GoNovelist (export.go -> ExportManuscriptODT) là kho lưu trữ ZIP chuẩn OASIS ODF 1.2 chứa:
     1. mimetype (application/vnd.oasis.opendocument.text - zip.Store không nén)
     2. META-INF/manifest.xml
     3. meta.xml
     4. styles.xml (Định nghĩa Heading 1/2/3 & Text body thụt đầu dòng 1cm)
     5. content.xml (Dưới đây là cấu trúc XML đã lọc theo Phạm vi xuất bản): -->
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0"
  xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0" office:version="1.2">
  <office:body>
    <office:text>
      <text:p text:style-name="DocTitle">${project.title}</text:p>
      <text:p text:style-name="DocMeta">Tác giả: ${project.author} | Phạm vi: ${filteredManuscript.scopeLabel}</text:p>`);
    for (const act of filteredManuscript.acts) {
      lines.push(
        `      <text:h text:style-name="Heading_20_1" text:outline-level="1">${act.title}</text:h>`
      );
      for (const ch of act.chapters) {
        lines.push(
          `      <text:h text:style-name="Heading_20_2" text:outline-level="2">${ch.title}</text:h>`
        );
        ch.scenes.forEach((sc, idx) => {
          if (includeSceneTitle && sc.title) {
            lines.push(
              `      <text:h text:style-name="Heading_20_3" text:outline-level="3">${sc.title}</text:h>`
            );
          }
          sc.content
            .split(/\n\s*\n/)
            .map((p) => p.trim())
            .filter(Boolean)
            .forEach((p) => {
              lines.push(
                `      <text:p text:style-name="Text_20_body">${p}</text:p>`
              );
            });
          if (idx < ch.scenes.length - 1) {
            lines.push(
              `      <text:p text:style-name="SceneSeparator">* * *</text:p>`
            );
          }
        });
      }
    }
    lines.push(`    </office:text>\n  </office:body>\n</office:document-content>`);
    return lines.join('\n');
  }, [project, filteredManuscript, includeSceneTitle]);

  // Bộ biên dịch EPUB (.epub OEBPS/content.opf + XHTML preview)
  const epubOutput = useMemo(() => {
    const lines: string[] = [];
    lines.push(`<!-- Tệp .epub thực tế trong GoNovelist (export.go -> ExportManuscriptEPUB) là kho lưu trữ ZIP chuẩn IDPF EPUB 3 chứa:
     1. mimetype (application/epub+zip - zip.Store không nén)
     2. META-INF/container.xml
     3. OEBPS/style.css
     4. OEBPS/title.xhtml
     5. OEBPS/nav.xhtml & OEBPS/toc.ncx (Mục lục phân cấp Hồi -> Chương)
     6. OEBPS/chapter_XXX.xhtml (Từng chương tách riêng chuẩn XHTML 1.1 Tiếng Việt UTF-8)
     7. OEBPS/content.opf -->`);
    lines.push(`<?xml version="1.0" encoding="UTF-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="BookId" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>${project.title} (${filteredManuscript.scopeLabel})</dc:title>
    <dc:creator>${project.author}</dc:creator>
    <dc:language>vi</dc:language>
  </metadata>
</package>\n`);
    lines.push(txtOutput);
    return lines.join('\n');
  }, [project, filteredManuscript, txtOutput]);

  const sqlOutput = compileProjectSQLiteDump(project);

  // Thu thập nội dung văn bản sạch để Edge-TTS đọc (CleanProseForSpeech)
  const defaultSpeechText = useMemo(() => {
    const paragraphs: string[] = [];
    paragraphs.push(project.title);
    if (includeSynopsis && project.synopsis) {
      paragraphs.push(project.synopsis);
    }
    for (const act of filteredManuscript.acts) {
      for (const ch of act.chapters) {
        for (const sc of ch.scenes) {
          if (sc.content.trim()) {
            paragraphs.push(sc.content);
          }
        }
      }
    }
    return cleanProseForSpeech(paragraphs.join('\n\n'));
  }, [project, filteredManuscript, includeSynopsis]);

  const activeAudioContent = customAudioText || defaultSpeechText;

  const currentVoice = useMemo(
    () =>
      VIETNAMESE_VOICE_PRESETS.find((v) => v.id === selectedVoiceId) ||
      VIETNAMESE_VOICE_PRESETS[0],
    [selectedVoiceId]
  );

  const audioWordCount = useMemo(() => {
    const trimmed = activeAudioContent.trim();
    return trimmed ? trimmed.split(/\s+/).length : 0;
  }, [activeAudioContent]);

  const audioCharCount = activeAudioContent.length;
  const estimatedAudioMinutes = (audioWordCount / 160).toFixed(1);

  const safeFileName = useMemo(() => {
    return (
      project.title
        .toLowerCase()
        .replace(/[^a-z0-9\u00C0-\u1EF9]+/gi, '_')
        .replace(/^_|_$/g, '') || 'audio_novel'
    );
  }, [project.title]);

  const edgeTTSCommand = useMemo(() => {
    return `edge-tts --voice ${selectedVoiceId} --text "${activeAudioContent.slice(0, 100).replace(/"/g, '\\"')}..." --write-media "${safeFileName}.mp3"`;
  }, [selectedVoiceId, activeAudioContent, safeFileName]);

  const handleExportAudio = () => {
    setIsExportingAudio(true);
    setAudioSuccessMsg(null);

    // Mô phỏng quá trình tổng hợp Neural TTS từ Edge-TTS
    setTimeout(() => {
      setIsExportingAudio(false);
      setAudioSuccessMsg('Xuất audio thành công!');

      // Tạo tệp download script / MP3 mô phỏng
      const blob = new Blob(
        [
          `# GoNovelist Edge-TTS Shell Script\n# Voice: ${selectedVoiceId} (${currentVoice.label})\n# Content Words: ${audioWordCount}\n\nedge-tts --voice "${selectedVoiceId}" --text "${activeAudioContent.replace(/"/g, '\\"')}" --write-media "${safeFileName}.mp3"\n`,
        ],
        { type: 'text/plain;charset=utf-8' }
      );
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${safeFileName}_tts_script.sh`;
      a.click();
      URL.revokeObjectURL(url);
    }, 1200);
  };

  const handleToggleWebSpeech = () => {
    if (!('speechSynthesis' in window)) {
      alert('Trình duyệt không hỗ trợ Web Speech API.');
      return;
    }

    if (isPlayingWebSpeech) {
      window.speechSynthesis.cancel();
      setIsPlayingWebSpeech(false);
      return;
    }

    window.speechSynthesis.cancel();
    const utterance = new SpeechSynthesisUtterance(activeAudioContent.slice(0, 800));
    utterance.lang = 'vi-VN';
    utterance.rate = 0.95;
    utterance.onend = () => setIsPlayingWebSpeech(false);
    utterance.onerror = () => setIsPlayingWebSpeech(false);

    const voices = window.speechSynthesis.getVoices();
    const viVoice = voices.find((v) => v.lang.startsWith('vi'));
    if (viVoice) {
      utterance.voice = viVoice;
    }

    window.speechSynthesis.speak(utterance);
    setIsPlayingWebSpeech(true);
  };

  const activeText =
    format === 'audio'
      ? activeAudioContent
      : format === 'txt' || format === 'pdf'
      ? txtOutput
      : format === 'odt'
      ? odtOutput
      : format === 'epub'
      ? epubOutput
      : format === 'html'
      ? htmlOutput
      : format === 'sql'
      ? sqlOutput
      : markdownOutput;

  const handleCopy = async () => {
    await navigator.clipboard.writeText(activeText);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = () => {
    if (format === 'audio') {
      handleExportAudio();
      return;
    }

    const extMap: Record<ExportFormat, string> = {
      rendered: '.txt',
      audio: '.mp3',
      txt: '.txt',
      odt: '.odt.xml',
      pdf: '.txt',
      epub: '.epub.xml',
      markdown: '.md',
      html: '.html',
      sql: '.sql',
    };
    const ext = extMap[format] || '.txt';
    const mime =
      format === 'html'
        ? 'text/html;charset=utf-8'
        : 'text/plain;charset=utf-8';
    const blob = new Blob([activeText], { type: mime });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `${safeFileName}${ext}`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="flex-1 overflow-y-auto bg-[#FAF7F2] px-8 py-8">
      <div className="max-w-5xl mx-auto space-y-6">
        {/* Tiêu đề & Thống kê theo Phạm vi xuất bản */}
        <div className="border-b border-[#D6D0C4] pb-6 flex flex-col md:flex-row md:items-end justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-[0.14em] text-[#8B3A2B] font-semibold">
              Hệ Thống Xuất Bản Đa Định Dạng & Chọn Phạm Vi (gonovelist/export.go)
            </p>
            <h1 className="font-serif-display text-3xl md:text-4xl font-bold text-[#1C1B18] mt-1">
              Xuất Bản Thảo — {project.title}
            </h1>
            <p className="text-xs text-[#57534E] mt-1">
              Phạm vi đang chọn:{' '}
              <span className="font-semibold text-[#1C1B18]">
                {filteredManuscript.scopeLabel}
              </span>
            </p>
          </div>
          <div className="flex items-center gap-4 text-xs font-mono-code text-[#57534E]">
            <span>{filteredManuscript.acts.length} Hồi</span>
            <span>&bull;</span>
            <span>{filteredManuscript.totalChapters} Chương</span>
            <span>&bull;</span>
            <span>{filteredManuscript.totalScenes} Cảnh</span>
            <span>&bull;</span>
            <span className="text-[#1C1B18] font-semibold">
              {filteredManuscript.totalWords.toLocaleString('vi-VN')} từ
            </span>
          </div>
        </div>

        {/* Hộp cấu hình Phạm vi xuất bản (Granular Export Scope) */}
        <div className="bg-[#F3EFE6] border border-[#D6D0C4] p-5 space-y-4">
          <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-[#1C1B18]">
            <Filter className="w-3.5 h-3.5 text-[#8B3A2B]" />
            <span>1. Phạm vi xuất bản (Granular Export Scope)</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
            <label
              className={`flex items-start gap-2.5 p-3 border cursor-pointer transition-colors ${
                scope === 'all_book'
                  ? 'bg-[#FAF7F2] border-[#8B3A2B]'
                  : 'bg-[#FAF7F2]/60 border-[#D6D0C4]'
              }`}
            >
              <input
                type="radio"
                name="exportScope"
                checked={scope === 'all_book'}
                onChange={() => setScope('all_book')}
                className="mt-0.5 accent-[#8B3A2B]"
              />
              <div>
                <div className="text-xs font-semibold text-[#1C1B18]">
                  Toàn bộ tác phẩm
                </div>
                <div className="text-[11px] text-[#57534E] mt-0.5">
                  Xuất tuần tự tất cả Hồi, Chương và Cảnh trong sách.
                </div>
              </div>
            </label>

            <label
              className={`flex items-start gap-2.5 p-3 border cursor-pointer transition-colors ${
                scope === 'selected_act'
                  ? 'bg-[#FAF7F2] border-[#8B3A2B]'
                  : 'bg-[#FAF7F2]/60 border-[#D6D0C4]'
              }`}
            >
              <input
                type="radio"
                name="exportScope"
                checked={scope === 'selected_act'}
                onChange={() => setScope('selected_act')}
                className="mt-0.5 accent-[#8B3A2B]"
              />
              <div className="flex-1">
                <div className="text-xs font-semibold text-[#1C1B18]">
                  Theo Hồi chỉ định
                </div>
                <select
                  value={selectedActId}
                  onChange={(e) => {
                    setSelectedActId(Number(e.target.value));
                    setScope('selected_act');
                  }}
                  className="mt-1.5 w-full text-xs bg-white border border-[#D6D0C4] px-2 py-1 text-[#1C1B18]"
                >
                  {project.acts.map((act) => (
                    <option key={act.id} value={act.id}>
                      {act.title}
                    </option>
                  ))}
                </select>
              </div>
            </label>

            <label
              className={`flex items-start gap-2.5 p-3 border cursor-pointer transition-colors ${
                scope === 'current_node'
                  ? 'bg-[#FAF7F2] border-[#8B3A2B]'
                  : 'bg-[#FAF7F2]/60 border-[#D6D0C4]'
              }`}
            >
              <input
                type="radio"
                name="exportScope"
                checked={scope === 'current_node'}
                onChange={() => setScope('current_node')}
                className="mt-0.5 accent-[#8B3A2B]"
              />
              <div className="flex-1">
                <div className="text-xs font-semibold text-[#1C1B18]">
                  Chương/Cảnh hiện tại
                </div>
                <select
                  value={selectedNodeKey}
                  onChange={(e) => {
                    setSelectedNodeKey(e.target.value);
                    setScope('current_node');
                  }}
                  className="mt-1.5 w-full text-xs bg-white border border-[#D6D0C4] px-2 py-1 text-[#1C1B18]"
                >
                  {project.acts.map((act) =>
                    act.chapters.map((ch: Chapter) => (
                      <React.Fragment key={ch.id}>
                        <option value={`chapter:${ch.id}`}>
                          📖 {ch.title} (Cả chương)
                        </option>
                        {ch.scenes.map((sc: Scene) => (
                          <option key={sc.id} value={`scene:${sc.id}`}>
                            &nbsp;&nbsp;🎬 {sc.title} (Riêng cảnh)
                          </option>
                        ))}
                      </React.Fragment>
                    ))
                  )}
                </select>
              </div>
            </label>
          </div>

          <div className="flex flex-wrap items-center gap-6 pt-1 text-xs text-[#3F3C36]">
            <label className="inline-flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={includeSynopsis}
                onChange={(e) => setIncludeSynopsis(e.target.checked)}
                className="accent-[#8B3A2B]"
              />
              <span>Đính kèm phần Tóm tắt tác phẩm</span>
            </label>
            <label className="inline-flex items-center gap-2 cursor-pointer">
              <input
                type="checkbox"
                checked={includeSceneTitle}
                onChange={(e) => setIncludeSceneTitle(e.target.checked)}
                className="accent-[#8B3A2B]"
              />
              <span>Hiển thị tiêu đề từng Cảnh trong Chương</span>
            </label>
          </div>
        </div>

        {/* Thanh chọn định dạng & nút tải về */}
        <div className="flex flex-wrap items-center justify-between gap-4 bg-[#F3EFE6] border border-[#D6D0C4] p-3">
          <div className="flex flex-wrap items-center gap-1">
            {(
              [
                ['rendered', 'Xem Trước Bản In (PDF/EPUB)'],
                ['audio', '🎧 Audio MP3 (Edge-TTS)'],
                ['txt', 'Xuất bản ra Text (.txt)'],
                ['odt', 'Xuất bản ra ODT (.odt)'],
                ['pdf', 'Xuất bản ra PDF (.pdf)'],
                ['epub', 'Xuất bản ra EPUB (.epub)'],
                ['markdown', 'Markdown (.md)'],
                ['html', 'HTML (.html)'],
                ['sql', 'SQLite (.sql)'],
              ] as const
            ).map(([key, label]) => (
              <button
                key={key}
                type="button"
                onClick={() => setFormat(key)}
                className={`px-3 py-1.5 text-xs font-medium transition-colors cursor-pointer ${
                  format === key
                    ? 'bg-[#1C1B18] text-[#FAF7F2]'
                    : 'text-[#57534E] hover:text-[#1C1B18]'
                }`}
              >
                {label}
              </button>
            ))}
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={handleCopy}
              className="inline-flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-medium border border-[#D6D0C4] bg-[#FAF7F2] text-[#1C1B18] hover:border-[#1C1B18] transition-colors cursor-pointer"
            >
              {copied ? (
                <>
                  <Check className="w-3.5 h-3.5 text-[#2D6A4F]" />
                  <span>Đã sao chép</span>
                </>
              ) : (
                <>
                  <Copy className="w-3.5 h-3.5" />
                  <span>Sao chép nội dung</span>
                </>
              )}
            </button>
            <button
              type="button"
              onClick={handleDownload}
              className="inline-flex items-center gap-1.5 px-4 py-1.5 text-xs font-medium bg-[#8B3A2B] text-[#FAF7F2] hover:bg-[#722E21] transition-colors cursor-pointer"
            >
              <Download className="w-3.5 h-3.5" />
              <span>Tải tệp xuống</span>
            </button>
          </div>
        </div>

        {/* Khung hiển thị bản thảo hoặc Studio Audio Edge-TTS */}
        {format === 'audio' ? (
          <div className="bg-[#FAF7F2] border border-[#D6D0C4] p-8 max-w-4xl mx-auto shadow-xs space-y-6">
            {/* Header Audio Studio */}
            <div className="border-b border-[#D6D0C4] pb-5">
              <div className="flex items-center gap-2 text-xs font-mono-code uppercase tracking-wider text-[#8B3A2B] font-semibold">
                <Headphones className="w-4 h-4" />
                <span>XUẤT BẢN FILE AUDIO (MP3) — GIỌNG ĐỌC AI TIẾNG VIỆT</span>
              </div>
              <h2 className="font-serif-display text-2xl font-bold text-[#1C1B18] mt-1">
                Edge-TTS Neural Voice Studio
              </h2>
              <p className="text-xs text-[#57534E] mt-1">
                Sử dụng công nghệ Neural Text-to-Speech miễn phí từ Edge-TTS với giọng đọc tự nhiên, chuẩn sắc thái Tiếng Việt.
              </p>
            </div>

            {/* Bước 1: Chọn giọng đọc */}
            <div className="bg-[#F3EFE6] border border-[#D6D0C4] p-4 space-y-3">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                <label htmlFor="voice-select" className="text-xs font-semibold text-[#1C1B18] flex items-center gap-1.5">
                  <Volume2 className="w-4 h-4 text-[#8B3A2B]" />
                  <span>Chọn giọng đọc Neural Tiếng Việt:</span>
                </label>
                <span className="text-[11px] font-mono-code text-[#78716C]">
                  Voice ID: {selectedVoiceId}
                </span>
              </div>

              <select
                id="voice-select"
                value={selectedVoiceId}
                onChange={(e) => {
                  setSelectedVoiceId(e.target.value);
                  setAudioSuccessMsg(null);
                }}
                className="w-full text-xs font-medium bg-white border border-[#D6D0C4] px-3 py-2 text-[#1C1B18] focus:outline-hidden focus:border-[#8B3A2B]"
              >
                {VIETNAMESE_VOICE_PRESETS.map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.label} — {v.gender}
                  </option>
                ))}
              </select>

              <div className="p-3 bg-[#FAF7F2] border border-[#E6E0D4] text-xs text-[#57534E] italic flex items-start gap-2">
                <Sparkles className="w-3.5 h-3.5 text-[#8B3A2B] shrink-0 mt-0.5" />
                <span>{currentVoice.description}</span>
              </div>
            </div>

            {/* Bước 2: Nội dung văn bản */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <label className="text-xs font-semibold text-[#1C1B18]">
                  Nội dung văn bản để đọc thành tiếng:
                </label>
                <button
                  type="button"
                  onClick={() => {
                    setCustomAudioText('');
                    setAudioSuccessMsg(null);
                  }}
                  className="text-xs text-[#8B3A2B] hover:underline cursor-pointer"
                >
                  Nạp lại từ bản thảo đang chọn
                </button>
              </div>

              <textarea
                value={activeAudioContent}
                onChange={(e) => {
                  setCustomAudioText(e.target.value);
                  setAudioSuccessMsg(null);
                }}
                rows={8}
                placeholder="Nhập hoặc dán nội dung cần chuyển sang âm thanh..."
                className="w-full font-serif-display text-sm text-[#1C1B18] bg-white border border-[#D6D0C4] p-3 focus:outline-hidden focus:border-[#8B3A2B] resize-y"
              />

              <div className="flex flex-wrap items-center justify-between text-xs text-[#57534E] pt-1">
                <div className="flex items-center gap-3">
                  <span>📊 Độ dài: <strong>{audioWordCount.toLocaleString('vi-VN')}</strong> từ</span>
                  <span>&bull;</span>
                  <span>{audioCharCount.toLocaleString('vi-VN')} ký tự</span>
                  <span>&bull;</span>
                  <span>(khoảng <strong>{estimatedAudioMinutes}</strong> phút nghe)</span>
                </div>
                <button
                  type="button"
                  onClick={handleToggleWebSpeech}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1 text-xs border border-[#D6D0C4] bg-white text-[#1C1B18] hover:bg-[#F3EFE6] transition-colors cursor-pointer"
                >
                  {isPlayingWebSpeech ? (
                    <>
                      <Square className="w-3 h-3 text-[#B91C1C]" />
                      <span>Dừng đọc thử</span>
                    </>
                  ) : (
                    <>
                      <Play className="w-3 h-3 text-[#2D6A4F]" />
                      <span>Đọc thử (Web Speech)</span>
                    </>
                  )}
                </button>
              </div>
            </div>

            {/* Bước 3: Đường dẫn lưu tệp đích */}
            <div className="bg-[#F3EFE6] border border-[#D6D0C4] p-4 space-y-2">
              <label htmlFor="filename-preview" className="text-xs font-semibold text-[#1C1B18] block">
                Đường dẫn lưu tệp MP3 đích (Fyne dialog.FileSave):
              </label>
              <div className="flex items-center gap-2">
                <input
                  id="filename-preview"
                  type="text"
                  readOnly
                  value={`${safeFileName}.mp3`}
                  className="flex-1 text-xs font-mono-code bg-white border border-[#D6D0C4] px-3 py-2 text-[#1C1B18]"
                />
                <span className="text-xs text-[#78716C] font-mono-code">.mp3</span>
              </div>
            </div>

            {/* Thanh tiến độ / Thông báo trạng thái */}
            {isExportingAudio && (
              <div className="p-4 bg-[#FEF3C7] border border-[#F59E0B] text-xs text-[#92400E] flex items-center gap-2.5 animate-pulse">
                <Loader2 className="w-4 h-4 animate-spin text-[#D97706]" />
                <span className="font-medium">
                  ⏳ Đang kết nối Edge-TTS và tổng hợp âm thanh MP3...
                </span>
              </div>
            )}

            {audioSuccessMsg && (
              <div className="p-4 bg-[#ECFDF5] border border-[#10B981] text-xs text-[#065F46] flex items-center justify-between gap-2">
                <div className="flex items-center gap-2 font-medium">
                  <Check className="w-4 h-4 text-[#059669]" />
                  <span>✅ {audioSuccessMsg} Đã chuẩn bị kịch bản & file âm thanh MP3.</span>
                </div>
                <span className="font-mono-code text-[11px] text-[#047857]">
                  {currentVoice.label} • {audioWordCount} từ
                </span>
              </div>
            )}

            {/* Hộp lệnh Edge-TTS CLI Pattern */}
            <div className="bg-[#181715] text-[#E7E2D8] border border-[#2E2C28] p-4 space-y-2">
              <div className="flex items-center justify-between text-xs text-[#A8A29E]">
                <div className="flex items-center gap-1.5 font-mono-code">
                  <Terminal className="w-3.5 h-3.5 text-[#E7E2D8]" />
                  <span>Mô thức lệnh Edge-TTS CLI (Backend Go `os/exec`):</span>
                </div>
                <button
                  type="button"
                  onClick={async () => {
                    await navigator.clipboard.writeText(
                      `edge-tts --voice ${selectedVoiceId} --text "${activeAudioContent.replace(/"/g, '\\"')}" --write-media "${safeFileName}.mp3"`
                    );
                    setCopied(true);
                    setTimeout(() => setCopied(false), 2000);
                  }}
                  className="text-[11px] text-[#E7E2D8] hover:text-white underline cursor-pointer"
                >
                  Sao chép lệnh Terminal đầy đủ
                </button>
              </div>
              <code className="block font-mono-code text-xs text-[#A7F3D0] break-all bg-black/40 p-2.5 rounded-xs">
                {edgeTTSCommand}
              </code>
            </div>

            {/* Nút hành động chính */}
            <div className="flex items-center justify-end gap-3 pt-2">
              <button
                type="button"
                disabled={isExportingAudio}
                onClick={handleExportAudio}
                className="inline-flex items-center gap-2 px-6 py-2.5 text-xs font-semibold bg-[#8B3A2B] text-[#FAF7F2] hover:bg-[#722E21] disabled:opacity-60 transition-colors shadow-xs cursor-pointer"
              >
                {isExportingAudio ? (
                  <>
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <span>Đang tổng hợp Audio...</span>
                  </>
                ) : (
                  <>
                    <Volume2 className="w-4 h-4" />
                    <span>Xuất file Audio</span>
                  </>
                )}
              </button>
            </div>
          </div>
        ) : format === 'rendered' || format === 'pdf' ? (
          <div className="bg-[#FAF7F2] border border-[#D6D0C4] px-10 py-12 max-w-3xl mx-auto shadow-xs space-y-8">
            <div className="border-b border-[#D6D0C4] pb-6 text-center space-y-2">
              <div className="inline-flex items-center gap-1.5 text-[11px] font-mono-code uppercase tracking-widest text-[#8B3A2B]">
                <FileText className="w-3.5 h-3.5" />
                <span>{filteredManuscript.scopeLabel}</span>
              </div>
              <h2 className="font-serif-display text-4xl font-bold text-[#1C1B18]">
                {project.title}
              </h2>
              <p className="text-sm italic text-[#57534E]">
                Tác giả: {project.author || 'Khuyết danh'} &bull; {project.genre}
              </p>
              {includeSynopsis && project.synopsis && (
                <p className="text-xs text-[#78716C] max-w-xl mx-auto pt-2">
                  {project.synopsis}
                </p>
              )}
            </div>

            {filteredManuscript.acts.map((act) => (
              <div key={act.id} className="space-y-8">
                <h3 className="font-serif-display text-2xl font-bold text-[#1C1B18] border-b border-[#E6E0D4] pb-2 uppercase tracking-wide">
                  {act.title}
                </h3>
                {act.chapters.map((ch) => (
                  <div key={ch.id} className="space-y-6">
                    <h4 className="font-serif-display text-xl font-semibold text-[#3F3C36]">
                      {ch.title}
                    </h4>
                    {ch.scenes.map((sc, idx) => (
                      <div key={sc.id} className="space-y-4">
                        {includeSceneTitle && (
                          <div className="text-xs uppercase tracking-[0.12em] text-[#78716C] font-mono-code">
                            {sc.title}
                          </div>
                        )}
                        <div className="font-serif-display text-lg leading-[1.85] text-[#1C1B18] space-y-4">
                          {sc.content
                            .split(/\n\s*\n/)
                            .filter(Boolean)
                            .map((para, pIdx) => (
                              <p key={pIdx} className="indent-6">
                                {para}
                              </p>
                            ))}
                        </div>
                        {idx < ch.scenes.length - 1 && (
                          <div className="text-center text-[#78716C] tracking-[0.4em] py-3">
                            * * *
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            ))}
          </div>
        ) : (
          <div className="bg-[#181715] text-[#E7E2D8] border border-[#2E2C28] p-6 overflow-x-auto">
            <pre className="font-mono-code text-xs leading-relaxed whitespace-pre-wrap">
              {activeText}
            </pre>
          </div>
        )}
      </div>
    </div>
  );
};
