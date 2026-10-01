import React, { useState } from 'react';
import { Plus, Trash2, ArrowUpRight } from 'lucide-react';
import { Character, Location, Project } from '../types/novelist';

interface WorldCastViewProps {
  project: Project;
  onAddCharacter: (name: string, role: string, description: string) => void;
  onUpdateCharacter: (character: Character) => void;
  onDeleteCharacter: (id: number) => void;
  onAddLocation: (name: string, description: string) => void;
  onUpdateLocation: (location: Location) => void;
  onDeleteLocation: (id: number) => void;
  onSelectScene: (sceneId: number) => void;
}

export const WorldCastView: React.FC<WorldCastViewProps> = ({
  project,
  onAddCharacter,
  onUpdateCharacter,
  onDeleteCharacter,
  onAddLocation,
  onUpdateLocation,
  onDeleteLocation,
  onSelectScene,
}) => {
  const [newCharName, setNewCharName] = useState('');
  const [newCharRole, setNewCharRole] = useState('Nhân vật chính');
  const [newCharDesc, setNewCharDesc] = useState('');

  const [newLocName, setNewLocName] = useState('');
  const [newLocDesc, setNewLocDesc] = useState('');

  const allScenes = project.acts.flatMap((a) =>
    a.chapters.flatMap((c) =>
      c.scenes.map((sc) => ({
        ...sc,
        chapterTitle: c.title,
        actTitle: a.title,
      }))
    )
  );

  const handleCreateCharacter = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newCharName.trim()) return;
    onAddCharacter(newCharName.trim(), newCharRole.trim() || 'Nhân vật phụ', newCharDesc.trim());
    setNewCharName('');
    setNewCharDesc('');
  };

  const handleCreateLocation = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newLocName.trim()) return;
    onAddLocation(newLocName.trim(), newLocDesc.trim());
    setNewLocName('');
    setNewLocDesc('');
  };

  return (
    <div className="flex-1 overflow-y-auto bg-[#FAF7F2] px-8 py-8">
      <div className="max-w-6xl mx-auto space-y-12">
        {/* Tiêu đề trang */}
        <div className="border-b border-[#D6D0C4] pb-6 flex flex-col md:flex-row md:items-end justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-[0.14em] text-[#78716C] font-semibold">
              Hồ Sơ Thế Giới & Nhân Vật (SQLite: characters, locations, scene_characters)
            </p>
            <h1 className="font-serif-display text-3xl md:text-4xl font-bold text-[#1C1B18] mt-1">
              Danh Bạ Nhân Vật & Bối Cảnh — {project.title}
            </h1>
          </div>
          <div className="flex items-center gap-6 text-xs font-mono-code text-[#57534E]">
            <span>{project.characters.length} Nhân vật</span>
            <span>&bull;</span>
            <span>{project.locations.length} Bối cảnh</span>
            <span>&bull;</span>
            <span>{allScenes.length} Cảnh có liên kết</span>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-10">
          {/* Cột Nhân vật */}
          <div className="lg:col-span-7 space-y-6">
            <div className="flex items-center justify-between">
              <h2 className="font-serif-display text-2xl font-bold text-[#1C1B18]">
                Danh Sách Nhân Vật (Dramatis Personae)
              </h2>
              <span className="text-xs font-mono-code text-[#78716C]">
                Gắn với Góc nhìn (POV) & Nhân vật xuất hiện trong Cảnh
              </span>
            </div>

            {/* Biểu mẫu thêm nhân vật */}
            <form
              onSubmit={handleCreateCharacter}
              className="bg-[#F3EFE6] border border-[#D6D0C4] p-5 space-y-4"
            >
              <div className="text-xs uppercase tracking-[0.08em] font-semibold text-[#57534E]">
                Thêm Nhân Vật Mới
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <input
                  type="text"
                  value={newCharName}
                  onChange={(e) => setNewCharName(e.target.value)}
                  placeholder="Họ và tên nhân vật..."
                  className="sm:col-span-2 bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
                />
                <input
                  type="text"
                  value={newCharRole}
                  onChange={(e) => setNewCharRole(e.target.value)}
                  placeholder="Vai trò (VD: Nhân vật chính)"
                  className="bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
                />
              </div>
              <div className="flex gap-3">
                <input
                  type="text"
                  value={newCharDesc}
                  onChange={(e) => setNewCharDesc(e.target.value)}
                  placeholder="Tiểu sử ngắn, ngoại hình, động cơ hoặc bí mật..."
                  className="flex-1 bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
                />
                <button
                  type="submit"
                  className="inline-flex items-center gap-1.5 bg-[#8B3A2B] text-[#FAF7F2] px-4 py-2 text-xs font-medium tracking-wide hover:bg-[#722E21] transition-colors cursor-pointer"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Thêm Nhân Vật
                </button>
              </div>
            </form>

            {/* Danh sách thẻ nhân vật */}
            <div className="space-y-4">
              {project.characters.map((char) => {
                const povScenes = allScenes.filter((s) => s.povCharacterId === char.id);
                const presentScenes = allScenes.filter((s) =>
                  s.characterIds.includes(char.id)
                );

                return (
                  <div
                    key={char.id}
                    className="bg-[#FAF7F2] border border-[#D6D0C4] p-5 space-y-3 transition-colors hover:border-[#78716C]"
                  >
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex-1 grid grid-cols-1 sm:grid-cols-3 gap-2">
                        <input
                          type="text"
                          value={char.name}
                          onChange={(e) =>
                            onUpdateCharacter({ ...char, name: e.target.value })
                          }
                          className="sm:col-span-2 font-serif-display text-xl font-bold text-[#1C1B18] bg-transparent border-b border-transparent focus:border-[#8B3A2B] focus:outline-none"
                        />
                        <input
                          type="text"
                          value={char.role}
                          onChange={(e) =>
                            onUpdateCharacter({ ...char, role: e.target.value })
                          }
                          className="text-xs font-mono-code uppercase tracking-wider text-[#8B3A2B] bg-transparent border-b border-transparent focus:border-[#8B3A2B] focus:outline-none sm:text-right"
                        />
                      </div>
                      <button
                        type="button"
                        onClick={() => onDeleteCharacter(char.id)}
                        title="Xóa nhân vật"
                        className="text-[#78716C] hover:text-[#991B1B] p-1 transition-colors cursor-pointer"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>

                    <textarea
                      rows={2}
                      value={char.description}
                      onChange={(e) =>
                        onUpdateCharacter({ ...char, description: e.target.value })
                      }
                      placeholder="Ghi chú về nhân vật..."
                      className="w-full text-sm text-[#57534E] bg-transparent border border-transparent focus:border-[#D6D0C4] focus:bg-[#F3EFE6]/50 p-1.5 focus:outline-none resize-none"
                    />

                    <div className="pt-2 border-t border-[#E6E0D4] flex flex-wrap items-center justify-between gap-2 text-xs text-[#78716C]">
                      <div className="font-mono-code">
                        Góc nhìn (POV) trong <strong>{povScenes.length}</strong> cảnh &bull; Xuất hiện trong{' '}
                        <strong>{presentScenes.length}</strong> cảnh
                      </div>
                      <div className="flex flex-wrap gap-1.5">
                        {presentScenes.slice(0, 3).map((sc) => (
                          <button
                            key={sc.id}
                            type="button"
                            onClick={() => onSelectScene(sc.id)}
                            className="inline-flex items-center gap-1 text-[11px] bg-[#F3EFE6] hover:bg-[#E8E2D5] text-[#1C1B18] px-2 py-0.5 border border-[#D6D0C4] transition-colors cursor-pointer"
                          >
                            <span>{sc.title}</span>
                            <ArrowUpRight className="w-3 h-3 text-[#78716C]" />
                          </button>
                        ))}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Cột Bối cảnh / Địa điểm */}
          <div className="lg:col-span-5 space-y-6">
            <div className="flex items-center justify-between">
              <h2 className="font-serif-display text-2xl font-bold text-[#1C1B18]">
                Bối Cảnh & Địa Điểm
              </h2>
              <span className="text-xs font-mono-code text-[#78716C]">
                Không gian diễn ra các Cảnh
              </span>
            </div>

            {/* Biểu mẫu thêm địa điểm */}
            <form
              onSubmit={handleCreateLocation}
              className="bg-[#F3EFE6] border border-[#D6D0C4] p-5 space-y-3"
            >
              <div className="text-xs uppercase tracking-[0.08em] font-semibold text-[#57534E]">
                Thêm Địa Điểm / Bối Cảnh Mới
              </div>
              <input
                type="text"
                value={newLocName}
                onChange={(e) => setNewLocName(e.target.value)}
                placeholder="Tên địa điểm (VD: Xưởng Thủy Tinh Phố Cổ)..."
                className="w-full bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
              />
              <div className="flex gap-2">
                <input
                  type="text"
                  value={newLocDesc}
                  onChange={(e) => setNewLocDesc(e.target.value)}
                  placeholder="Chi tiết không gian, ánh sáng, âm thanh..."
                  className="flex-1 bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
                />
                <button
                  type="submit"
                  className="inline-flex items-center gap-1.5 bg-[#1C1B18] text-[#FAF7F2] px-4 py-2 text-xs font-medium tracking-wide hover:bg-[#3A3832] transition-colors cursor-pointer"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Thêm
                </button>
              </div>
            </form>

            {/* Danh sách thẻ địa điểm */}
            <div className="space-y-4">
              {project.locations.map((loc) => {
                const scenesHere = allScenes.filter((s) => s.locationId === loc.id);
                return (
                  <div
                    key={loc.id}
                    className="bg-[#FAF7F2] border border-[#D6D0C4] p-5 space-y-3 transition-colors hover:border-[#78716C]"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <input
                        type="text"
                        value={loc.name}
                        onChange={(e) =>
                          onUpdateLocation({ ...loc, name: e.target.value })
                        }
                        className="flex-1 font-serif-display text-xl font-bold text-[#1C1B18] bg-transparent border-b border-transparent focus:border-[#8B3A2B] focus:outline-none"
                      />
                      <button
                        type="button"
                        onClick={() => onDeleteLocation(loc.id)}
                        title="Xóa địa điểm"
                        className="text-[#78716C] hover:text-[#991B1B] p-1 transition-colors cursor-pointer"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>

                    <textarea
                      rows={2}
                      value={loc.description}
                      onChange={(e) =>
                        onUpdateLocation({ ...loc, description: e.target.value })
                      }
                      placeholder="Mô tả chi tiết kiến trúc, khí hậu, cảm xúc không gian..."
                      className="w-full text-sm text-[#57534E] bg-transparent border border-transparent focus:border-[#D6D0C4] focus:bg-[#F3EFE6]/50 p-1.5 focus:outline-none resize-none"
                    />

                    <div className="pt-2 border-t border-[#E6E0D4] flex flex-wrap items-center justify-between gap-2 text-xs text-[#78716C]">
                      <span className="font-mono-code">
                        Sử dụng trong <strong>{scenesHere.length}</strong> cảnh
                      </span>
                      <div className="flex flex-wrap gap-1.5">
                        {scenesHere.map((sc) => (
                          <button
                            key={sc.id}
                            type="button"
                            onClick={() => onSelectScene(sc.id)}
                            className="inline-flex items-center gap-1 text-[11px] bg-[#F3EFE6] hover:bg-[#E8E2D5] text-[#1C1B18] px-2 py-0.5 border border-[#D6D0C4] transition-colors cursor-pointer"
                          >
                            <span>{sc.title}</span>
                            <ArrowUpRight className="w-3 h-3 text-[#78716C]" />
                          </button>
                        ))}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
