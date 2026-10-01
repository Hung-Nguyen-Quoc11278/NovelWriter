import React, { useState } from 'react';
import { Check, Copy, Download } from 'lucide-react';
import {
  BUILD_COMMANDS,
  GO_SOURCE_FILES,
  PROJECT_TREE_LAYOUT,
} from '../data/goSourceFiles';

interface GoSourceExplorerProps {
  mode: 'source' | 'architecture';
}

export const GoSourceExplorer: React.FC<GoSourceExplorerProps> = ({ mode }) => {
  const [selectedFileIndex, setSelectedFileIndex] = useState(0);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  const activeFile = GO_SOURCE_FILES[selectedFileIndex] || GO_SOURCE_FILES[0];
  const schemaFile =
    GO_SOURCE_FILES.find((f) => f.filename === 'schema.sql') || GO_SOURCE_FILES[0];

  const copyText = async (key: string, text: string) => {
    await navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 1800);
  };

  const downloadSingleFile = (filename: string, content: string) => {
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  const downloadBundleScript = () => {
    const scriptLines: string[] = [
      '#!/usr/bin/env bash',
      'set -e',
      'mkdir -p gonovelist',
      'cd gonovelist',
      '',
    ];
    for (const file of GO_SOURCE_FILES) {
      scriptLines.push(`cat << 'EOF_GONOVELIST_FILE' > ${file.filename}`);
      scriptLines.push(file.code);
      scriptLines.push('EOF_GONOVELIST_FILE');
      scriptLines.push('');
    }
    scriptLines.push('echo "GoNovelist source files extracted into ./gonovelist"');
    scriptLines.push('echo "Run: cd gonovelist && go mod tidy && go run ."');

    downloadSingleFile('setup_gonovelist.sh', scriptLines.join('\n'));
  };

  if (mode === 'architecture') {
    return (
      <div className="max-w-6xl mx-auto px-6 py-8 space-y-10">
        <div className="border-b border-[#E6E4DD] pb-6 flex flex-col md:flex-row md:items-end justify-between gap-4">
          <div>
            <p className="text-xs text-[#68655E] mb-1">
              Clean Architecture · Go 1.22+ · Fyne v2 · Local SQLite
            </p>
            <h1 className="text-3xl font-semibold text-[#1C1B18] text-balance">
              Project Structure, SQLite Schema &amp; Build Guide
            </h1>
          </div>
          <button
            onClick={downloadBundleScript}
            className="inline-flex items-center gap-2 px-4 py-2 text-xs font-semibold text-white bg-[#1E3A5F] rounded-lg hover:bg-[#162B47] transition-colors whitespace-nowrap"
          >
            <Download className="w-3.5 h-3.5" />
            Download Full Project Setup Script (.sh)
          </button>
        </div>

        {/* 1. Directory Structure + Clean Architecture Layers */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
          <div className="lg:col-span-5 border border-[#E6E4DD] bg-[#161615] text-[#EAE7DF] rounded-xl overflow-hidden">
            <div className="px-4 py-2.5 border-b border-[#2A2927] flex items-center justify-between text-xs text-[#A09D95] font-mono">
              <span>1. Project Directory Structure</span>
              <span>7 files</span>
            </div>
            <pre className="p-5 text-xs font-mono leading-relaxed overflow-x-auto">
              <code>{PROJECT_TREE_LAYOUT}</code>
            </pre>
          </div>

          <div className="lg:col-span-7 border border-[#E6E4DD] bg-white rounded-xl p-6 space-y-4">
            <h2 className="text-xl font-semibold text-[#1C1B18]">
              Modular Monolith &amp; Clean Architecture Separation
            </h2>
            <div className="divide-y divide-[#F1EFEA] text-sm">
              <div className="py-3 flex flex-col sm:flex-row sm:items-baseline justify-between gap-2">
                <span className="font-mono text-xs font-semibold text-[#1E3A5F] shrink-0">
                  models.go (Domain Layer)
                </span>
                <span className="text-xs text-[#3A3832]">
                  Pure Go domain entities (`Project`, `Act`, `Chapter`, `Scene`, `Character`, `Location`), `TreeNode` UID codec (`act:1`, `scene:14`), and allocation-free Unicode word counting.
                </span>
              </div>
              <div className="py-3 flex flex-col sm:flex-row sm:items-baseline justify-between gap-2">
                <span className="font-mono text-xs font-semibold text-[#1E3A5F] shrink-0">
                  database.go (Repository &amp; Services)
                </span>
                <span className="text-xs text-[#3A3832]">
                  SQLite WAL connection (`modernc.org/sqlite` or `mattn/go-sqlite3`), schema migration, hierarchical loading, transactional `UpdateScene` with M:N `scene_characters` sync, and Markdown/HTML compilers.
                </span>
              </div>
              <div className="py-3 flex flex-col sm:flex-row sm:items-baseline justify-between gap-2">
                <span className="font-mono text-xs font-semibold text-[#1E3A5F] shrink-0">
                  ui_main.go (Fyne Shell &amp; Tree)
                </span>
                <span className="text-xs text-[#3A3832]">
                  Main Fyne window layout (`container.NewHSplit`), reactive `widget.Tree` sidebar, node CRUD &amp; sibling reordering (`MoveUp`/`MoveDown`), Cast/Locations dialog, and Distraction-Free toggle.
                </span>
              </div>
              <div className="py-3 flex flex-col sm:flex-row sm:items-baseline justify-between gap-2">
                <span className="font-mono text-xs font-semibold text-[#1E3A5F] shrink-0">
                  ui_editor.go (Editor &amp; Auto-Save)
                </span>
                <span className="text-xs text-[#3A3832]">
                  Multi-line prose editor + `widget.RichText` Markdown preview, `750ms` debounced auto-save engine (`time.AfterFunc` + `fyne.Do`), Scene/Chapter `widget.ProgressBar`, metadata selectors, and Side Notes tab.
                </span>
              </div>
            </div>
          </div>
        </div>

        {/* 2. SQLite Schema DDL */}
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-xl font-semibold text-[#1C1B18]">
                2. Local SQLite Database Schema (schema.sql)
              </h2>
              <p className="text-xs text-[#68655E]">
                Enforces cascading deletes across Project → Act → Chapter → Scene and M:N scene_characters mapping.
              </p>
            </div>
            <button
              onClick={() => copyText('ddl', schemaFile.code)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-[#1C1B18] bg-white border border-[#DCD9D0] rounded-lg hover:bg-[#F1EFEA] transition-colors whitespace-nowrap"
            >
              {copiedKey === 'ddl' ? (
                <>
                  <Check className="w-3.5 h-3.5 text-emerald-700" />
                  Copied DDL
                </>
              ) : (
                <>
                  <Copy className="w-3.5 h-3.5" />
                  Copy SQLite DDL
                </>
              )}
            </button>
          </div>
          <div className="border border-[#E6E4DD] bg-[#161615] text-[#EAE7DF] rounded-xl overflow-hidden">
            <pre className="p-5 text-xs font-mono leading-relaxed overflow-x-auto">
              <code>{schemaFile.code}</code>
            </pre>
          </div>
        </div>

        {/* 4. Terminal Build & Run Instructions */}
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-xl font-semibold text-[#1C1B18]">
                4. Build &amp; Execution Instructions (Go 1.22+ &amp; Fyne v2)
              </h2>
              <p className="text-xs text-[#68655E]">
                Exact terminal commands to initialize the module, fetch dependencies, and build native desktop binaries.
              </p>
            </div>
            <button
              onClick={() => copyText('build', BUILD_COMMANDS)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-[#1C1B18] bg-white border border-[#DCD9D0] rounded-lg hover:bg-[#F1EFEA] transition-colors whitespace-nowrap"
            >
              {copiedKey === 'build' ? (
                <>
                  <Check className="w-3.5 h-3.5 text-emerald-700" />
                  Copied Commands
                </>
              ) : (
                <>
                  <Copy className="w-3.5 h-3.5" />
                  Copy Build Commands
                </>
              )}
            </button>
          </div>
          <div className="border border-[#E6E4DD] bg-[#161615] text-[#EAE7DF] rounded-xl overflow-hidden">
            <pre className="p-5 text-xs font-mono leading-relaxed overflow-x-auto">
              <code>{BUILD_COMMANDS}</code>
            </pre>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto px-6 py-8 space-y-6">
      <div className="border-b border-[#E6E4DD] pb-5 flex flex-col md:flex-row md:items-end justify-between gap-4">
        <div>
          <p className="text-xs text-[#68655E] mb-1">
            Production Go (Golang) + Fyne v2 + SQLite Source Repository
          </p>
          <h1 className="text-3xl font-semibold text-[#1C1B18] text-balance">
            GoNovelist Complete Source Code
          </h1>
        </div>
        <div className="flex items-center gap-2.5">
          <button
            onClick={() =>
              downloadSingleFile(activeFile.filename, activeFile.code)
            }
            className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-medium text-[#1C1B18] bg-white border border-[#DCD9D0] rounded-lg hover:bg-[#F1EFEA] transition-colors whitespace-nowrap"
          >
            <Download className="w-3.5 h-3.5" />
            Download {activeFile.filename}
          </button>
          <button
            onClick={downloadBundleScript}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold text-white bg-[#1E3A5F] rounded-lg hover:bg-[#162B47] transition-colors whitespace-nowrap"
          >
            <Download className="w-3.5 h-3.5" />
            Download All Files (.sh Bundle)
          </button>
        </div>
      </div>

      {/* File Selector Tabs */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-wrap items-center gap-1 p-1 bg-[#EAE7DF] rounded-lg">
          {GO_SOURCE_FILES.map((file, idx) => (
            <button
              key={file.filename}
              onClick={() => setSelectedFileIndex(idx)}
              className={`px-3 py-1.5 text-xs font-mono font-medium rounded-md transition-colors whitespace-nowrap ${
                selectedFileIndex === idx
                  ? 'bg-white text-[#1C1B18] shadow-xs'
                  : 'text-[#68655E] hover:text-[#1C1B18]'
              }`}
            >
              {file.filename}
            </button>
          ))}
        </div>

        <button
          onClick={() => copyText(activeFile.filename, activeFile.code)}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-[#1C1B18] bg-white border border-[#DCD9D0] rounded-lg hover:bg-[#F1EFEA] transition-colors whitespace-nowrap"
        >
          {copiedKey === activeFile.filename ? (
            <>
              <Check className="w-3.5 h-3.5 text-emerald-700" />
              Copied {activeFile.filename}
            </>
          ) : (
            <>
              <Copy className="w-3.5 h-3.5" />
              Copy {activeFile.filename}
            </>
          )}
        </button>
      </div>

      {/* File Info Header + Code Viewer */}
      <div className="border border-[#E6E4DD] bg-[#161615] text-[#EAE7DF] rounded-xl overflow-hidden">
        <div className="px-5 py-3.5 border-b border-[#2A2927] flex flex-col sm:flex-row sm:items-center justify-between gap-2">
          <div>
            <div className="flex items-center gap-2 text-xs font-mono">
              <span className="font-semibold text-white">{activeFile.path}</span>
              <span className="text-[#68655E]">·</span>
              <span className="text-[#A09D95]">{activeFile.layer}</span>
            </div>
            <p className="text-xs text-[#A09D95] mt-1">{activeFile.summary}</p>
          </div>
          <div className="text-xs font-mono text-[#A09D95] tabular-nums shrink-0">
            {activeFile.code.split('\n').length} lines
          </div>
        </div>

        <div className="overflow-x-auto max-h-[680px] overflow-y-auto p-5">
          <table className="w-full border-collapse font-mono text-xs leading-relaxed">
            <tbody>
              {activeFile.code.split('\n').map((line, i) => (
                <tr key={i} className="hover:bg-white/5">
                  <td className="select-none pr-4 text-right text-[#68655E] tabular-nums w-10 align-top">
                    {i + 1}
                  </td>
                  <td className="whitespace-pre text-[#EAE7DF]">{line || ' '}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
