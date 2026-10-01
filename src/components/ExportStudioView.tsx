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

type ExportFormat = 'markdown' | 'html-source' | 'html-preview' | 'sqlite-sql';

export const ExportStudioView: React.FC<ExportStudioViewProps> = ({ project }) => {
  const [format, setFormat] = useState<ExportFormat>('markdown');
  const [copied, setCopied] = useState(false);

  const mdOutput = compileProjectMarkdown(project);
  const htmlOutput = compileProjectHTML(project);
  const sqlOutput = compileProjectSQLiteDump(project);

  const activeText =
    format === 'markdown'
      ? mdOutput
      : format === 'sqlite-sql'
      ? sqlOutput
      : htmlOutput;

  const totalWords = project.acts.reduce(
    (sum, a) =>
      sum +
      a.chapters.reduce(
        (cSum, c) => cSum + c.scenes.reduce((sSum, s) => sSum + s.wordCount, 0),
        0
      ),
    0
  );

  const totalChapters = project.acts.reduce((s, a) => s + a.chapters.length, 0);
  const totalScenes = project.acts.reduce(
    (s, a) => s + a.chapters.reduce((cs, c) => cs + c.scenes.length, 0),
    0
  );

  const handleCopy = async () => {
    await navigator.clipboard.writeText(activeText);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  };

  const handleDownload = (ext: 'md' | 'html' | 'sql') => {
    const content =
      ext === 'md' ? mdOutput : ext === 'html' ? htmlOutput : sqlOutput;
    const mime =
      ext === 'md'
        ? 'text/markdown;charset=utf-8'
        : ext === 'html'
        ? 'text/html;charset=utf-8'
        : 'application/sql;charset=utf-8';
    const blob = new Blob([content], { type: mime });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    const slug = project.title.toLowerCase().replace(/[^a-z0-9]+/g, '_');
    a.href = url;
    a.download = `${slug}_manuscript.${ext}`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="max-w-6xl mx-auto px-6 py-8 space-y-6">
      <div className="border-b border-[#E6E4DD] pb-6 flex flex-col lg:flex-row lg:items-end justify-between gap-4">
        <div>
          <p className="text-xs text-[#68655E] mb-1">
            Sequential Manuscript Compiler · ExportSubEngine
          </p>
          <h1 className="text-3xl font-semibold text-[#1C1B18] text-balance">
            Compile Acts, Chapters &amp; Scenes
          </h1>
          <div className="flex items-center gap-2 text-xs text-[#68655E] mt-1 font-mono tabular-nums">
            <span>{project.acts.length} Acts</span>
            <span aria-hidden="true">·</span>
            <span>{totalChapters} Chapters</span>
            <span aria-hidden="true">·</span>
            <span>{totalScenes} Scenes</span>
            <span aria-hidden="true">·</span>
            <span>{totalWords.toLocaleString()} Words Compiled</span>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <button
            onClick={() => handleDownload('md')}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold text-white bg-[#1E3A5F] rounded-lg hover:bg-[#162B47] transition-colors whitespace-nowrap"
          >
            <Download className="w-3.5 h-3.5" />
            Download .MD
          </button>
          <button
            onClick={() => handleDownload('html')}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold text-[#1C1B18] bg-white border border-[#DCD9D0] rounded-lg hover:bg-[#F1EFEA] transition-colors whitespace-nowrap"
          >
            <Download className="w-3.5 h-3.5" />
            Download .HTML
          </button>
          <button
            onClick={() => handleDownload('sql')}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold text-[#1C1B18] bg-white border border-[#DCD9D0] rounded-lg hover:bg-[#F1EFEA] transition-colors whitespace-nowrap"
          >
            <Download className="w-3.5 h-3.5" />
            Export SQLite .SQL
          </button>
        </div>
      </div>

      {/* Format Segmented Control + Copy Button */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-1 p-1 bg-[#EAE7DF] rounded-lg">
          <button
            onClick={() => setFormat('markdown')}
            className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap ${
              format === 'markdown'
                ? 'bg-white text-[#1C1B18] shadow-xs'
                : 'text-[#68655E] hover:text-[#1C1B18]'
            }`}
          >
            Markdown Output (.md)
          </button>
          <button
            onClick={() => setFormat('html-preview')}
            className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap ${
              format === 'html-preview'
                ? 'bg-white text-[#1C1B18] shadow-xs'
                : 'text-[#68655E] hover:text-[#1C1B18]'
            }`}
          >
            Typeset HTML Proof
          </button>
          <button
            onClick={() => setFormat('html-source')}
            className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap ${
              format === 'html-source'
                ? 'bg-white text-[#1C1B18] shadow-xs'
                : 'text-[#68655E] hover:text-[#1C1B18]'
            }`}
          >
            HTML5 Source (.html)
          </button>
          <button
            onClick={() => setFormat('sqlite-sql')}
            className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap ${
              format === 'sqlite-sql'
                ? 'bg-white text-[#1C1B18] shadow-xs'
                : 'text-[#68655E] hover:text-[#1C1B18]'
            }`}
          >
            SQLite INSERT Dump (.sql)
          </button>
        </div>

        <button
          onClick={handleCopy}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-[#1C1B18] bg-white border border-[#DCD9D0] rounded-lg hover:bg-[#F1EFEA] transition-colors whitespace-nowrap"
        >
          {copied ? (
            <>
              <Check className="w-3.5 h-3.5 text-emerald-700" />
              Copied to Clipboard
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              Copy Compiled Output
            </>
          )}
        </button>
      </div>

      {/* Output Viewer */}
      {format === 'html-preview' ? (
        <div className="border border-[#E6E4DD] bg-[#FAF9F5] rounded-xl p-8 md:p-14 max-w-4xl mx-auto">
          <div className="text-center border-b border-[#DCD9D0] pb-8 mb-10">
            <h2 className="font-serif-display text-4xl font-semibold text-[#1C1B18]">
              {project.title}
            </h2>
            <p className="text-xs text-[#68655E] mt-2">
              By {project.author} · {project.genre}
            </p>
          </div>

          <div className="space-y-10">
            {project.acts.map((act) => (
              <div key={act.id} className="space-y-6">
                <h3 className="font-serif-display text-2xl font-semibold uppercase tracking-wider text-[#1C1B18] border-b border-[#E6E4DD] pb-2">
                  {act.title}
                </h3>
                {act.chapters.map((ch) => (
                  <div key={ch.id} className="space-y-5">
                    <h4 className="font-serif-display text-xl font-semibold text-[#1C1B18]">
                      {ch.title}
                    </h4>
                    {ch.scenes.map((sc, idx) => (
                      <div key={sc.id} className="space-y-3">
                        <div className="text-xs text-[#68655E]">
                          <span className="font-semibold text-[#1C1B18]">{sc.title}</span>
                          <span className="mx-2">·</span>
                          <span>{sc.status}</span>
                        </div>
                        <div className="font-serif-display text-lg leading-relaxed text-[#1C1B18] space-y-4">
                          {sc.content
                            .split(/\n\s*\n/)
                            .filter(Boolean)
                            .map((para, pIdx) => (
                              <p
                                key={pIdx}
                                className={pIdx > 0 ? 'indent-6' : ''}
                              >
                                {para}
                              </p>
                            ))}
                        </div>
                        {idx < ch.scenes.length - 1 && (
                          <div className="text-center text-[#68655E] tracking-[0.4em] py-3">
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
        </div>
      ) : (
        <div className="border border-[#E6E4DD] bg-[#161615] text-[#EAE7DF] rounded-xl overflow-hidden">
          <div className="px-4 py-2.5 border-b border-[#2A2927] flex items-center justify-between text-xs text-[#A09D95] font-mono tabular-nums">
            <span>
              {format === 'markdown'
                ? 'ExportManuscriptMarkdown() — Sequential Markdown Output'
                : format === 'sqlite-sql'
                ? 'SQLite Transactional Dump — Compatible with gonovelist.db'
                : 'ExportManuscriptHTML() — Standalone HTML5 Document'}
            </span>
            <span>{activeText.split('\n').length} lines</span>
          </div>
          <pre className="p-5 text-xs font-mono leading-relaxed overflow-x-auto max-h-[620px] overflow-y-auto">
            <code>{activeText}</code>
          </pre>
        </div>
      )}
    </div>
  );
};
