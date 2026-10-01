import React, { useState } from 'react';
import { Check, Copy, Download } from 'lucide-react';
import { Project } from '../types/novelist';
import {
  compileProjectHTML,
  compileProjectMarkdown,
  compileProjectSQLiteDump,
} from '../data/initialNovelData';

interface ExportStudioViewProps {
  project: Project;
}

type ExportFormat = 'markdown' | 'html' | 'rendered' | 'sql';

export const ExportStudioView: React.FC<ExportStudioViewProps> = ({ project }) => {
  const [format, setFormat] = useState<ExportFormat>('markdown');
  const [copied, setCopied] = useState(false);

  const markdownOutput = compileProjectMarkdown(project);
  const htmlOutput = compileProjectHTML(project);
  const sqlOutput = compileProjectSQLiteDump(project);

  const totalChapters = project.acts.reduce((acc, a) => acc + a.chapters.length, 0);
  const totalScenes = project.acts.reduce(
    (acc, a) => acc + a.chapters.reduce((cAcc, c) => cAcc + c.scenes.length, 0),
    0
  );
  const totalWords = project.acts.reduce(
    (acc, a) =>
      acc +
      a.chapters.reduce(
        (cAcc, c) => cAcc + c.scenes.reduce((sAcc, s) => sAcc + s.wordCount, 0),
        0
      ),
    0
  );

  const activeText =
    format === 'html'
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
    const ext = format === 'html' ? '.html' : format === 'sql' ? '.sql' : '.md';
    const mime =
      format === 'html'
        ? 'text/html;charset=utf-8'
        : 'text/plain;charset=utf-8';
    const blob = new Blob([activeText], { type: mime });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    const safeName = project.title
      .toLowerCase()
      .replace(/[^a-z0-9\u00C0-\u1EF9]+/gi, '_')
      .replace(/^_|_$/g, '');
    a.href = url;
    a.download = `${safeName || 'ban_thao'}${ext}`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="flex-1 overflow-y-auto bg-[#FAF7F2] px-8 py-8">
      <div className="max-w-5xl mx-auto space-y-8">
        {/* Tiêu đề & Thống kê xuất bản thảo */}
        <div className="border-b border-[#D6D0C4] pb-6 flex flex-col md:flex-row md:items-end justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-[0.14em] text-[#78716C] font-semibold">
              Bộ Biên Dịch Bản Thảo Tuần Tự (ExportManuscriptMarkdown / ExportManuscriptHTML)
            </p>
            <h1 className="font-serif-display text-3xl md:text-4xl font-bold text-[#1C1B18] mt-1">
              Xuất Bản Thảo — {project.title}
            </h1>
          </div>
          <div className="flex items-center gap-4 text-xs font-mono-code text-[#57534E]">
            <span>{project.acts.length} Hồi</span>
            <span>&bull;</span>
            <span>{totalChapters} Chương</span>
            <span>&bull;</span>
            <span>{totalScenes} Cảnh</span>
            <span>&bull;</span>
            <span className="text-[#1C1B18] font-semibold">
              {totalWords.toLocaleString('vi-VN')} từ
            </span>
          </div>
        </div>

        {/* Thanh chọn định dạng & nút tải về */}
        <div className="flex flex-wrap items-center justify-between gap-4 bg-[#F3EFE6] border border-[#D6D0C4] p-3">
          <div className="flex flex-wrap items-center gap-1">
            <button
              type="button"
              onClick={() => setFormat('markdown')}
              className={`px-3.5 py-1.5 text-xs font-medium transition-colors cursor-pointer ${
                format === 'markdown'
                  ? 'bg-[#1C1B18] text-[#FAF7F2]'
                  : 'text-[#57534E] hover:text-[#1C1B18]'
              }`}
            >
              Markdown (.md)
            </button>
            <button
              type="button"
              onClick={() => setFormat('rendered')}
              className={`px-3.5 py-1.5 text-xs font-medium transition-colors cursor-pointer ${
                format === 'rendered'
                  ? 'bg-[#1C1B18] text-[#FAF7F2]'
                  : 'text-[#57534E] hover:text-[#1C1B18]'
              }`}
            >
              Xem Trước Bản In
            </button>
            <button
              type="button"
              onClick={() => setFormat('html')}
              className={`px-3.5 py-1.5 text-xs font-medium transition-colors cursor-pointer ${
                format === 'html'
                  ? 'bg-[#1C1B18] text-[#FAF7F2]'
                  : 'text-[#57534E] hover:text-[#1C1B18]'
              }`}
            >
              Mã Nguồn HTML (.html)
            </button>
            <button
              type="button"
              onClick={() => setFormat('sql')}
              className={`px-3.5 py-1.5 text-xs font-medium transition-colors cursor-pointer ${
                format === 'sql'
                  ? 'bg-[#1C1B18] text-[#FAF7F2]'
                  : 'text-[#57534E] hover:text-[#1C1B18]'
              }`}
            >
              Bản Sao Lưu SQLite (.sql)
            </button>
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

        {/* Khung hiển thị bản thảo */}
        {format === 'rendered' ? (
          <div className="bg-[#FAF7F2] border border-[#D6D0C4] px-10 py-12 max-w-3xl mx-auto shadow-xs space-y-8">
            <div className="border-b border-[#D6D0C4] pb-6 text-center space-y-2">
              <h2 className="font-serif-display text-4xl font-bold text-[#1C1B18]">
                {project.title}
              </h2>
              <p className="text-sm italic text-[#57534E]">
                Tác giả: {project.author || 'Khuyết danh'} &bull; {project.genre}
              </p>
              {project.synopsis && (
                <p className="text-xs text-[#78716C] max-w-xl mx-auto pt-2">
                  {project.synopsis}
                </p>
              )}
            </div>

            {project.acts.map((act) => (
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
                        <div className="text-xs uppercase tracking-[0.12em] text-[#78716C] font-mono-code">
                          {sc.title}
                        </div>
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
