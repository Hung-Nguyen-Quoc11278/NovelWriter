/**
 * @license
 * SPDX-License-Identifier: Apache-2.0
 */

import React, { useEffect, useMemo, useRef, useState } from 'react';
import {
  ArrowDown,
  ArrowUp,
  Check,
  ChevronDown,
  ChevronRight,
  Download,
  Edit3,
  Eye,
  Maximize2,
  Minimize2,
  Plus,
  Search,
  Trash2,
} from 'lucide-react';
import {
  ALL_STATUSES,
  Act,
  Chapter,
  Character,
  Location,
  Project,
  Prop,
  Scene,
  SceneStatus,
  Tag,
  WorldEvent,
  countWords,
  normalizeStatus,
} from './types/novelist';
import { INITIAL_PROJECTS } from './data/initialNovelData';
import { WorldCastView } from './components/WorldCastView';
import { ExportStudioView } from './components/ExportStudioView';
import { GoSourceExplorer } from './components/GoSourceExplorer';

type NavTab = 'studio' | 'world' | 'export' | 'go-source' | 'architecture';

const STORAGE_KEY = 'gonovelist_sqlite_vi_v4';

export default function App() {
  const [projects, setProjects] = useState<Project[]>(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) {
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed) && parsed.length > 0) {
          return parsed;
        }
      }
    } catch {
      // Dùng dữ liệu mẫu mặc định
    }
    return INITIAL_PROJECTS;
  });

  const [activeProjectId, setActiveProjectId] = useState<number>(
    () => projects[0]?.id || 1
  );
  const [activeTab, setActiveTab] = useState<NavTab>('studio');
  const [distractionFree, setDistractionFree] = useState(false);

  const activeProject = useMemo(
    () => projects.find((p) => p.id === activeProjectId) || projects[0],
    [projects, activeProjectId]
  );

  // ID Cảnh đang chọn
  const [selectedSceneId, setSelectedSceneId] = useState<number>(() => {
    const firstScene =
      activeProject?.acts[0]?.chapters[0]?.scenes[0]?.id || 1111;
    return firstScene;
  });

  // Trạng thái đóng/mở nhánh cây
  const [collapsedActs, setCollapsedActs] = useState<Record<number, boolean>>(
    {}
  );
  const [collapsedChapters, setCollapsedChapters] = useState<
    Record<number, boolean>
  >({});
  const [treeSearch, setTreeSearch] = useState('');

  // Đổi tên trực tiếp
  const [renamingNode, setRenamingNode] = useState<{
    kind: 'act' | 'chapter' | 'scene';
    id: number;
    title: string;
  } | null>(null);

  // Tạo tác phẩm mới
  const [showNewBookForm, setShowNewBookForm] = useState(false);
  const [newBookTitle, setNewBookTitle] = useState('');
  const [newBookAuthor, setNewBookAuthor] = useState('');

  // Trạng thái trình soạn thảo & Bộ tự động lưu trễ 750ms
  const [editorMode, setEditorMode] = useState<'write' | 'preview'>('write');
  const [inspectorTab, setInspectorTab] = useState<'context' | 'notes'>(
    'context'
  );
  const [saveStatus, setSaveStatus] = useState<'saved' | 'pending'>('saved');
  const [lastSavedClock, setLastSavedClock] = useState<string>(() =>
    new Date().toLocaleTimeString('vi-VN', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  );

  // Thêm nhanh Nhân vật / Bối cảnh ngay trong thanh bên phải
  const [quickCharName, setQuickCharName] = useState('');
  const [quickLocName, setQuickLocName] = useState('');

  const debounceTimerRef = useRef<number | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  // Tìm Hồi, Chương và Cảnh hiện tại
  const activeContext = useMemo(() => {
    for (const act of activeProject.acts) {
      for (const chapter of act.chapters) {
        for (const scene of chapter.scenes) {
          if (scene.id === selectedSceneId) {
            return { act, chapter, scene };
          }
        }
      }
    }
    for (const act of activeProject.acts) {
      for (const chapter of act.chapters) {
        if (chapter.scenes.length > 0) {
          return { act, chapter, scene: chapter.scenes[0] };
        }
      }
    }
    return null;
  }, [activeProject, selectedSceneId]);

  // Tự động lưu trễ 750ms (mô phỏng cơ chế time.AfterFunc trong ui_editor.go)
  useEffect(() => {
    setSaveStatus('pending');
    if (debounceTimerRef.current) {
      window.clearTimeout(debounceTimerRef.current);
    }
    debounceTimerRef.current = window.setTimeout(() => {
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(projects));
      } catch {
        // Bỏ qua nếu đầy bộ nhớ trình duyệt
      }
      setSaveStatus('saved');
      setLastSavedClock(
        new Date().toLocaleTimeString('vi-VN', {
          hour: '2-digit',
          minute: '2-digit',
          second: '2-digit',
        })
      );
    }, 750);

    return () => {
      if (debounceTimerRef.current) {
        window.clearTimeout(debounceTimerRef.current);
      }
    };
  }, [projects]);

  const updateActiveProject = (updater: (proj: Project) => Project) => {
    setProjects((prev) =>
      prev.map((p) => (p.id === activeProject.id ? updater(p) : p))
    );
  };

  const updateActiveScene = (updater: (sc: Scene) => Scene) => {
    if (!activeContext) return;
    const targetId = activeContext.scene.id;
    updateActiveProject((proj) => ({
      ...proj,
      updatedAt: new Date().toISOString(),
      acts: proj.acts.map((act) => ({
        ...act,
        chapters: act.chapters.map((ch) => ({
          ...ch,
          scenes: ch.scenes.map((sc) => {
            if (sc.id !== targetId) return sc;
            const updated = updater(sc);
            return {
              ...updated,
              wordCount: countWords(updated.content),
              updatedAt: new Date().toISOString(),
            };
          }),
        })),
      })),
    }));
  };

  const updateChapterTarget = (chapterId: number, targetWords: number) => {
    updateActiveProject((proj) => ({
      ...proj,
      acts: proj.acts.map((act) => ({
        ...act,
        chapters: act.chapters.map((ch) =>
          ch.id === chapterId ? { ...ch, targetWords } : ch
        ),
      })),
    }));
  };

  // Thao tác thêm/sửa/xóa/di chuyển trên cây Hồi -> Chương -> Cảnh
  const handleAddAct = () => {
    const nextNum = activeProject.acts.length + 1;
    const newActId = Date.now();
    const newChapId = newActId + 1;
    const newSceneId = newActId + 2;
    const newAct: Act = {
      id: newActId,
      projectId: activeProject.id,
      title: `Hồi ${nextNum}: Chuyển Đoạn Mới`,
      position: nextNum,
      chapters: [
        {
          id: newChapId,
          actId: newActId,
          title: `Chương 1: Khởi Đầu Mới`,
          targetWords: 3000,
          position: 1,
          scenes: [
            {
              id: newSceneId,
              chapterId: newChapId,
              title: 'Cảnh 1: Phân Cảnh Mở Đầu',
              summary: '',
              content: '',
              sideNotes: '',
              status: 'Ý tưởng',
              povCharacterId: activeProject.characters[0]?.id ?? null,
              locationId: activeProject.locations[0]?.id ?? null,
              characterIds: activeProject.characters[0]
                ? [activeProject.characters[0].id]
                : [],
              targetWords: 1200,
              wordCount: 0,
              position: 1,
              updatedAt: new Date().toISOString(),
            },
          ],
        },
      ],
    };
    updateActiveProject((proj) => ({
      ...proj,
      acts: [...proj.acts, newAct],
    }));
    setSelectedSceneId(newSceneId);
  };

  const handleAddChapter = (actId: number) => {
    const newChapId = Date.now();
    const newSceneId = newChapId + 1;
    updateActiveProject((proj) => ({
      ...proj,
      acts: proj.acts.map((act) => {
        if (act.id !== actId) return act;
        const nextNum = act.chapters.length + 1;
        const newChap: Chapter = {
          id: newChapId,
          actId: act.id,
          title: `Chương ${nextNum}: Chương Mới`,
          targetWords: 3000,
          position: nextNum,
          scenes: [
            {
              id: newSceneId,
              chapterId: newChapId,
              title: 'Cảnh 1: Bản Nháp Đầu Tiên',
              summary: '',
              content: '',
              sideNotes: '',
              status: 'Ý tưởng',
              povCharacterId: proj.characters[0]?.id ?? null,
              locationId: proj.locations[0]?.id ?? null,
              characterIds: [],
              targetWords: 1200,
              wordCount: 0,
              position: 1,
              updatedAt: new Date().toISOString(),
            },
          ],
        };
        return { ...act, chapters: [...act.chapters, newChap] };
      }),
    }));
    setCollapsedActs((prev) => ({ ...prev, [actId]: false }));
    setSelectedSceneId(newSceneId);
  };

  const handleAddScene = (chapterId: number) => {
    const newSceneId = Date.now();
    updateActiveProject((proj) => ({
      ...proj,
      acts: proj.acts.map((act) => ({
        ...act,
        chapters: act.chapters.map((ch) => {
          if (ch.id !== chapterId) return ch;
          const nextNum = ch.scenes.length + 1;
          const newSc: Scene = {
            id: newSceneId,
            chapterId: ch.id,
            title: `Cảnh ${nextNum}: Cảnh Chưa Đặt Tên`,
            summary: '',
            content: '',
            sideNotes: '',
            status: 'Ý tưởng',
            povCharacterId: proj.characters[0]?.id ?? null,
            locationId: proj.locations[0]?.id ?? null,
            characterIds: [],
            targetWords: 1200,
            wordCount: 0,
            position: nextNum,
            updatedAt: new Date().toISOString(),
          };
          return { ...ch, scenes: [...ch.scenes, newSc] };
        }),
      })),
    }));
    setCollapsedChapters((prev) => ({ ...prev, [chapterId]: false }));
    setSelectedSceneId(newSceneId);
  };

  const handleCommitRename = () => {
    if (!renamingNode || !renamingNode.title.trim()) {
      setRenamingNode(null);
      return;
    }
    const clean = renamingNode.title.trim();
    updateActiveProject((proj) => ({
      ...proj,
      acts: proj.acts.map((act) => {
        if (renamingNode.kind === 'act' && act.id === renamingNode.id) {
          return { ...act, title: clean };
        }
        return {
          ...act,
          chapters: act.chapters.map((ch) => {
            if (renamingNode.kind === 'chapter' && ch.id === renamingNode.id) {
              return { ...ch, title: clean };
            }
            return {
              ...ch,
              scenes: ch.scenes.map((sc) =>
                renamingNode.kind === 'scene' && sc.id === renamingNode.id
                  ? { ...sc, title: clean }
                  : sc
              ),
            };
          }),
        };
      }),
    }));
    setRenamingNode(null);
  };

  const handleMoveAct = (actIndex: number, dir: -1 | 1) => {
    updateActiveProject((proj) => {
      const targetIdx = actIndex + dir;
      if (targetIdx < 0 || targetIdx >= proj.acts.length) return proj;
      const copy = [...proj.acts];
      const [item] = copy.splice(actIndex, 1);
      copy.splice(targetIdx, 0, item);
      return {
        ...proj,
        acts: copy.map((a, idx) => ({ ...a, position: idx + 1 })),
      };
    });
  };

  const handleMoveChapter = (
    actId: number,
    chapIndex: number,
    dir: -1 | 1
  ) => {
    updateActiveProject((proj) => ({
      ...proj,
      acts: proj.acts.map((act) => {
        if (act.id !== actId) return act;
        const targetIdx = chapIndex + dir;
        if (targetIdx < 0 || targetIdx >= act.chapters.length) return act;
        const copy = [...act.chapters];
        const [item] = copy.splice(chapIndex, 1);
        copy.splice(targetIdx, 0, item);
        return {
          ...act,
          chapters: copy.map((c, idx) => ({ ...c, position: idx + 1 })),
        };
      }),
    }));
  };

  const handleMoveScene = (
    chapterId: number,
    sceneIndex: number,
    dir: -1 | 1
  ) => {
    updateActiveProject((proj) => ({
      ...proj,
      acts: proj.acts.map((act) => ({
        ...act,
        chapters: act.chapters.map((ch) => {
          if (ch.id !== chapterId) return ch;
          const targetIdx = sceneIndex + dir;
          if (targetIdx < 0 || targetIdx >= ch.scenes.length) return ch;
          const copy = [...ch.scenes];
          const [item] = copy.splice(sceneIndex, 1);
          copy.splice(targetIdx, 0, item);
          return {
            ...ch,
            scenes: copy.map((s, idx) => ({ ...s, position: idx + 1 })),
          };
        }),
      })),
    }));
  };

  const handleDeleteNode = (
    kind: 'act' | 'chapter' | 'scene',
    id: number
  ) => {
    updateActiveProject((proj) => {
      if (kind === 'act') {
        if (proj.acts.length <= 1) return proj;
        return { ...proj, acts: proj.acts.filter((a) => a.id !== id) };
      }
      if (kind === 'chapter') {
        return {
          ...proj,
          acts: proj.acts.map((a) => ({
            ...a,
            chapters:
              a.chapters.length > 1
                ? a.chapters.filter((c) => c.id !== id)
                : a.chapters,
          })),
        };
      }
      return {
        ...proj,
        acts: proj.acts.map((a) => ({
          ...a,
          chapters: a.chapters.map((c) => ({
            ...c,
            scenes:
              c.scenes.length > 1
                ? c.scenes.filter((s) => s.id !== id)
                : c.scenes,
          })),
        })),
      };
    });
  };

  // Tạo dự án sách mới
  const handleCreateProject = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newBookTitle.trim()) return;
    const pid = Date.now();
    const actId = pid + 1;
    const chapId = pid + 2;
    const sceneId = pid + 3;
    const charId = pid + 4;
    const locId = pid + 5;

    const newProj: Project = {
      id: pid,
      title: newBookTitle.trim(),
      author: newBookAuthor.trim() || 'Tác giả',
      genre: 'Tiểu thuyết Văn học',
      synopsis: '',
      targetWords: 60000,
      updatedAt: new Date().toISOString(),
      characters: [
        {
          id: charId,
          projectId: pid,
          name: 'Nhân vật dẫn chuyện',
          role: 'Nhân vật chính',
          description: 'Nhân vật trung tâm của tác phẩm.',
        },
      ],
      locations: [
        {
          id: locId,
          projectId: pid,
          name: 'Bối cảnh chính',
          description: 'Không gian mở đầu của tiểu thuyết.',
        },
      ],
      acts: [
        {
          id: actId,
          projectId: pid,
          title: 'Hồi I: Khởi Đầu',
          position: 1,
          chapters: [
            {
              id: chapId,
              actId,
              title: 'Chương 1: Chương Mở Đầu',
              targetWords: 3000,
              position: 1,
              scenes: [
                {
                  id: sceneId,
                  chapterId: chapId,
                  title: 'Cảnh 1: Hình Ảnh Đầu Tiên',
                  summary: 'Phân cảnh mở đầu của tiểu thuyết.',
                  content:
                    'Bắt đầu viết cảnh mở đầu của bạn tại đây. Mọi thay đổi sẽ tự động được lưu xuống bộ nhớ sau 750ms.',
                  sideNotes: 'Ghi chú ý tưởng và dàn ý cho cảnh mở đầu.',
                  status: 'Đang viết',
                  povCharacterId: charId,
                  locationId: locId,
                  characterIds: [charId],
                  targetWords: 1200,
                  wordCount: 20,
                  position: 1,
                  updatedAt: new Date().toISOString(),
                },
              ],
            },
          ],
        },
      ],
    };

    setProjects((prev) => [newProj, ...prev]);
    setActiveProjectId(pid);
    setSelectedSceneId(sceneId);
    setNewBookTitle('');
    setNewBookAuthor('');
    setShowNewBookForm(false);
  };

  // Quản lý Nhân vật & Bối cảnh
  const handleAddCharacter = (
    name: string,
    role: string,
    description: string,
    tagIds: number[] = []
  ) => {
    const newChar: Character = {
      id: Date.now(),
      projectId: activeProject.id,
      name,
      role,
      description,
      tagIds,
    };
    updateActiveProject((proj) => ({
      ...proj,
      characters: [...proj.characters, newChar],
    }));
    return newChar.id;
  };

  const handleUpdateCharacter = (updated: Character) => {
    updateActiveProject((proj) => ({
      ...proj,
      characters: proj.characters.map((c) => (c.id === updated.id ? updated : c)),
    }));
  };

  const handleDeleteCharacter = (id: number) => {
    updateActiveProject((proj) => ({
      ...proj,
      characters: proj.characters.filter((c) => c.id !== id),
      acts: proj.acts.map((a) => ({
        ...a,
        chapters: a.chapters.map((c) => ({
          ...c,
          scenes: c.scenes.map((s) => ({
            ...s,
            povCharacterId: s.povCharacterId === id ? null : s.povCharacterId,
            characterIds: s.characterIds.filter((cid) => cid !== id),
          })),
        })),
      })),
    }));
  };

  const handleAddLocation = (
    name: string,
    description: string,
    tagIds: number[] = []
  ) => {
    const newLoc: Location = {
      id: Date.now(),
      projectId: activeProject.id,
      name,
      description,
      tagIds,
    };
    updateActiveProject((proj) => ({
      ...proj,
      locations: [...proj.locations, newLoc],
    }));
    return newLoc.id;
  };

  const handleUpdateLocation = (updated: Location) => {
    updateActiveProject((proj) => ({
      ...proj,
      locations: proj.locations.map((l) => (l.id === updated.id ? updated : l)),
    }));
  };

  const handleDeleteLocation = (id: number) => {
    updateActiveProject((proj) => ({
      ...proj,
      locations: proj.locations.filter((l) => l.id !== id),
      acts: proj.acts.map((a) => ({
        ...a,
        chapters: a.chapters.map((c) => ({
          ...c,
          scenes: c.scenes.map((s) => ({
            ...s,
            locationId: s.locationId === id ? null : s.locationId,
          })),
        })),
      })),
    }));
  };

  // Quản lý Vật phẩm (Props), Sự kiện (Events) & Thẻ (Tags)
  const handleAddProp = (
    name: string,
    category: string,
    description: string,
    significance: string,
    tagIds: number[] = []
  ) => {
    const newProp: Prop = {
      id: Date.now(),
      bookId: activeProject.id,
      name,
      category,
      description,
      significance,
      tagIds,
    };
    updateActiveProject((proj) => ({
      ...proj,
      props: [...(proj.props || []), newProp],
    }));
  };

  const handleUpdateProp = (updated: Prop) => {
    updateActiveProject((proj) => ({
      ...proj,
      props: (proj.props || []).map((p) => (p.id === updated.id ? updated : p)),
    }));
  };

  const handleDeleteProp = (id: number) => {
    updateActiveProject((proj) => ({
      ...proj,
      props: (proj.props || []).filter((p) => p.id !== id),
      acts: proj.acts.map((a) => ({
        ...a,
        chapters: a.chapters.map((c) => ({
          ...c,
          scenes: c.scenes.map((s) => ({
            ...s,
            propIds: (s.propIds || []).filter((pid) => pid !== id),
          })),
        })),
      })),
    }));
  };

  const handleAddEvent = (
    title: string,
    timelineOrder: number,
    description: string,
    tagIds: number[] = []
  ) => {
    const newEv: WorldEvent = {
      id: Date.now(),
      bookId: activeProject.id,
      title,
      timelineOrder,
      description,
      tagIds,
    };
    updateActiveProject((proj) => ({
      ...proj,
      events: [...(proj.events || []), newEv],
    }));
  };

  const handleUpdateEvent = (updated: WorldEvent) => {
    updateActiveProject((proj) => ({
      ...proj,
      events: (proj.events || []).map((ev) =>
        ev.id === updated.id ? updated : ev
      ),
    }));
  };

  const handleDeleteEvent = (id: number) => {
    updateActiveProject((proj) => ({
      ...proj,
      events: (proj.events || []).filter((ev) => ev.id !== id),
      acts: proj.acts.map((a) => ({
        ...a,
        chapters: a.chapters.map((c) => ({
          ...c,
          scenes: c.scenes.map((s) => ({
            ...s,
            eventIds: (s.eventIds || []).filter((eid) => eid !== id),
          })),
        })),
      })),
    }));
  };

  const handleAddTag = (name: string) => {
    const newTag: Tag = {
      id: Date.now(),
      bookId: activeProject.id,
      name,
    };
    updateActiveProject((proj) => ({
      ...proj,
      tags: [...(proj.tags || []), newTag],
    }));
  };

  const handleDeleteTag = (tagId: number) => {
    updateActiveProject((proj) => ({
      ...proj,
      tags: (proj.tags || []).filter((t) => t.id !== tagId),
      characters: proj.characters.map((c) => ({
        ...c,
        tagIds: (c.tagIds || []).filter((id) => id !== tagId),
      })),
      locations: proj.locations.map((l) => ({
        ...l,
        tagIds: (l.tagIds || []).filter((id) => id !== tagId),
      })),
      props: (proj.props || []).map((p) => ({
        ...p,
        tagIds: (p.tagIds || []).filter((id) => id !== tagId),
      })),
      events: (proj.events || []).map((ev) => ({
        ...ev,
        tagIds: (ev.tagIds || []).filter((id) => id !== tagId),
      })),
    }));
  };

  const insertProseSnippet = (snippet: string) => {
    if (!activeContext) return;
    const el = textareaRef.current;
    if (!el) {
      updateActiveScene((sc) => ({ ...sc, content: sc.content + snippet }));
      return;
    }
    const start = el.selectionStart;
    const end = el.selectionEnd;
    const text = activeContext.scene.content;
    const next = text.slice(0, start) + snippet + text.slice(end);
    updateActiveScene((sc) => ({ ...sc, content: next }));
    setTimeout(() => {
      el.focus();
      el.setSelectionRange(start + snippet.length, start + snippet.length);
    }, 10);
  };

  // Tính toán số từ và tiến độ
  const activeSceneWords = activeContext?.scene.wordCount ?? 0;
  const activeSceneTarget = Math.max(1, activeContext?.scene.targetWords ?? 200);
  const scenePercent = Math.min(
    100,
    Math.round((activeSceneWords / activeSceneTarget) * 100)
  );

  const activeChapterWords = useMemo(() => {
    if (!activeContext) return 0;
    return activeContext.chapter.scenes.reduce((s, sc) => s + sc.wordCount, 0);
  }, [activeContext]);

  const activeChapterTarget = Math.max(
    1,
    activeContext?.chapter.targetWords ?? 400
  );
  const chapterPercent = Math.min(
    100,
    Math.round((activeChapterWords / activeChapterTarget) * 100)
  );

  const totalProjectWords = useMemo(
    () =>
      activeProject.acts.reduce(
        (sum, a) =>
          sum +
          a.chapters.reduce(
            (cSum, c) =>
              cSum + c.scenes.reduce((sSum, s) => sSum + s.wordCount, 0),
            0
          ),
        0
      ),
    [activeProject]
  );

  const renameKindLabel =
    renamingNode?.kind === 'act'
      ? 'Hồi'
      : renamingNode?.kind === 'chapter'
      ? 'Chương'
      : 'Cảnh';

  return (
    <div className="min-h-screen flex flex-col bg-[#F8F7F4] text-[#1C1B18]">
      {/* Thanh Điều Hướng Trên Cùng: Tuân thủ chuẩn 3 vùng (Thương hiệu — 5 Liên kết — 2 Nút hành động) */}
      <header className="h-14 px-6 border-b border-[#E6E4DD] bg-[#F8F7F4] flex items-center justify-between shrink-0">
        {/* Vùng 1: Tên ứng dụng */}
        <a
          href="#studio"
          onClick={(e) => {
            e.preventDefault();
            setActiveTab('studio');
          }}
          className="font-serif-display text-2xl font-bold tracking-tight text-[#1C1B18] whitespace-nowrap"
        >
          GoNovelist
        </a>

        {/* Vùng 2: 5 liên kết điều hướng Tiếng Việt */}
        <nav className="hidden md:flex items-center gap-6 text-sm font-medium text-[#68655E]">
          <button
            onClick={() => setActiveTab('studio')}
            className={`py-1 transition-colors whitespace-nowrap ${
              activeTab === 'studio'
                ? 'text-[#1C1B18] underline underline-offset-8 decoration-[#1E3A5F] decoration-2 font-semibold'
                : 'hover:text-[#1C1B18]'
            }`}
          >
            Bàn Viết Bản Thảo
          </button>
          <button
            onClick={() => setActiveTab('world')}
            className={`py-1 transition-colors whitespace-nowrap ${
              activeTab === 'world'
                ? 'text-[#1C1B18] underline underline-offset-8 decoration-[#1E3A5F] decoration-2 font-semibold'
                : 'hover:text-[#1C1B18]'
            }`}
          >
            Nhân Vật &amp; Bối Cảnh
          </button>
          <button
            onClick={() => setActiveTab('export')}
            className={`py-1 transition-colors whitespace-nowrap ${
              activeTab === 'export'
                ? 'text-[#1C1B18] underline underline-offset-8 decoration-[#1E3A5F] decoration-2 font-semibold'
                : 'hover:text-[#1C1B18]'
            }`}
          >
            Kết Xuất Bản Thảo
          </button>
          <button
            onClick={() => setActiveTab('go-source')}
            className={`py-1 transition-colors whitespace-nowrap ${
              activeTab === 'go-source'
                ? 'text-[#1C1B18] underline underline-offset-8 decoration-[#1E3A5F] decoration-2 font-semibold'
                : 'hover:text-[#1C1B18]'
            }`}
          >
            Mã Nguồn Go + Fyne
          </button>
          <button
            onClick={() => setActiveTab('architecture')}
            className={`py-1 transition-colors whitespace-nowrap ${
              activeTab === 'architecture'
                ? 'text-[#1C1B18] underline underline-offset-8 decoration-[#1E3A5F] decoration-2 font-semibold'
                : 'hover:text-[#1C1B18]'
            }`}
          >
            Lược Đồ &amp; Lệnh Build
          </button>
        </nav>

        {/* Vùng 3: 2 Nút hành động chính */}
        <div className="flex items-center gap-2.5">
          <button
            onClick={() => {
              setActiveTab('studio');
              setDistractionFree((prev) => !prev);
            }}
            className="inline-flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-medium text-[#1C1B18] bg-white border border-[#DCD9D0] rounded-lg hover:bg-[#F1EFEA] transition-colors whitespace-nowrap"
          >
            {distractionFree ? (
              <>
                <Minimize2 className="w-3.5 h-3.5" />
                Thoát Tập Trung
              </>
            ) : (
              <>
                <Maximize2 className="w-3.5 h-3.5" />
                Chế Độ Tập Trung
              </>
            )}
          </button>
          <button
            onClick={() => setActiveTab('export')}
            className="inline-flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-semibold text-white bg-[#1E3A5F] rounded-lg hover:bg-[#162B47] transition-colors whitespace-nowrap"
          >
            <Download className="w-3.5 h-3.5" />
            Xuất Bản Thảo
          </button>
        </div>
      </header>

      {/* Thanh điều hướng trên màn hình nhỏ */}
      <div className="md:hidden flex items-center gap-2 overflow-x-auto px-4 py-2 border-b border-[#E6E4DD] bg-[#F1EFEA] text-xs">
        {(
          [
            ['studio', 'Bàn Viết'],
            ['world', 'Nhân Vật & Bối Cảnh'],
            ['export', 'Kết Xuất'],
            ['go-source', 'Mã Nguồn Go'],
            ['architecture', 'Lược Đồ & Build'],
          ] as const
        ).map(([id, label]) => (
          <button
            key={id}
            onClick={() => setActiveTab(id)}
            className={`px-2.5 py-1 rounded-md font-medium whitespace-nowrap ${
              activeTab === id
                ? 'bg-[#1E3A5F] text-white'
                : 'text-[#68655E] hover:text-[#1C1B18]'
            }`}
          >
            {label}
          </button>
        ))}
      </div>

      {/* Khu vực nội dung chính */}
      {activeTab === 'world' && (
        <main className="flex-1 overflow-y-auto">
          <WorldCastView
            project={activeProject}
            onAddCharacter={handleAddCharacter}
            onUpdateCharacter={handleUpdateCharacter}
            onDeleteCharacter={handleDeleteCharacter}
            onAddLocation={handleAddLocation}
            onUpdateLocation={handleUpdateLocation}
            onDeleteLocation={handleDeleteLocation}
            onAddProp={handleAddProp}
            onUpdateProp={handleUpdateProp}
            onDeleteProp={handleDeleteProp}
            onAddEvent={handleAddEvent}
            onUpdateEvent={handleUpdateEvent}
            onDeleteEvent={handleDeleteEvent}
            onAddTag={handleAddTag}
            onDeleteTag={handleDeleteTag}
            onSelectScene={(sceneId) => {
              setSelectedSceneId(sceneId);
              setActiveTab('studio');
            }}
          />
        </main>
      )}

      {activeTab === 'export' && (
        <main className="flex-1 overflow-y-auto">
          <ExportStudioView project={activeProject} />
        </main>
      )}

      {activeTab === 'go-source' && (
        <main className="flex-1 overflow-y-auto">
          <GoSourceExplorer mode="source" />
        </main>
      )}

      {activeTab === 'architecture' && (
        <main className="flex-1 overflow-y-auto">
          <GoSourceExplorer mode="build" />
        </main>
      )}

      {activeTab === 'studio' && (
        <main className="flex-1 grid grid-cols-1 lg:grid-cols-12 min-h-[calc(100vh-3.5rem)]">
          {/* 1. Cột Trái: Cây Phân Cấp Bản Thảo (Dự án -> Hồi -> Chương -> Cảnh) */}
          {!distractionFree && (
            <aside className="lg:col-span-3 border-b lg:border-b-0 lg:border-r border-[#E6E4DD] bg-[#F1EFEA]/70 flex flex-col">
              {/* Chọn tác phẩm & Tạo tác phẩm mới */}
              <div className="p-4 border-b border-[#E6E4DD] space-y-3">
                <div className="flex items-center justify-between gap-2">
                  <select
                    value={activeProject.id}
                    onChange={(e) => {
                      const nextId = Number(e.target.value);
                      setActiveProjectId(nextId);
                      const proj = projects.find((p) => p.id === nextId);
                      const first =
                        proj?.acts[0]?.chapters[0]?.scenes[0]?.id;
                      if (first) setSelectedSceneId(first);
                    }}
                    className="flex-1 px-2.5 py-1.5 text-xs font-semibold text-[#1C1B18] bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F] truncate"
                  >
                    {projects.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.title}
                      </option>
                    ))}
                  </select>
                  <button
                    onClick={() => setShowNewBookForm((v) => !v)}
                    className="px-2.5 py-1.5 text-xs font-medium text-[#1C1B18] bg-white border border-[#DCD9D0] rounded-lg hover:bg-[#EAE7DF] transition-colors whitespace-nowrap"
                  >
                    + Sách Mới
                  </button>
                </div>

                {showNewBookForm && (
                  <form
                    onSubmit={handleCreateProject}
                    className="p-3 bg-white border border-[#DCD9D0] rounded-lg space-y-2"
                  >
                    <input
                      type="text"
                      value={newBookTitle}
                      onChange={(e) => setNewBookTitle(e.target.value)}
                      placeholder="Tên tiểu thuyết..."
                      className="w-full px-2.5 py-1.5 text-xs border border-[#DCD9D0] rounded focus:outline-none focus:border-[#1E3A5F]"
                    />
                    <input
                      type="text"
                      value={newBookAuthor}
                      onChange={(e) => setNewBookAuthor(e.target.value)}
                      placeholder="Tên bút danh / tác giả..."
                      className="w-full px-2.5 py-1.5 text-xs border border-[#DCD9D0] rounded focus:outline-none focus:border-[#1E3A5F]"
                    />
                    <div className="flex items-center justify-end gap-1.5">
                      <button
                        type="button"
                        onClick={() => setShowNewBookForm(false)}
                        className="px-2.5 py-1 text-xs text-[#68655E]"
                      >
                        Hủy
                      </button>
                      <button
                        type="submit"
                        className="px-2.5 py-1 text-xs font-semibold text-white bg-[#1E3A5F] rounded"
                      >
                        Tạo Mới
                      </button>
                    </div>
                  </form>
                )}

                {/* Ô tìm kiếm & Nút thêm Hồi */}
                <div className="flex items-center gap-2">
                  <div className="relative flex-1">
                    <Search className="w-3.5 h-3.5 text-[#68655E] absolute left-2.5 top-1/2 -translate-y-1/2" />
                    <input
                      type="text"
                      value={treeSearch}
                      onChange={(e) => setTreeSearch(e.target.value)}
                      placeholder="Tìm hồi, chương, cảnh..."
                      className="w-full pl-8 pr-2.5 py-1.5 text-xs bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F]"
                    />
                  </div>
                  <button
                    onClick={handleAddAct}
                    className="inline-flex items-center gap-1 px-2.5 py-1.5 text-xs font-semibold text-white bg-[#1E3A5F] rounded-lg hover:bg-[#162B47] transition-colors whitespace-nowrap"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    Hồi
                  </button>
                </div>
              </div>

              {/* Thanh đổi tên trực tiếp */}
              {renamingNode && (
                <div className="p-3 bg-[#FFFBEB] border-b border-[#E6E4DD] space-y-2">
                  <div className="text-xs font-medium text-[#1C1B18]">
                    Đổi tên {renameKindLabel}:
                  </div>
                  <div className="flex items-center gap-1.5">
                    <input
                      type="text"
                      value={renamingNode.title}
                      onChange={(e) =>
                        setRenamingNode({
                          ...renamingNode,
                          title: e.target.value,
                        })
                      }
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') handleCommitRename();
                        if (e.key === 'Escape') setRenamingNode(null);
                      }}
                      autoFocus
                      className="flex-1 px-2.5 py-1 text-xs bg-white border border-[#DCD9D0] rounded focus:outline-none focus:border-[#1E3A5F]"
                    />
                    <button
                      onClick={handleCommitRename}
                      className="px-2.5 py-1 text-xs font-semibold text-white bg-[#1E3A5F] rounded"
                    >
                      Lưu
                    </button>
                    <button
                      onClick={() => setRenamingNode(null)}
                      className="px-2 py-1 text-xs text-[#68655E]"
                    >
                      Hủy
                    </button>
                  </div>
                </div>
              )}

              {/* Danh sách Cây Hồi -> Chương -> Cảnh */}
              <div className="flex-1 overflow-y-auto p-3 space-y-3">
                {activeProject.acts.map((act, actIdx) => {
                  const isActCollapsed = !!collapsedActs[act.id];
                  const actWords = act.chapters.reduce(
                    (s, c) =>
                      s + c.scenes.reduce((cs, sc) => cs + sc.wordCount, 0),
                    0
                  );

                  return (
                    <div
                      key={act.id}
                      className="border border-[#E6E4DD] bg-white rounded-lg overflow-hidden"
                    >
                      {/* Dòng Hồi */}
                      <div className="px-2.5 py-2 bg-[#EAE7DF]/60 flex items-center justify-between gap-1 group">
                        <button
                          onClick={() =>
                            setCollapsedActs((prev) => ({
                              ...prev,
                              [act.id]: !prev[act.id],
                            }))
                          }
                          className="flex items-center gap-1.5 text-left min-w-0 flex-1"
                        >
                          {isActCollapsed ? (
                            <ChevronRight className="w-3.5 h-3.5 text-[#68655E] shrink-0" />
                          ) : (
                            <ChevronDown className="w-3.5 h-3.5 text-[#68655E] shrink-0" />
                          )}
                          <span className="text-xs font-bold text-[#1C1B18] truncate">
                            {act.title}
                          </span>
                        </button>

                        <div className="flex items-center gap-1 shrink-0">
                          <span className="text-[11px] font-mono text-[#68655E] tabular-nums mr-1">
                            {actWords} từ
                          </span>
                          <button
                            onClick={() => handleAddChapter(act.id)}
                            title="Thêm Chương vào Hồi này"
                            className="px-1.5 py-0.5 text-[11px] font-medium text-[#1E3A5F] hover:bg-white rounded whitespace-nowrap"
                          >
                            +Chương
                          </button>
                          <button
                            onClick={() =>
                              setRenamingNode({
                                kind: 'act',
                                id: act.id,
                                title: act.title,
                              })
                            }
                            title="Đổi tên Hồi"
                            className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                          >
                            <Edit3 className="w-3 h-3" />
                          </button>
                          <button
                            onClick={() => handleMoveAct(actIdx, -1)}
                            title="Chuyển Hồi lên"
                            className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                          >
                            <ArrowUp className="w-3 h-3" />
                          </button>
                          <button
                            onClick={() => handleMoveAct(actIdx, 1)}
                            title="Chuyển Hồi xuống"
                            className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                          >
                            <ArrowDown className="w-3 h-3" />
                          </button>
                          {activeProject.acts.length > 1 && (
                            <button
                              onClick={() => handleDeleteNode('act', act.id)}
                              title="Xóa Hồi"
                              className="p-1 text-[#68655E] hover:text-red-700"
                            >
                              <Trash2 className="w-3 h-3" />
                            </button>
                          )}
                        </div>
                      </div>

                      {/* Danh sách Chương trong Hồi */}
                      {!isActCollapsed && (
                        <div className="divide-y divide-[#F1EFEA]">
                          {act.chapters.map((chapter, chIdx) => {
                            const isChapCollapsed =
                              !!collapsedChapters[chapter.id];
                            const chapWords = chapter.scenes.reduce(
                              (s, sc) => s + sc.wordCount,
                              0
                            );

                            const filteredScenes = treeSearch.trim()
                              ? chapter.scenes.filter(
                                  (sc) =>
                                    sc.title
                                      .toLowerCase()
                                      .includes(treeSearch.toLowerCase()) ||
                                    sc.content
                                      .toLowerCase()
                                      .includes(treeSearch.toLowerCase())
                                )
                              : chapter.scenes;

                            if (
                              treeSearch.trim() &&
                              filteredScenes.length === 0 &&
                              !chapter.title
                                .toLowerCase()
                                .includes(treeSearch.toLowerCase())
                            ) {
                              return null;
                            }

                            return (
                              <div key={chapter.id} className="bg-white">
                                {/* Dòng Chương */}
                                <div className="pl-4 pr-2.5 py-1.5 bg-[#F8F7F4]/80 flex items-center justify-between gap-1">
                                  <button
                                    onClick={() =>
                                      setCollapsedChapters((prev) => ({
                                        ...prev,
                                        [chapter.id]: !prev[chapter.id],
                                      }))
                                    }
                                    className="flex items-center gap-1.5 text-left min-w-0 flex-1"
                                  >
                                    {isChapCollapsed ? (
                                      <ChevronRight className="w-3 h-3 text-[#68655E] shrink-0" />
                                    ) : (
                                      <ChevronDown className="w-3 h-3 text-[#68655E] shrink-0" />
                                    )}
                                    <span className="text-xs font-semibold text-[#3A3832] truncate">
                                      {chapter.title}
                                    </span>
                                  </button>

                                  <div className="flex items-center gap-0.5 shrink-0">
                                    <span className="text-[11px] font-mono text-[#68655E] tabular-nums mr-1">
                                      {chapWords} từ
                                    </span>
                                    <button
                                      onClick={() => handleAddScene(chapter.id)}
                                      title="Thêm Cảnh vào Chương này"
                                      className="px-1.5 py-0.5 text-[11px] font-medium text-[#1E3A5F] hover:bg-white rounded whitespace-nowrap"
                                    >
                                      +Cảnh
                                    </button>
                                    <button
                                      onClick={() =>
                                        setRenamingNode({
                                          kind: 'chapter',
                                          id: chapter.id,
                                          title: chapter.title,
                                        })
                                      }
                                      title="Đổi tên Chương"
                                      className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                                    >
                                      <Edit3 className="w-3 h-3" />
                                    </button>
                                    <button
                                      onClick={() =>
                                        handleMoveChapter(act.id, chIdx, -1)
                                      }
                                      title="Chuyển Chương lên"
                                      className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                                    >
                                      <ArrowUp className="w-3 h-3" />
                                    </button>
                                    <button
                                      onClick={() =>
                                        handleMoveChapter(act.id, chIdx, 1)
                                      }
                                      title="Chuyển Chương xuống"
                                      className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                                    >
                                      <ArrowDown className="w-3 h-3" />
                                    </button>
                                    {act.chapters.length > 1 && (
                                      <button
                                        onClick={() =>
                                          handleDeleteNode(
                                            'chapter',
                                            chapter.id
                                          )
                                        }
                                        title="Xóa Chương"
                                        className="p-1 text-[#68655E] hover:text-red-700"
                                      >
                                        <Trash2 className="w-3 h-3" />
                                      </button>
                                    )}
                                  </div>
                                </div>

                                {/* Danh sách Cảnh trong Chương */}
                                {!isChapCollapsed && (
                                  <div className="py-1 space-y-0.5">
                                    {filteredScenes.map((scene, scIdx) => {
                                      const isSelected =
                                        scene.id === activeContext?.scene.id;
                                      return (
                                        <div
                                          key={scene.id}
                                          className={`pl-7 pr-2.5 py-1.5 flex items-center justify-between gap-2 transition-colors ${
                                            isSelected
                                              ? 'bg-[#1E3A5F]/10 border-l-2 border-[#1E3A5F]'
                                              : 'hover:bg-[#F1EFEA]/60'
                                          }`}
                                        >
                                          <button
                                            onClick={() =>
                                              setSelectedSceneId(scene.id)
                                            }
                                            className="text-left min-w-0 flex-1"
                                          >
                                            <div
                                              className={`text-xs truncate ${
                                                isSelected
                                                  ? 'font-semibold text-[#1E3A5F]'
                                                  : 'text-[#1C1B18]'
                                              }`}
                                            >
                                              {scene.title}
                                            </div>
                                            {/* Metadata văn bản thuần có dấu chấm ngăn cách */}
                                            <div className="text-[11px] text-[#68655E] font-mono tabular-nums">
                                              <span>
                                                {normalizeStatus(
                                                  scene.status
                                                )}
                                              </span>
                                              <span className="mx-1.5">·</span>
                                              <span>{scene.wordCount} từ</span>
                                            </div>
                                          </button>

                                          <div className="flex items-center gap-0.5 shrink-0">
                                            <button
                                              onClick={() =>
                                                setRenamingNode({
                                                  kind: 'scene',
                                                  id: scene.id,
                                                  title: scene.title,
                                                })
                                              }
                                              title="Đổi tên Cảnh"
                                              className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                                            >
                                              <Edit3 className="w-3 h-3" />
                                            </button>
                                            <button
                                              onClick={() =>
                                                handleMoveScene(
                                                  chapter.id,
                                                  scIdx,
                                                  -1
                                                )
                                              }
                                              title="Chuyển Cảnh lên"
                                              className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                                            >
                                              <ArrowUp className="w-3 h-3" />
                                            </button>
                                            <button
                                              onClick={() =>
                                                handleMoveScene(
                                                  chapter.id,
                                                  scIdx,
                                                  1
                                                )
                                              }
                                              title="Chuyển Cảnh xuống"
                                              className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                                            >
                                              <ArrowDown className="w-3 h-3" />
                                            </button>
                                            {chapter.scenes.length > 1 && (
                                              <button
                                                onClick={() =>
                                                  handleDeleteNode(
                                                    'scene',
                                                    scene.id
                                                  )
                                                }
                                                title="Xóa Cảnh"
                                                className="p-1 text-[#68655E] hover:text-red-700"
                                              >
                                                <Trash2 className="w-3 h-3" />
                                              </button>
                                            )}
                                          </div>
                                        </div>
                                      );
                                    })}
                                  </div>
                                )}
                              </div>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>

              {/* Chân trang tổng số từ toàn bản thảo */}
              <div className="p-3.5 border-t border-[#E6E4DD] bg-[#EAE7DF]/50 flex items-center justify-between text-xs font-mono tabular-nums text-[#3A3832]">
                <span>Tổng số từ bản thảo</span>
                <span className="font-semibold">
                  {totalProjectWords.toLocaleString()} /{' '}
                  {activeProject.targetWords.toLocaleString()} từ
                </span>
              </div>
            </aside>
          )}

          {/* 2. Khung Giữa: Trình Soạn Thảo Văn Xuôi & Chế Độ Tập Trung */}
          <section
            className={`${
              distractionFree
                ? 'lg:col-span-12 max-w-4xl mx-auto w-full'
                : 'lg:col-span-6'
            } flex flex-col bg-[#FAF9F5] border-b lg:border-b-0 lg:border-r border-[#E6E4DD]`}
          >
            {activeContext ? (
              <>
                {/* Thanh tiêu đề & Công cụ định dạng */}
                <div className="px-6 py-4 border-b border-[#E6E4DD] space-y-3">
                  <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-[#68655E]">
                    <div className="truncate">
                      <span>{activeContext.act.title}</span>
                      <span className="mx-1.5">/</span>
                      <span>{activeContext.chapter.title}</span>
                    </div>
                    <div className="font-mono tabular-nums text-xs">
                      {saveStatus === 'pending' ? (
                        <span className="text-amber-700">
                          Đang tự động lưu (trễ 750ms)...
                        </span>
                      ) : (
                        <span className="text-[#68655E]">
                          Đã lưu vào SQLite · {lastSavedClock}
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Tiêu đề Cảnh có thể sửa trực tiếp */}
                  <input
                    type="text"
                    value={activeContext.scene.title}
                    onChange={(e) =>
                      updateActiveScene((sc) => ({
                        ...sc,
                        title: e.target.value,
                      }))
                    }
                    className="w-full font-serif-display text-2xl md:text-3xl font-semibold text-[#1C1B18] bg-transparent border-none focus:outline-none"
                    placeholder="Nhập tiêu đề Cảnh..."
                  />

                  {/* Nút định dạng nhanh & Chuyển chế độ Soạn thảo / Xem bản in */}
                  <div className="flex flex-wrap items-center justify-between gap-3 pt-1">
                    <div className="flex items-center gap-1.5">
                      <button
                        onClick={() => insertProseSnippet('**in đậm**')}
                        className="px-2.5 py-1 text-xs font-semibold text-[#1C1B18] bg-white border border-[#DCD9D0] rounded hover:bg-[#F1EFEA] transition-colors"
                      >
                        Đậm
                      </button>
                      <button
                        onClick={() => insertProseSnippet('*in nghiêng*')}
                        className="px-2.5 py-1 text-xs italic font-serif-display text-[#1C1B18] bg-white border border-[#DCD9D0] rounded hover:bg-[#F1EFEA] transition-colors"
                      >
                        Nghiêng
                      </button>
                      <button
                        onClick={() =>
                          insertProseSnippet('\n\n## Tiêu đề phân đoạn\n\n')
                        }
                        className="px-2.5 py-1 text-xs font-medium text-[#1C1B18] bg-white border border-[#DCD9D0] rounded hover:bg-[#F1EFEA] transition-colors"
                      >
                        Tiêu đề phụ
                      </button>
                      <button
                        onClick={() =>
                          insertProseSnippet('\n\n> Đoạn trích dẫn\n\n')
                        }
                        className="px-2.5 py-1 text-xs font-medium text-[#1C1B18] bg-white border border-[#DCD9D0] rounded hover:bg-[#F1EFEA] transition-colors"
                      >
                        Trích dẫn
                      </button>
                      <button
                        onClick={() => insertProseSnippet('\n\n* * *\n\n')}
                        className="px-2.5 py-1 text-xs font-mono text-[#1C1B18] bg-white border border-[#DCD9D0] rounded hover:bg-[#F1EFEA] transition-colors whitespace-nowrap"
                      >
                        * * *
                      </button>
                    </div>

                    <div className="flex items-center gap-1 p-1 bg-[#EAE7DF] rounded-lg">
                      <button
                        onClick={() => setEditorMode('write')}
                        className={`px-2.5 py-1 text-xs font-medium rounded transition-colors whitespace-nowrap ${
                          editorMode === 'write'
                            ? 'bg-white text-[#1C1B18] shadow-xs'
                            : 'text-[#68655E] hover:text-[#1C1B18]'
                        }`}
                      >
                        Soạn Thảo
                      </button>
                      <button
                        onClick={() => setEditorMode('preview')}
                        className={`inline-flex items-center gap-1 px-2.5 py-1 text-xs font-medium rounded transition-colors whitespace-nowrap ${
                          editorMode === 'preview'
                            ? 'bg-white text-[#1C1B18] shadow-xs'
                            : 'text-[#68655E] hover:text-[#1C1B18]'
                        }`}
                      >
                        <Eye className="w-3 h-3" />
                        Xem Bản In
                      </button>
                    </div>
                  </div>
                </div>

                {/* Vùng viết văn xuôi trung tâm */}
                <div className="flex-1 flex flex-col p-6 md:px-10 md:py-8 overflow-y-auto">
                  {editorMode === 'write' ? (
                    <textarea
                      ref={textareaRef}
                      value={activeContext.scene.content}
                      onChange={(e) =>
                        updateActiveScene((sc) => ({
                          ...sc,
                          content: e.target.value,
                        }))
                      }
                      placeholder="Bắt đầu viết nội dung cảnh truyện bằng tiếng Việt tại đây..."
                      className="w-full flex-1 min-h-[420px] resize-none bg-transparent border-none focus:outline-none font-serif-display text-xl leading-[1.8] text-[#1C1B18] max-w-[68ch] mx-auto"
                    />
                  ) : (
                    <div className="max-w-[65ch] mx-auto w-full font-serif-display text-xl leading-[1.8] text-[#1C1B18] space-y-5">
                      {activeContext.scene.content
                        .split(/\n\s*\n/)
                        .filter(Boolean)
                        .map((para, idx) => {
                          const trimmed = para.trim();
                          if (trimmed === '* * *') {
                            return (
                              <div
                                key={idx}
                                className="text-center tracking-[0.4em] text-[#68655E] py-2"
                              >
                                * * *
                              </div>
                            );
                          }
                          if (trimmed.startsWith('## ')) {
                            return (
                              <h3
                                key={idx}
                                className="text-2xl font-semibold text-[#1C1B18] pt-4"
                              >
                                {trimmed.replace(/^##\s+/, '')}
                              </h3>
                            );
                          }
                          if (trimmed.startsWith('> ')) {
                            return (
                              <blockquote
                                key={idx}
                                className="pl-4 border-l-2 border-[#1E3A5F] italic text-[#3A3832]"
                              >
                                {trimmed.replace(/^>\s+/, '')}
                              </blockquote>
                            );
                          }
                          return (
                            <p key={idx} className={idx > 0 ? 'indent-6' : ''}>
                              {trimmed}
                            </p>
                          );
                        })}
                    </div>
                  )}
                </div>

                {/* Thanh Tiến Độ Số Từ Theo Thời Gian Thực (Cảnh & Chương) */}
                <div className="px-6 py-3.5 border-t border-[#E6E4DD] bg-[#F1EFEA]/60 grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div className="space-y-1.5">
                    <div className="flex items-center justify-between text-xs font-mono tabular-nums">
                      <span className="text-[#68655E]">Tiến độ từ của Cảnh</span>
                      <span className="font-semibold text-[#1C1B18]">
                        {activeSceneWords} / {activeContext.scene.targetWords}{' '}
                        từ ({scenePercent}%)
                      </span>
                    </div>
                    <div className="h-1.5 w-full bg-[#DCD9D0] rounded-full overflow-hidden">
                      <div
                        className="h-full bg-[#1E3A5F] transition-transform duration-150 origin-left"
                        style={{
                          transform: `scaleX(${Math.min(1, scenePercent / 100)})`,
                        }}
                      />
                    </div>
                  </div>

                  <div className="space-y-1.5">
                    <div className="flex items-center justify-between text-xs font-mono tabular-nums">
                      <span className="text-[#68655E]">
                        Tổng tiến độ của Chương
                      </span>
                      <span className="font-semibold text-[#1C1B18]">
                        {activeChapterWords} /{' '}
                        {activeContext.chapter.targetWords} từ ({chapterPercent}
                        %)
                      </span>
                    </div>
                    <div className="h-1.5 w-full bg-[#DCD9D0] rounded-full overflow-hidden">
                      <div
                        className="h-full bg-emerald-700 transition-transform duration-150 origin-left"
                        style={{
                          transform: `scaleX(${Math.min(1, chapterPercent / 100)})`,
                        }}
                      />
                    </div>
                  </div>
                </div>
              </>
            ) : (
              <div className="p-12 text-center text-sm text-[#68655E]">
                Hãy chọn hoặc tạo mới một Cảnh ở cây thư mục bên trái để bắt đầu viết.
              </div>
            )}
          </section>

          {/* 3. Cột Phải: Ngữ Cảnh, Nhân Vật, Bối Cảnh & Sổ Tay Ghi Chú Bên Lề */}
          {!distractionFree && activeContext && (
            <aside className="lg:col-span-3 bg-[#F1EFEA]/70 flex flex-col">
              {/* Chuyển đổi tab Ngữ cảnh / Ghi chú bên lề */}
              <div className="p-4 border-b border-[#E6E4DD]">
                <div className="grid grid-cols-2 gap-1 p-1 bg-[#EAE7DF] rounded-lg">
                  <button
                    onClick={() => setInspectorTab('context')}
                    className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap ${
                      inspectorTab === 'context'
                        ? 'bg-white text-[#1C1B18] shadow-xs'
                        : 'text-[#68655E] hover:text-[#1C1B18]'
                    }`}
                  >
                    Ngữ Cảnh Cảnh
                  </button>
                  <button
                    onClick={() => setInspectorTab('notes')}
                    className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap ${
                      inspectorTab === 'notes'
                        ? 'bg-white text-[#1C1B18] shadow-xs'
                        : 'text-[#68655E] hover:text-[#1C1B18]'
                    }`}
                  >
                    Ghi Chú Bên Lề
                  </button>
                </div>
              </div>

              {inspectorTab === 'context' ? (
                <div className="flex-1 overflow-y-auto p-5 space-y-6">
                  {/* Bộ chọn Trạng thái biên tập */}
                  <div>
                    <label className="block text-xs font-semibold text-[#1C1B18] mb-2">
                      Trạng Thái Cảnh
                    </label>
                    <div className="grid grid-cols-2 gap-1.5 p-1 bg-[#EAE7DF] rounded-lg">
                      {ALL_STATUSES.map((st: SceneStatus) => {
                        const active =
                          normalizeStatus(activeContext.scene.status) === st;
                        return (
                          <button
                            key={st}
                            onClick={() =>
                              updateActiveScene((sc) => ({ ...sc, status: st }))
                            }
                            className={`px-2.5 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap ${
                              active
                                ? 'bg-[#1E3A5F] text-white font-semibold'
                                : 'text-[#3A3832] hover:bg-white/60'
                            }`}
                          >
                            {st}
                          </button>
                        );
                      })}
                    </div>
                  </div>

                  {/* Chọn Nhân vật Góc nhìn (POV) */}
                  <div>
                    <label className="block text-xs font-semibold text-[#1C1B18] mb-1.5">
                      Góc Nhìn Trần Thuật (POV)
                    </label>
                    <select
                      value={activeContext.scene.povCharacterId ?? ''}
                      onChange={(e) => {
                        const val = e.target.value
                          ? Number(e.target.value)
                          : null;
                        updateActiveScene((sc) => ({
                          ...sc,
                          povCharacterId: val,
                        }));
                      }}
                      className="w-full px-3 py-2 text-xs bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F]"
                    >
                      <option value="">(Không chọn góc nhìn riêng)</option>
                      {activeProject.characters.map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name} · {c.role}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Chọn Bối cảnh / Địa điểm */}
                  <div>
                    <label className="block text-xs font-semibold text-[#1C1B18] mb-1.5">
                      Bối Cảnh / Địa Điểm Diễn Ra
                    </label>
                    <select
                      value={activeContext.scene.locationId ?? ''}
                      onChange={(e) => {
                        const val = e.target.value
                          ? Number(e.target.value)
                          : null;
                        updateActiveScene((sc) => ({
                          ...sc,
                          locationId: val,
                        }));
                      }}
                      className="w-full px-3 py-2 text-xs bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F]"
                    >
                      <option value="">(Chưa gắn bối cảnh)</option>
                      {activeProject.locations.map((l) => (
                        <option key={l.id} value={l.id}>
                          {l.name}
                        </option>
                      ))}
                    </select>

                    {/* Thêm nhanh bối cảnh */}
                    <div className="mt-2 flex items-center gap-1.5">
                      <input
                        type="text"
                        value={quickLocName}
                        onChange={(e) => setQuickLocName(e.target.value)}
                        placeholder="Tên bối cảnh mới..."
                        className="flex-1 px-2.5 py-1 text-xs bg-white border border-[#DCD9D0] rounded focus:outline-none focus:border-[#1E3A5F]"
                      />
                      <button
                        type="button"
                        onClick={() => {
                          if (!quickLocName.trim()) return;
                          const newId = handleAddLocation(
                            quickLocName.trim(),
                            ''
                          );
                          updateActiveScene((sc) => ({
                            ...sc,
                            locationId: newId,
                          }));
                          setQuickLocName('');
                        }}
                        className="px-2.5 py-1 text-xs font-medium text-[#1E3A5F] bg-white border border-[#DCD9D0] rounded hover:bg-[#EAE7DF] whitespace-nowrap"
                      >
                        + Thêm
                      </button>
                    </div>
                  </div>

                  {/* Chọn nhiều Nhân vật xuất hiện trong Cảnh */}
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <label className="text-xs font-semibold text-[#1C1B18]">
                        Nhân Vật Xuất Hiện Trong Cảnh
                      </label>
                      <span className="text-[11px] font-mono text-[#68655E] tabular-nums">
                        Đã chọn {activeContext.scene.characterIds.length}
                      </span>
                    </div>

                    <div className="border border-[#DCD9D0] bg-white rounded-lg divide-y divide-[#F1EFEA]">
                      {activeProject.characters.map((char) => {
                        const checked =
                          activeContext.scene.characterIds.includes(char.id);
                        return (
                          <button
                            key={char.id}
                            type="button"
                            onClick={() => {
                              updateActiveScene((sc) => {
                                const exists = sc.characterIds.includes(
                                  char.id
                                );
                                return {
                                  ...sc,
                                  characterIds: exists
                                    ? sc.characterIds.filter(
                                        (id) => id !== char.id
                                      )
                                    : [...sc.characterIds, char.id],
                                };
                              });
                            }}
                            className="w-full px-3 py-2 flex items-center justify-between text-left hover:bg-[#F8F7F4] transition-colors"
                          >
                            <div className="min-w-0">
                              <div className="text-xs font-medium text-[#1C1B18] truncate">
                                {char.name}
                              </div>
                              <div className="text-[11px] text-[#68655E]">
                                {char.role}
                              </div>
                            </div>
                            <div
                              className={`w-4 h-4 rounded flex items-center justify-center border ${
                                checked
                                  ? 'bg-[#1E3A5F] border-[#1E3A5F] text-white'
                                  : 'border-[#DCD9D0] bg-white'
                              }`}
                            >
                              {checked && <Check className="w-3 h-3" />}
                            </div>
                          </button>
                        );
                      })}
                    </div>

                    {/* Thêm nhanh nhân vật */}
                    <div className="mt-2 flex items-center gap-1.5">
                      <input
                        type="text"
                        value={quickCharName}
                        onChange={(e) => setQuickCharName(e.target.value)}
                        placeholder="Tên nhân vật mới..."
                        className="flex-1 px-2.5 py-1 text-xs bg-white border border-[#DCD9D0] rounded focus:outline-none focus:border-[#1E3A5F]"
                      />
                      <button
                        type="button"
                        onClick={() => {
                          if (!quickCharName.trim()) return;
                          const newId = handleAddCharacter(
                            quickCharName.trim(),
                            'Nhân vật phụ',
                            ''
                          );
                          updateActiveScene((sc) => ({
                            ...sc,
                            characterIds: [...sc.characterIds, newId],
                          }));
                          setQuickCharName('');
                        }}
                        className="px-2.5 py-1 text-xs font-medium text-[#1E3A5F] bg-white border border-[#DCD9D0] rounded hover:bg-[#EAE7DF] whitespace-nowrap"
                      >
                        + Thêm
                      </button>
                    </div>
                  </div>

                  {/* Chọn Vật phẩm trong Cảnh (scene_props) */}
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <label className="text-xs font-semibold text-[#1C1B18]">
                        🧭 Vật Phẩm Trong Cảnh (Props)
                      </label>
                      <span className="text-[11px] font-mono text-[#68655E] tabular-nums">
                        Đã chọn {(activeContext.scene.propIds || []).length}
                      </span>
                    </div>
                    <div className="border border-[#DCD9D0] bg-white rounded-lg divide-y divide-[#F1EFEA]">
                      {(activeProject.props || []).map((prop) => {
                        const checked = (
                          activeContext.scene.propIds || []
                        ).includes(prop.id);
                        return (
                          <button
                            key={prop.id}
                            type="button"
                            onClick={() => {
                              updateActiveScene((sc) => {
                                const current = sc.propIds || [];
                                const exists = current.includes(prop.id);
                                return {
                                  ...sc,
                                  propIds: exists
                                    ? current.filter((id) => id !== prop.id)
                                    : [...current, prop.id],
                                };
                              });
                            }}
                            className="w-full px-3 py-2 flex items-center justify-between text-left hover:bg-[#F8F7F4] transition-colors"
                          >
                            <div className="min-w-0">
                              <div className="text-xs font-medium text-[#1C1B18] truncate">
                                {prop.name}
                              </div>
                              <div className="text-[11px] text-[#68655E]">
                                {prop.category}
                              </div>
                            </div>
                            <div
                              className={`w-4 h-4 rounded flex items-center justify-center border ${
                                checked
                                  ? 'bg-[#1E3A5F] border-[#1E3A5F] text-white'
                                  : 'border-[#DCD9D0] bg-white'
                              }`}
                            >
                              {checked && <Check className="w-3 h-3" />}
                            </div>
                          </button>
                        );
                      })}
                    </div>
                  </div>

                  {/* Chọn Sự kiện trong Cảnh (scene_events) */}
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <label className="text-xs font-semibold text-[#1C1B18]">
                        ⏳ Sự Kiện Trong Cảnh (Events)
                      </label>
                      <span className="text-[11px] font-mono text-[#68655E] tabular-nums">
                        Đã chọn {(activeContext.scene.eventIds || []).length}
                      </span>
                    </div>
                    <div className="border border-[#DCD9D0] bg-white rounded-lg divide-y divide-[#F1EFEA]">
                      {(activeProject.events || []).map((ev) => {
                        const checked = (
                          activeContext.scene.eventIds || []
                        ).includes(ev.id);
                        return (
                          <button
                            key={ev.id}
                            type="button"
                            onClick={() => {
                              updateActiveScene((sc) => {
                                const current = sc.eventIds || [];
                                const exists = current.includes(ev.id);
                                return {
                                  ...sc,
                                  eventIds: exists
                                    ? current.filter((id) => id !== ev.id)
                                    : [...current, ev.id],
                                };
                              });
                            }}
                            className="w-full px-3 py-2 flex items-center justify-between text-left hover:bg-[#F8F7F4] transition-colors"
                          >
                            <div className="min-w-0">
                              <div className="text-xs font-medium text-[#1C1B18] truncate">
                                [Mốc #{ev.timelineOrder}] {ev.title}
                              </div>
                            </div>
                            <div
                              className={`w-4 h-4 rounded flex items-center justify-center border ${
                                checked
                                  ? 'bg-[#1E3A5F] border-[#1E3A5F] text-white'
                                  : 'border-[#DCD9D0] bg-white'
                              }`}
                            >
                              {checked && <Check className="w-3 h-3" />}
                            </div>
                          </button>
                        );
                      })}
                    </div>
                  </div>

                  {/* Thiết lập chỉ tiêu số từ */}
                  <div className="pt-4 border-t border-[#E6E4DD] grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-[11px] font-medium text-[#68655E] mb-1">
                        Mục tiêu từ (Cảnh)
                      </label>
                      <input
                        type="number"
                        min={50}
                        step={50}
                        value={activeContext.scene.targetWords}
                        onChange={(e) => {
                          const val = Math.max(50, Number(e.target.value) || 50);
                          updateActiveScene((sc) => ({
                            ...sc,
                            targetWords: val,
                          }));
                        }}
                        className="w-full px-2.5 py-1.5 text-xs font-mono tabular-nums bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F]"
                      />
                    </div>
                    <div>
                      <label className="block text-[11px] font-medium text-[#68655E] mb-1">
                        Mục tiêu từ (Chương)
                      </label>
                      <input
                        type="number"
                        min={100}
                        step={100}
                        value={activeContext.chapter.targetWords}
                        onChange={(e) => {
                          const val = Math.max(
                            100,
                            Number(e.target.value) || 100
                          );
                          updateChapterTarget(activeContext.chapter.id, val);
                        }}
                        className="w-full px-2.5 py-1.5 text-xs font-mono tabular-nums bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F]"
                      />
                    </div>
                  </div>
                </div>
              ) : (
                /* Tab Sổ Tay Ghi Chú Bên Lề */
                <div className="flex-1 flex flex-col p-5 space-y-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="text-xs font-semibold text-[#1C1B18]">
                        Sổ Tay Nháp &amp; Ghi Chú Liên Kết
                      </h3>
                      <p className="text-[11px] text-[#68655E]">
                        Gắn riêng với {activeContext.scene.title}
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={() =>
                        updateActiveScene((sc) => ({
                          ...sc,
                          sideNotes:
                            (sc.sideNotes ? sc.sideNotes + '\n\n' : '') +
                            `• Ghi chú (${new Date().toLocaleTimeString(
                              'vi-VN',
                              {
                                hour: '2-digit',
                                minute: '2-digit',
                              }
                            )}): `,
                        }))
                      }
                      className="px-2.5 py-1 text-xs font-medium text-[#1E3A5F] bg-white border border-[#DCD9D0] rounded hover:bg-[#EAE7DF] whitespace-nowrap"
                    >
                      + Dòng ghi chú
                    </button>
                  </div>

                  <textarea
                    value={activeContext.scene.sideNotes}
                    onChange={(e) =>
                      updateActiveScene((sc) => ({
                        ...sc,
                        sideNotes: e.target.value,
                      }))
                    }
                    placeholder="Ghi nhanh ý tưởng thoại, chi tiết phục tuyến, tư liệu lịch sử hoặc lưu ý chỉnh sửa cho cảnh này..."
                    className="w-full flex-1 min-h-[320px] p-3.5 text-xs leading-relaxed bg-white border border-[#DCD9D0] rounded-xl focus:outline-none focus:border-[#1E3A5F] resize-none"
                  />
                </div>
              )}
            </aside>
          )}
        </main>
      )}
    </div>
  );
}
