import React, { useState } from 'react';
import { Check, Copy, Download } from 'lucide-react';
import {
  BUILD_COMMANDS,
  GO_SOURCE_FILES,
  PROJECT_TREE_LAYOUT,
} from '../data/goSourceFiles';

interface GoSourceExplorerProps {
  mode: 'source' | 'build';
}

export const GoSourceExplorer: React.FC<GoSourceExplorerProps> = ({ mode }) => {
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [copiedFile, setCopiedFile] = useState<string | null>(null);

  const activeFile = GO_SOURCE_FILES[selectedIndex] || GO_SOURCE_FILES[0];

  const copyText = async (text: string, key: string) => {
    await navigator.clipboard.writeText(text);
    setCopiedFile(key);
    setTimeout(() => setCopiedFile(null), 2000);
  };

  const downloadFile = (filename: string, content: string) => {
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (mode === 'build') {
    const schemaFile = GO_SOURCE_FILES.find((f) => f.filename === 'schema.sql');
    return (
      <div className="flex-1 overflow-y-auto bg-[#FAF7F2] px-8 py-8">
        <div className="max-w-5xl mx-auto space-y-10">
          <div className="border-b border-[#D6D0C4] pb-6">
            <p className="text-xs uppercase tracking-[0.14em] text-[#78716C] font-semibold">
              Kiến Trúc Hệ Thống & Hướng Dẫn Biên Dịch Thực Tế
            </p>
            <h1 className="font-serif-display text-3xl md:text-4xl font-bold text-[#1C1B18] mt-1">
              Cấu Trúc Thư Mục, Lược Đồ SQLite & Lệnh Biên Dịch Go 1.22+
            </h1>
          </div>

          {/* 1. Cấu trúc thư mục */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h2 className="font-serif-display text-2xl font-bold text-[#1C1B18]">
                1. Cấu Trúc Thư Mục Dự Án (Clean Architecture / Modular Monolith)
              </h2>
              <span className="text-xs font-mono-code text-[#78716C]">
                Tách biệt Tầng Giao diện Fyne, Nghiệp vụ và Lưu trữ SQLite
              </span>
            </div>
            <div className="bg-[#181715] text-[#E7E2D8] p-6 border border-[#2E2C28] overflow-x-auto">
              <pre className="font-mono-code text-xs leading-relaxed">
                {PROJECT_TREE_LAYOUT}
              </pre>
            </div>
          </div>

          {/* 2. Hướng dẫn chạy & đóng gói */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <h2 className="font-serif-display text-2xl font-bold text-[#1C1B18]">
                2. Các Lệnh Terminal Khởi Tạo, Chạy và Đóng Gói Ứng Dụng
              </h2>
              <button
                type="button"
                onClick={() => copyText(BUILD_COMMANDS, 'build-cmd')}
                className="inline-flex items-center gap-1.5 px-3 py-1 text-xs font-medium border border-[#D6D0C4] bg-[#F3EFE6] text-[#1C1B18] hover:border-[#1C1B18] transition-colors cursor-pointer"
              >
                {copiedFile === 'build-cmd' ? (
                  <>
                    <Check className="w-3.5 h-3.5 text-[#2D6A4F]" />
                    <span>Đã sao chép lệnh</span>
                  </>
                ) : (
                  <>
                    <Copy className="w-3.5 h-3.5" />
                    <span>Sao chép lệnh Terminal</span>
                  </>
                )}
              </button>
            </div>
            <div className="bg-[#181715] text-[#E7E2D8] p-6 border border-[#2E2C28] overflow-x-auto">
              <pre className="font-mono-code text-xs leading-relaxed">
                {BUILD_COMMANDS}
              </pre>
            </div>
          </div>

          {/* 3. Lược đồ SQLite DDL */}
          {schemaFile && (
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <h2 className="font-serif-display text-2xl font-bold text-[#1C1B18]">
                  3. Lược Đồ Cơ Sở Dữ Liệu SQLite (schema.sql)
                </h2>
                <button
                  type="button"
                  onClick={() => copyText(schemaFile.code, 'schema-sql')}
                  className="inline-flex items-center gap-1.5 px-3 py-1 text-xs font-medium border border-[#D6D0C4] bg-[#F3EFE6] text-[#1C1B18] hover:border-[#1C1B18] transition-colors cursor-pointer"
                >
                  {copiedFile === 'schema-sql' ? (
                    <>
                      <Check className="w-3.5 h-3.5 text-[#2D6A4F]" />
                      <span>Đã sao chép DDL</span>
                    </>
                  ) : (
                    <>
                      <Copy className="w-3.5 h-3.5" />
                      <span>Sao chép SQL DDL</span>
                    </>
                  )}
                </button>
              </div>
              <div className="bg-[#181715] text-[#E7E2D8] p-6 border border-[#2E2C28] overflow-x-auto">
                <pre className="font-mono-code text-xs leading-relaxed">
                  {schemaFile.code}
                </pre>
              </div>
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 flex overflow-hidden bg-[#FAF7F2]">
      {/* Thanh danh sách tệp mã nguồn Go */}
      <aside className="w-80 border-r border-[#D6D0C4] bg-[#F3EFE6] flex flex-col shrink-0">
        <div className="p-4 border-b border-[#D6D0C4]">
          <div className="text-[11px] uppercase tracking-[0.12em] text-[#78716C] font-semibold">
            Mã Nguồn Go + Fyne v2 (Tiếng Việt)
          </div>
          <div className="font-serif-display text-lg font-bold text-[#1C1B18] mt-0.5">
            /gonovelist/*.go
          </div>
        </div>

        <div className="flex-1 overflow-y-auto divide-y divide-[#E6E0D4]">
          {GO_SOURCE_FILES.map((file, idx) => {
            const isSelected = idx === selectedIndex;
            const lineCount = file.code.split('\n').length;
            return (
              <button
                key={file.filename}
                type="button"
                onClick={() => setSelectedIndex(idx)}
                className={`w-full text-left p-4 transition-colors cursor-pointer ${
                  isSelected
                    ? 'bg-[#FAF7F2] border-l-2 border-l-[#8B3A2B]'
                    : 'hover:bg-[#EAE4D7]'
                }`}
              >
                <div className="flex items-center justify-between">
                  <span className="font-mono-code text-sm font-semibold text-[#1C1B18]">
                    {file.filename}
                  </span>
                  <span className="font-mono-code text-[11px] text-[#78716C]">
                    {lineCount} dòng
                  </span>
                </div>
                <div className="text-[11px] uppercase tracking-wider text-[#8B3A2B] font-medium mt-1">
                  {file.layer}
                </div>
                <p className="text-xs text-[#57534E] mt-1 line-clamp-2">
                  {file.summary}
                </p>
              </button>
            );
          })}
        </div>
      </aside>

      {/* Khung xem mã nguồn */}
      <div className="flex-1 flex flex-col overflow-hidden">
        <div className="px-6 py-4 bg-[#FAF7F2] border-b border-[#D6D0C4] flex flex-wrap items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-3">
              <h2 className="font-mono-code text-base font-bold text-[#1C1B18]">
                {activeFile.path}
              </h2>
              <span className="text-xs font-mono-code text-[#8B3A2B]">
                {activeFile.layer}
              </span>
            </div>
            <p className="text-xs text-[#57534E] mt-0.5">{activeFile.summary}</p>
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => copyText(activeFile.code, activeFile.filename)}
              className="inline-flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-medium border border-[#D6D0C4] bg-[#F3EFE6] text-[#1C1B18] hover:border-[#1C1B18] transition-colors cursor-pointer"
            >
              {copiedFile === activeFile.filename ? (
                <>
                  <Check className="w-3.5 h-3.5 text-[#2D6A4F]" />
                  <span>Đã sao chép {activeFile.filename}</span>
                </>
              ) : (
                <>
                  <Copy className="w-3.5 h-3.5" />
                  <span>Sao chép mã nguồn</span>
                </>
              )}
            </button>
            <button
              type="button"
              onClick={() => downloadFile(activeFile.filename, activeFile.code)}
              className="inline-flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-medium bg-[#1C1B18] text-[#FAF7F2] hover:bg-[#3A3832] transition-colors cursor-pointer"
            >
              <Download className="w-3.5 h-3.5" />
              <span>Tải về {activeFile.filename}</span>
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-auto bg-[#181715] text-[#E7E2D8] p-6">
          <pre className="font-mono-code text-xs leading-relaxed">
            {activeFile.code}
          </pre>
        </div>
      </div>
    </div>
  );
};
