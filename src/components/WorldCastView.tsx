import React, { useState } from 'react';
import { Plus, Trash2, ArrowUpRight, Tag as TagIcon, Check } from 'lucide-react';
import { Character, Location, Project, Prop, Tag, WorldEvent } from '../types/novelist';

interface WorldCastViewProps {
  project: Project;
  onAddCharacter: (name: string, role: string, description: string, tagIds?: number[]) => void;
  onUpdateCharacter: (character: Character) => void;
  onDeleteCharacter: (id: number) => void;
  onAddLocation: (name: string, description: string, tagIds?: number[]) => void;
  onUpdateLocation: (location: Location) => void;
  onDeleteLocation: (id: number) => void;
  onAddProp?: (name: string, category: string, description: string, significance: string, tagIds?: number[]) => void;
  onUpdateProp?: (prop: Prop) => void;
  onDeleteProp?: (id: number) => void;
  onAddEvent?: (title: string, timelineOrder: number, description: string, tagIds?: number[]) => void;
  onUpdateEvent?: (event: WorldEvent) => void;
  onDeleteEvent?: (id: number) => void;
  onAddTag?: (name: string) => void;
  onDeleteTag?: (id: number) => void;
  onSelectScene: (sceneId: number) => void;
}

type WorldTab = 'characters' | 'locations' | 'props' | 'events' | 'tags';

export const WorldCastView: React.FC<WorldCastViewProps> = ({
  project,
  onAddCharacter,
  onUpdateCharacter,
  onDeleteCharacter,
  onAddLocation,
  onUpdateLocation,
  onDeleteLocation,
  onAddProp,
  onUpdateProp,
  onDeleteProp,
  onAddEvent,
  onUpdateEvent,
  onDeleteEvent,
  onAddTag,
  onDeleteTag,
  onSelectScene,
}) => {
  const [activeTab, setActiveTab] = useState<WorldTab>('characters');
  const [selectedTagFilter, setSelectedTagFilter] = useState<number | 'all'>('all');

  // Quản lý Thẻ nhanh
  const [newTagName, setNewTagName] = useState('');

  // Tab 1: Nhân vật
  const [newCharName, setNewCharName] = useState('');
  const [newCharRole, setNewCharRole] = useState('Nhân vật chính');
  const [newCharDesc, setNewCharDesc] = useState('');
  const [newCharTags, setNewCharTags] = useState<number[]>([]);

  // Tab 2: Địa điểm
  const [newLocName, setNewLocName] = useState('');
  const [newLocDesc, setNewLocDesc] = useState('');
  const [newLocTags, setNewLocTags] = useState<number[]>([]);

  // Tab 3: Vật phẩm (Props)
  const [newPropName, setNewPropName] = useState('');
  const [newPropCat, setNewPropCat] = useState('Cổ vật');
  const [newPropDesc, setNewPropDesc] = useState('');
  const [newPropSig, setNewPropSig] = useState('');
  const [newPropTags, setNewPropTags] = useState<number[]>([]);

  // Tab 4: Sự kiện (Events)
  const [newEventTitle, setNewEventTitle] = useState('');
  const [newEventOrder, setNewEventOrder] = useState<number>(
    (project.events?.length || 0) + 1
  );
  const [newEventDesc, setNewEventDesc] = useState('');
  const [newEventTags, setNewEventTags] = useState<number[]>([]);

  const tags: Tag[] = project.tags || [];
  const charTags: Tag[] = tags.filter((t) => !t.entityType || t.entityType === 'character');
  const locTags: Tag[] = tags.filter((t) => t.entityType === 'location');
  const propTags: Tag[] = tags.filter((t) => t.entityType === 'prop');
  const eventTags: Tag[] = tags.filter((t) => t.entityType === 'event');

  // Danh sách tên thẻ theo từng danh mục (category-specific tag names)
  const charTagNames: string[] = charTags.map((t) => t.name);
  const locTagNames: string[] = locTags.map((t) => t.name);
  const propTagNames: string[] = propTags.map((t) => t.name);
  const eventTagNames: string[] = eventTags.map((t) => t.name);
  const propsList: Prop[] = project.props || [];
  const eventsList: WorldEvent[] = [...(project.events || [])].sort(
    (a, b) => a.timelineOrder - b.timelineOrder
  );

  const allScenes = project.acts.flatMap((a) =>
    a.chapters.flatMap((c) =>
      c.scenes.map((sc) => ({
        ...sc,
        chapterTitle: c.title,
        actTitle: a.title,
      }))
    )
  );

  const matchesTagFilter = (entityTagIds?: number[]) => {
    if (selectedTagFilter === 'all') return true;
    return (entityTagIds || []).includes(selectedTagFilter);
  };

  const toggleTagId = (list: number[], id: number): number[] =>
    list.includes(id) ? list.filter((x) => x !== id) : [...list, id];

  const handleCreateTag = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTagName.trim() || !onAddTag) return;
    onAddTag(newTagName.trim());
    setNewTagName('');
  };

  const handleCreateCharacter = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newCharName.trim()) return;
    onAddCharacter(
      newCharName.trim(),
      newCharRole.trim() || 'Nhân vật phụ',
      newCharDesc.trim(),
      newCharTags
    );
    setNewCharName('');
    setNewCharDesc('');
    setNewCharTags([]);
  };

  const handleCreateLocation = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newLocName.trim()) return;
    onAddLocation(newLocName.trim(), newLocDesc.trim(), newLocTags);
    setNewLocName('');
    setNewLocDesc('');
    setNewLocTags([]);
  };

  const handleCreateProp = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newPropName.trim() || !onAddProp) return;
    onAddProp(
      newPropName.trim(),
      newPropCat.trim() || 'Cổ vật',
      newPropDesc.trim(),
      newPropSig.trim(),
      newPropTags
    );
    setNewPropName('');
    setNewPropDesc('');
    setNewPropSig('');
    setNewPropTags([]);
  };

  const handleCreateEvent = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newEventTitle.trim() || !onAddEvent) return;
    onAddEvent(
      newEventTitle.trim(),
      Number(newEventOrder) || 1,
      newEventDesc.trim(),
      newEventTags
    );
    setNewEventTitle('');
    setNewEventOrder(eventsList.length + 2);
    setNewEventDesc('');
    setNewEventTags([]);
  };

  const filteredCharacters = project.characters.filter((c) =>
    matchesTagFilter(c.tagIds)
  );
  const filteredLocations = project.locations.filter((l) =>
    matchesTagFilter(l.tagIds)
  );
  const filteredProps = propsList.filter((p) => matchesTagFilter(p.tagIds));
  const filteredEvents = eventsList.filter((ev) => matchesTagFilter(ev.tagIds));

  const renderTagButton = (
    t: Tag,
    active: boolean,
    onClick: () => void,
    size: 'sm' | 'xs' = 'sm'
  ) => {
    const col = t.color || '#3498db';
    return (
      <button
        key={t.id}
        type="button"
        onClick={onClick}
        style={{
          color: col,
          borderColor: active ? col : '#D6D0C4',
          backgroundColor: active ? `${col}18` : '#FAF7F2',
        }}
        className={`inline-flex items-center gap-1.5 font-medium border rounded transition-all cursor-pointer ${
          size === 'sm' ? 'px-2.5 py-1 text-xs' : 'px-2 py-0.5 text-[11px]'
        }`}
      >
        <span
          className="w-1.5 h-1.5 rounded-full inline-block"
          style={{ backgroundColor: col }}
        />
        <span>#{t.name}</span>
        {active && <Check className="w-3 h-3 ml-0.5" />}
      </button>
    );
  };

  return (
    <div className="flex-1 overflow-y-auto bg-[#FAF7F2] px-8 py-8">
      <div className="max-w-6xl mx-auto space-y-8">
        {/* Tiêu đề Trung tâm Xây dựng Thế giới */}
        <div className="border-b border-[#D6D0C4] pb-6 flex flex-col md:flex-row md:items-end justify-between gap-4">
          <div>
            <p className="text-xs uppercase tracking-[0.14em] text-[#78716C] font-semibold">
              Trung Tâm Xây Dựng Thế Giới &amp; Hệ Thống Thẻ Đa Năng (ui_worldbuilding.go)
            </p>
            <h1 className="font-serif-display text-3xl md:text-4xl font-bold text-[#1C1B18] mt-1">
              Thế Giới Tiểu Thuyết — {project.title}
            </h1>
          </div>
          <div className="flex flex-wrap items-center gap-4 text-xs font-mono-code text-[#57534E]">
            <span>{project.characters.length} Nhân vật</span>
            <span>&bull;</span>
            <span>{project.locations.length} Địa điểm</span>
            <span>&bull;</span>
            <span>{propsList.length} Vật phẩm</span>
            <span>&bull;</span>
            <span>{eventsList.length} Sự kiện</span>
            <span>&bull;</span>
            <span>{tags.length} Thẻ</span>
          </div>
        </div>

        {/* Thanh Quản lý Thẻ nhanh & Bộ lọc theo Thẻ */}
        <div className="bg-[#F3EFE6] border border-[#D6D0C4] p-4 flex flex-col lg:flex-row items-stretch lg:items-center justify-between gap-4">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xs font-semibold uppercase tracking-wider text-[#57534E] mr-1">
              Lọc theo thẻ:
            </span>
            <button
              type="button"
              onClick={() => setSelectedTagFilter('all')}
              className={`px-3 py-1 text-xs font-medium border transition-colors cursor-pointer ${
                selectedTagFilter === 'all'
                  ? 'bg-[#1C1B18] text-[#FAF7F2] border-[#1C1B18]'
                  : 'bg-[#FAF7F2] text-[#57534E] border-[#D6D0C4] hover:border-[#1C1B18]'
              }`}
            >
              Tất cả thẻ ({tags.length})
            </button>
            {tags.map((t) => (
              <button
                key={t.id}
                type="button"
                onClick={() =>
                  setSelectedTagFilter(selectedTagFilter === t.id ? 'all' : t.id)
                }
                className={`px-3 py-1 text-xs font-medium border transition-colors cursor-pointer ${
                  selectedTagFilter === t.id
                    ? 'bg-[#8B3A2B] text-[#FAF7F2] border-[#8B3A2B]'
                    : 'bg-[#FAF7F2] text-[#1C1B18] border-[#D6D0C4] hover:border-[#8B3A2B]'
                }`}
              >
                #{t.name}
              </button>
            ))}
          </div>

          {onAddTag && (
            <form onSubmit={handleCreateTag} className="flex items-center gap-2">
              <input
                type="text"
                value={newTagName}
                onChange={(e) => setNewTagName(e.target.value)}
                placeholder="Tạo thẻ mới (VD: Thiên giới, Khu vực cấm)..."
                className="bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-1.5 text-xs text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B] min-w-[240px]"
              />
              <button
                type="submit"
                className="inline-flex items-center gap-1 bg-[#1C1B18] text-[#FAF7F2] px-3 py-1.5 text-xs font-medium hover:bg-[#3A3832] transition-colors cursor-pointer whitespace-nowrap"
              >
                <Plus className="w-3.5 h-3.5" />
                Thêm Thẻ
              </button>
            </form>
          )}
        </div>

        {/* 5 Tab Quản lý Thế giới tương ứng với container.NewAppTabs trong ui_worldbuilding.go */}
        <div className="border-b border-[#D6D0C4] flex flex-wrap gap-2">
          {(
            [
              ['characters', `Nhân vật (${filteredCharacters.length})`],
              ['locations', `Địa điểm (${filteredLocations.length})`],
              ['props', `Vật phẩm (${filteredProps.length})`],
              ['events', `Sự kiện (${filteredEvents.length})`],
              ['tags', `Quản lý Thẻ (${tags.length})`],
            ] as const
          ).map(([tabKey, label]) => (
            <button
              key={tabKey}
              type="button"
              onClick={() => setActiveTab(tabKey)}
              className={`px-5 py-2.5 text-xs font-semibold uppercase tracking-wider border-b-2 transition-colors cursor-pointer ${
                activeTab === tabKey
                  ? 'border-[#8B3A2B] text-[#8B3A2B] bg-[#F3EFE6]/60'
                  : 'border-transparent text-[#57534E] hover:text-[#1C1B18]'
              }`}
            >
              {label}
            </button>
          ))}
        </div>

        {/* TAB 1: NHÂN VẬT */}
        {activeTab === 'characters' && (
          <div className="space-y-6">
            <form
              onSubmit={handleCreateCharacter}
              className="bg-[#F3EFE6] border border-[#D6D0C4] p-5 space-y-4"
            >
              <div className="text-xs uppercase tracking-[0.08em] font-semibold text-[#57534E]">
                Thêm Nhân Vật Mới (Hỗ trợ gõ Tiếng Việt &amp; Gắn Thẻ)
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
              <input
                type="text"
                value={newCharDesc}
                onChange={(e) => setNewCharDesc(e.target.value)}
                placeholder="Tiểu sử ngắn, ngoại hình, tính cách, động cơ..."
                className="w-full bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
              />
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs text-[#57534E] font-medium">Gắn thẻ nhân vật ({charTagNames.length}):</span>
                  {charTags.length === 0 ? (
                    <span className="text-xs italic text-[#78716C]">(Chưa có thẻ nhân vật nào)</span>
                  ) : (
                    charTags.map((t) =>
                      renderTagButton(
                        t,
                        newCharTags.includes(t.id),
                        () => setNewCharTags((prev) => toggleTagId(prev, t.id)),
                        'sm'
                      )
                    )
                  )}
                </div>
                <button
                  type="submit"
                  className="inline-flex items-center gap-1.5 bg-[#8B3A2B] text-[#FAF7F2] px-4 py-2 text-xs font-medium tracking-wide hover:bg-[#722E21] transition-colors cursor-pointer"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Lưu Nhân Vật
                </button>
              </div>
            </form>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {filteredCharacters.map((char) => {
                const povScenes = allScenes.filter((s) => s.povCharacterId === char.id);
                const presentScenes = allScenes.filter((s) =>
                  s.characterIds.includes(char.id)
                );
                const charTagIds = char.tagIds || [];

                return (
                  <div
                    key={char.id}
                    className="bg-[#FAF7F2] border border-[#D6D0C4] p-5 space-y-3 hover:border-[#78716C] transition-colors"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex-1 space-y-1">
                        <input
                          type="text"
                          value={char.name}
                          onChange={(e) =>
                            onUpdateCharacter({ ...char, name: e.target.value })
                          }
                          className="w-full font-serif-display text-xl font-bold text-[#1C1B18] bg-transparent border-b border-transparent focus:border-[#8B3A2B] focus:outline-none"
                        />
                        <input
                          type="text"
                          value={char.role}
                          onChange={(e) =>
                            onUpdateCharacter({ ...char, role: e.target.value })
                          }
                          className="text-xs font-mono-code uppercase tracking-wider text-[#8B3A2B] bg-transparent border-b border-transparent focus:border-[#8B3A2B] focus:outline-none"
                        />
                      </div>
                      <button
                        type="button"
                        onClick={() => onDeleteCharacter(char.id)}
                        title="Xóa nhân vật"
                        className="text-[#78716C] hover:text-[#991B1B] p-1 cursor-pointer"
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
                      className="w-full text-sm text-[#57534E] bg-transparent border border-transparent focus:border-[#D6D0C4] focus:bg-[#F3EFE6]/50 p-1.5 focus:outline-none resize-none"
                    />

                    {/* Gắn/bỏ gắn thẻ trực tiếp */}
                    <div className="flex flex-wrap items-center gap-1.5 pt-1">
                      <span className="text-[11px] text-[#78716C] mr-1">Thẻ ({charTagNames.length}):</span>
                      {charTags.map((t) =>
                        renderTagButton(
                          t,
                          charTagIds.includes(t.id),
                          () =>
                            onUpdateCharacter({
                              ...char,
                              tagIds: toggleTagId(charTagIds, t.id),
                            }),
                          'xs'
                        )
                      )}
                    </div>

                    <div className="pt-2 border-t border-[#E6E0D4] flex flex-wrap items-center justify-between gap-2 text-xs text-[#78716C]">
                      <span className="font-mono-code">
                        POV: <strong>{povScenes.length}</strong> cảnh &bull; Xuất hiện:{' '}
                        <strong>{presentScenes.length}</strong> cảnh
                      </span>
                      <div className="flex flex-wrap gap-1">
                        {presentScenes.slice(0, 2).map((sc) => (
                          <button
                            key={sc.id}
                            type="button"
                            onClick={() => onSelectScene(sc.id)}
                            className="inline-flex items-center gap-1 text-[11px] bg-[#F3EFE6] hover:bg-[#E8E2D5] text-[#1C1B18] px-2 py-0.5 border border-[#D6D0C4] cursor-pointer"
                          >
                            <span>{sc.title}</span>
                            <ArrowUpRight className="w-3 h-3" />
                          </button>
                        ))}
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* TAB 2: ĐỊA ĐIỂM */}
        {activeTab === 'locations' && (
          <div className="space-y-6">
            <form
              onSubmit={handleCreateLocation}
              className="bg-[#F3EFE6] border border-[#D6D0C4] p-5 space-y-4"
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
              <input
                type="text"
                value={newLocDesc}
                onChange={(e) => setNewLocDesc(e.target.value)}
                placeholder="Mô tả không gian, kiến trúc, ánh sáng, âm thanh..."
                className="w-full bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
              />
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs text-[#57534E] font-medium">Gắn thẻ địa điểm ({locTagNames.length}):</span>
                  {locTags.length === 0 ? (
                    <span className="text-xs italic text-[#78716C]">(Chưa có thẻ địa điểm nào)</span>
                  ) : (
                    locTags.map((t) =>
                      renderTagButton(
                        t,
                        newLocTags.includes(t.id),
                        () => setNewLocTags((prev) => toggleTagId(prev, t.id)),
                        'sm'
                      )
                    )
                  )}
                </div>
                <button
                  type="submit"
                  className="inline-flex items-center gap-1.5 bg-[#8B3A2B] text-[#FAF7F2] px-4 py-2 text-xs font-medium tracking-wide hover:bg-[#722E21] transition-colors cursor-pointer"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Lưu Địa Điểm
                </button>
              </div>
            </form>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {filteredLocations.map((loc) => {
                const scenesHere = allScenes.filter((s) => s.locationId === loc.id);
                const locTagIds = loc.tagIds || [];
                return (
                  <div
                    key={loc.id}
                    className="bg-[#FAF7F2] border border-[#D6D0C4] p-5 space-y-3 hover:border-[#78716C] transition-colors"
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
                        className="text-[#78716C] hover:text-[#991B1B] p-1 cursor-pointer"
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
                      className="w-full text-sm text-[#57534E] bg-transparent border border-transparent focus:border-[#D6D0C4] p-1.5 focus:outline-none resize-none"
                    />
                    <div className="flex flex-wrap items-center gap-1.5">
                      <span className="text-[11px] text-[#78716C] mr-1">Thẻ ({locTagNames.length}):</span>
                      {locTags.map((t) =>
                        renderTagButton(
                          t,
                          locTagIds.includes(t.id),
                          () =>
                            onUpdateLocation({
                              ...loc,
                              tagIds: toggleTagId(locTagIds, t.id),
                            }),
                          'xs'
                        )
                      )}
                    </div>
                    <div className="pt-2 border-t border-[#E6E0D4] text-xs text-[#78716C] font-mono-code">
                      Sử dụng trong <strong>{scenesHere.length}</strong> cảnh
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* TAB 3: VẬT PHẨM (PROPS) */}
        {activeTab === 'props' && (
          <div className="space-y-6">
            <form
              onSubmit={handleCreateProp}
              className="bg-[#F3EFE6] border border-[#D6D0C4] p-5 space-y-4"
            >
              <div className="text-xs uppercase tracking-[0.08em] font-semibold text-[#57534E]">
                Thêm Vật Phẩm / Cổ Vật Mới (Bảng SQLite: props &amp; scene_props)
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <input
                  type="text"
                  value={newPropName}
                  onChange={(e) => setNewPropName(e.target.value)}
                  placeholder="Tên vật phẩm (VD: Thấu Kính Hải Đăng Cổ)..."
                  className="sm:col-span-2 bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
                />
                <input
                  type="text"
                  value={newPropCat}
                  onChange={(e) => setNewPropCat(e.target.value)}
                  placeholder="Phân loại (VD: Cổ vật, Tín vật)"
                  className="bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
                />
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <input
                  type="text"
                  value={newPropDesc}
                  onChange={(e) => setNewPropDesc(e.target.value)}
                  placeholder="Mô tả hình dáng, chất liệu, nguồn gốc..."
                  className="bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
                />
                <input
                  type="text"
                  value={newPropSig}
                  onChange={(e) => setNewPropSig(e.target.value)}
                  placeholder="Ý nghĩa cốt truyện (Significance)..."
                  className="bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
                />
              </div>
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs text-[#57534E] font-medium">Gắn thẻ vật phẩm ({propTagNames.length}):</span>
                  {propTags.length === 0 ? (
                    <span className="text-xs italic text-[#78716C]">(Chưa có thẻ vật phẩm nào)</span>
                  ) : (
                    propTags.map((t) =>
                      renderTagButton(
                        t,
                        newPropTags.includes(t.id),
                        () => setNewPropTags((prev) => toggleTagId(prev, t.id)),
                        'sm'
                      )
                    )
                  )}
                </div>
                <button
                  type="submit"
                  className="inline-flex items-center gap-1.5 bg-[#8B3A2B] text-[#FAF7F2] px-4 py-2 text-xs font-medium tracking-wide hover:bg-[#722E21] transition-colors cursor-pointer"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Lưu Vật Phẩm
                </button>
              </div>
            </form>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {filteredProps.map((prop) => {
                const scenesWithProp = allScenes.filter((s) =>
                  (s.propIds || []).includes(prop.id)
                );
                const propTagIds = prop.tagIds || [];
                return (
                  <div
                    key={prop.id}
                    className="bg-[#FAF7F2] border border-[#D6D0C4] p-5 space-y-3 hover:border-[#78716C] transition-colors"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <div className="font-serif-display text-xl font-bold text-[#1C1B18]">
                          🧭 {prop.name}
                        </div>
                        <div className="text-xs font-mono-code uppercase tracking-wider text-[#8B3A2B]">
                          {prop.category}
                        </div>
                      </div>
                      {onDeleteProp && (
                        <button
                          type="button"
                          onClick={() => onDeleteProp(prop.id)}
                          className="text-[#78716C] hover:text-[#991B1B] p-1 cursor-pointer"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      )}
                    </div>
                    <p className="text-sm text-[#57534E]">{prop.description}</p>
                    {prop.significance && (
                      <p className="text-xs text-[#1C1B18] bg-[#F3EFE6] p-2.5 border-l-2 border-[#8B3A2B]">
                        <strong>Ý nghĩa cốt truyện:</strong> {prop.significance}
                      </p>
                    )}
                    {onUpdateProp && (
                      <div className="flex flex-wrap items-center gap-1.5">
                        <span className="text-[11px] text-[#78716C] mr-1">Thẻ ({propTagNames.length}):</span>
                        {propTags.map((t) =>
                          renderTagButton(
                            t,
                            propTagIds.includes(t.id),
                            () =>
                              onUpdateProp({
                                ...prop,
                                tagIds: toggleTagId(propTagIds, t.id),
                              }),
                            'xs'
                          )
                        )}
                      </div>
                    )}
                    <div className="pt-2 border-t border-[#E6E0D4] text-xs text-[#78716C] font-mono-code">
                      Xuất hiện trong <strong>{scenesWithProp.length}</strong> cảnh
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* TAB 4: SỰ KIỆN (EVENTS TIMELINE) */}
        {activeTab === 'events' && (
          <div className="space-y-6">
            <form
              onSubmit={handleCreateEvent}
              className="bg-[#F3EFE6] border border-[#D6D0C4] p-5 space-y-4"
            >
              <div className="text-xs uppercase tracking-[0.08em] font-semibold text-[#57534E]">
                Thêm Sự Kiện Dòng Thời Gian Mới (Bảng SQLite: events &amp; scene_events)
              </div>
              <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
                <input
                  type="text"
                  value={newEventTitle}
                  onChange={(e) => setNewEventTitle(e.target.value)}
                  placeholder="Tiêu đề sự kiện..."
                  className="sm:col-span-3 bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
                />
                <input
                  type="number"
                  min={1}
                  value={newEventOrder}
                  onChange={(e) => setNewEventOrder(Number(e.target.value) || 1)}
                  placeholder="Thứ tự thời gian"
                  className="bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm font-mono-code text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
                />
              </div>
              <input
                type="text"
                value={newEventDesc}
                onChange={(e) => setNewEventDesc(e.target.value)}
                placeholder="Diễn biến chính và hệ quả đối với mạch truyện..."
                className="w-full bg-[#FAF7F2] border border-[#D6D0C4] px-3 py-2 text-sm text-[#1C1B18] focus:outline-none focus:border-[#8B3A2B]"
              />
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-xs text-[#57534E] font-medium">Gắn thẻ sự kiện ({eventTagNames.length}):</span>
                  {eventTags.length === 0 ? (
                    <span className="text-xs italic text-[#78716C]">(Chưa có thẻ sự kiện nào)</span>
                  ) : (
                    eventTags.map((t) => {
                      const active = newEventTags.includes(t.id);
                      return renderTagButton(
                        t,
                        active,
                        () => setNewEventTags((prev) => toggleTagId(prev, t.id)),
                        'sm'
                      );
                    })
                  )}
                </div>
                <button
                  type="submit"
                  className="inline-flex items-center gap-1.5 bg-[#8B3A2B] text-[#FAF7F2] px-4 py-2 text-xs font-medium tracking-wide hover:bg-[#722E21] transition-colors cursor-pointer"
                >
                  <Plus className="w-3.5 h-3.5" />
                  Lưu Sự Kiện
                </button>
              </div>
            </form>

            <div className="space-y-4">
              {filteredEvents.map((ev) => {
                const scenesWithEvent = allScenes.filter((s) =>
                  (s.eventIds || []).includes(ev.id)
                );
                const evTagIds = ev.tagIds || [];
                return (
                  <div
                    key={ev.id}
                    className="bg-[#FAF7F2] border border-[#D6D0C4] p-5 space-y-3 hover:border-[#78716C] transition-colors"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex items-center gap-3">
                        <span className="px-2.5 py-1 bg-[#1C1B18] text-[#FAF7F2] font-mono-code text-xs font-semibold">
                          Mốc #{ev.timelineOrder}
                        </span>
                        <h3 className="font-serif-display text-xl font-bold text-[#1C1B18]">
                          {ev.title}
                        </h3>
                      </div>
                      {onDeleteEvent && (
                        <button
                          type="button"
                          onClick={() => onDeleteEvent(ev.id)}
                          className="text-[#78716C] hover:text-[#991B1B] p-1 cursor-pointer"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      )}
                    </div>
                    <p className="text-sm text-[#57534E]">{ev.description}</p>
                    {onUpdateEvent && (
                      <div className="flex flex-wrap items-center gap-1.5">
                        <span className="text-[11px] text-[#78716C] mr-1">Thẻ ({eventTagNames.length}):</span>
                        {eventTags.map((t) =>
                          renderTagButton(
                            t,
                            evTagIds.includes(t.id),
                            () =>
                              onUpdateEvent({
                                ...ev,
                                tagIds: toggleTagId(evTagIds, t.id),
                              }),
                            'xs'
                          )
                        )}
                      </div>
                    )}
                    <div className="pt-2 border-t border-[#E6E0D4] text-xs text-[#78716C] font-mono-code">
                      Liên kết trong <strong>{scenesWithEvent.length}</strong> cảnh
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* TAB 5: QUẢN LÝ THẺ */}
        {activeTab === 'tags' && (
          <div className="bg-[#FAF7F2] border border-[#D6D0C4] p-6 space-y-4">
            <h2 className="font-serif-display text-2xl font-bold text-[#1C1B18]">
              Danh Sách Thẻ Phân Loại Đa Năng (Bảng tags &amp; entity_tags)
            </h2>
            <p className="text-xs text-[#57534E]">
              Thẻ có thể được gắn đồng thời cho Nhân vật, Địa điểm, Vật phẩm và Sự kiện để lọc nhanh theo tuyến cốt truyện hoặc thế giới quan.
            </p>
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3 pt-2">
              {tags.map((t) => (
                <div
                  key={t.id}
                  className="flex items-center justify-between bg-[#F3EFE6] border border-[#D6D0C4] px-4 py-3"
                >
                  <div className="flex items-center gap-2">
                    <span
                      className="w-2.5 h-2.5 rounded-full inline-block shrink-0"
                      style={{ backgroundColor: t.color || '#3498db' }}
                    />
                    <span
                      className="font-mono-code text-sm font-semibold"
                      style={{ color: t.color || '#3498db' }}
                    >
                      #{t.name}
                    </span>
                  </div>
                  {onDeleteTag && (
                    <button
                      type="button"
                      onClick={() => onDeleteTag(t.id)}
                      className="text-[#78716C] hover:text-[#991B1B] p-1 cursor-pointer"
                      title="Xóa thẻ"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
