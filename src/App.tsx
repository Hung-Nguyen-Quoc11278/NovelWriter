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
  Project,
  Scene,
  SceneStatus,
  countWords,
} from './types/novelist';
import { INITIAL_PROJECTS } from './data/initialNovelData';
import { WorldCastView } from './components/WorldCastView';
import { ExportStudioView } from './components/ExportStudioView';
import { GoSourceExplorer } from './components/GoSourceExplorer';

type NavTab = 'studio' | 'world' | 'export' | 'go-source' | 'architecture';

const STORAGE_KEY = 'gonovelist_sqlite_mirror_v1';

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
      // Fallback to initial seed
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

  // Selected scene ID
  const [selectedSceneId, setSelectedSceneId] = useState<number>(() => {
    const firstScene =
      activeProject?.acts[0]?.chapters[0]?.scenes[0]?.id || 1111;
    return firstScene;
  });

  // Tree expansion state
  const [collapsedActs, setCollapsedActs] = useState<Record<number, boolean>>(
    {}
  );
  const [collapsedChapters, setCollapsedChapters] = useState<
    Record<number, boolean>
  >({});
  const [treeSearch, setTreeSearch] = useState('');

  // Inline rename modal/input state
  const [renamingNode, setRenamingNode] = useState<{
    kind: 'act' | 'chapter' | 'scene';
    id: number;
    title: string;
  } | null>(null);

  // New book inline creator
  const [showNewBookForm, setShowNewBookForm] = useState(false);
  const [newBookTitle, setNewBookTitle] = useState('');
  const [newBookAuthor, setNewBookAuthor] = useState('');

  // Editor state & 750ms Debounced Auto-Save Engine
  const [editorMode, setEditorMode] = useState<'write' | 'preview'>('write');
  const [inspectorTab, setInspectorTab] = useState<'context' | 'notes'>(
    'context'
  );
  const [saveStatus, setSaveStatus] = useState<'saved' | 'pending'>('saved');
  const [lastSavedClock, setLastSavedClock] = useState<string>(() =>
    new Date().toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  );

  // Quick-add character/location inside scene inspector
  const [quickCharName, setQuickCharName] = useState('');
  const [quickLocName, setQuickLocName] = useState('');

  const debounceTimerRef = useRef<number | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);

  // Locate current Act, Chapter, and Scene
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
    // Fallback to first scene if available
    for (const act of activeProject.acts) {
      for (const chapter of act.chapters) {
        if (chapter.scenes.length > 0) {
          return { act, chapter, scene: chapter.scenes[0] };
        }
      }
    }
    return null;
  }, [activeProject, selectedSceneId]);

  // Debounced persistence to localStorage (simulating the 750ms SQLite transaction in ui_editor.go)
  useEffect(() => {
    setSaveStatus('pending');
    if (debounceTimerRef.current) {
      window.clearTimeout(debounceTimerRef.current);
    }
    debounceTimerRef.current = window.setTimeout(() => {
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(projects));
      } catch {
        // Ignore storage quota errors
      }
      setSaveStatus('saved');
      setLastSavedClock(
        new Date().toLocaleTimeString([], {
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

  // Helper to update the active project immutably
  const updateActiveProject = (updater: (proj: Project) => Project) => {
    setProjects((prev) =>
      prev.map((p) => (p.id === activeProject.id ? updater(p) : p))
    );
  };

  // Update the currently active scene
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

  // Update chapter target words
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

  // Hierarchy CRUD operations
  const handleAddAct = () => {
    const nextNum = activeProject.acts.length + 1;
    const newActId = Date.now();
    const newChapId = newActId + 1;
    const newSceneId = newActId + 2;
    const newAct: Act = {
      id: newActId,
      projectId: activeProject.id,
      title: `Act ${nextNum}: New Movement`,
      sortOrder: nextNum,
      chapters: [
        {
          id: newChapId,
          actId: newActId,
          title: `Chapter 1: Opening Sequence`,
          targetWords: 300,
          sortOrder: 1,
          scenes: [
            {
              id: newSceneId,
              chapterId: newChapId,
              title: 'Scene 1: Untold Passage',
              content: '',
              sideNotes: '',
              status: 'Idea',
              povCharacterId: activeProject.characters[0]?.id ?? null,
              locationId: activeProject.locations[0]?.id ?? null,
              characterIds: activeProject.characters[0]
                ? [activeProject.characters[0].id]
                : [],
              targetWords: 200,
              wordCount: 0,
              sortOrder: 1,
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
          title: `Chapter ${nextNum}: New Chapter`,
          targetWords: 300,
          sortOrder: nextNum,
          scenes: [
            {
              id: newSceneId,
              chapterId: newChapId,
              title: 'Scene 1: Draft Scene',
              content: '',
              sideNotes: '',
              status: 'Idea',
              povCharacterId: proj.characters[0]?.id ?? null,
              locationId: proj.locations[0]?.id ?? null,
              characterIds: [],
              targetWords: 200,
              wordCount: 0,
              sortOrder: 1,
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
            title: `Scene ${nextNum}: Untitled Scene`,
            content: '',
            sideNotes: '',
            status: 'Idea',
            povCharacterId: proj.characters[0]?.id ?? null,
            locationId: proj.locations[0]?.id ?? null,
            characterIds: [],
            targetWords: 200,
            wordCount: 0,
            sortOrder: nextNum,
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
        acts: copy.map((a, idx) => ({ ...a, sortOrder: idx + 1 })),
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
          chapters: copy.map((c, idx) => ({ ...c, sortOrder: idx + 1 })),
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
            scenes: copy.map((s, idx) => ({ ...s, sortOrder: idx + 1 })),
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

  // Create new project
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
      author: newBookAuthor.trim() || 'Author',
      genre: 'Literary Fiction',
      synopsis: '',
      targetWords: 60000,
      updatedAt: new Date().toISOString(),
      characters: [
        {
          id: charId,
          projectId: pid,
          name: 'Narrator',
          role: 'Protagonist',
          bio: 'Primary viewpoint character.',
        },
      ],
      locations: [
        {
          id: locId,
          projectId: pid,
          name: 'Primary Setting',
          description: 'Opening location of the manuscript.',
        },
      ],
      acts: [
        {
          id: actId,
          projectId: pid,
          title: 'Act I: Premise',
          sortOrder: 1,
          chapters: [
            {
              id: chapId,
              actId,
              title: 'Chapter 1: First Movement',
              targetWords: 300,
              sortOrder: 1,
              scenes: [
                {
                  id: sceneId,
                  chapterId: chapId,
                  title: 'Scene 1: Opening Image',
                  content:
                    'Begin drafting your opening scene here. Changes are automatically debounced and persisted to local storage.',
                  sideNotes: 'Initial scene notes and structural beats.',
                  status: 'Drafting',
                  povCharacterId: charId,
                  locationId: locId,
                  characterIds: [charId],
                  targetWords: 200,
                  wordCount: 16,
                  sortOrder: 1,
                  updatedAt: new Date().toISOString(),
                },
              ],
            },
          ],
        },
      ],
    };

    setProjects((prev) => [...prev, newProj]);
    setActiveProjectId(pid);
    setSelectedSceneId(sceneId);
    setNewBookTitle('');
    setNewBookAuthor('');
    setShowNewBookForm(false);
  };

  // Character & Location management
  const handleAddCharacter = (
    name: string,
    role: Character['role'],
    bio: string
  ) => {
    const newChar: Character = {
      id: Date.now(),
      projectId: activeProject.id,
      name,
      role,
      bio,
    };
    updateActiveProject((proj) => ({
      ...proj,
      characters: [...proj.characters, newChar],
    }));
    return newChar.id;
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

  const handleAddLocation = (name: string, description: string) => {
    const newLoc = {
      id: Date.now(),
      projectId: activeProject.id,
      name,
      description,
    };
    updateActiveProject((proj) => ({
      ...proj,
      locations: [...proj.locations, newLoc],
    }));
    return newLoc.id;
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

  // Insert formatting snippet into prose textarea
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

  // Word count calculations
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

  return (
    <div className="min-h-screen flex flex-col bg-[#F8F7F4] text-[#1C1B18]">
      {/* Top Bar Contract: Strictly 1 row, 3 zones (Brand Wordmark — 5 Nav Links — 2 Actions) */}
      <header className="h-14 px-6 border-b border-[#E6E4DD] bg-[#F8F7F4] flex items-center justify-between shrink-0">
        {/* Zone 1: Single text element wordmark */}
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

        {/* Zone 2: 5 clean text navigation links */}
        <nav className="hidden md:flex items-center gap-6 text-sm font-medium text-[#68655E]">
          <button
            onClick={() => setActiveTab('studio')}
            className={`py-1 transition-colors whitespace-nowrap ${
              activeTab === 'studio'
                ? 'text-[#1C1B18] underline underline-offset-8 decoration-[#1E3A5F] decoration-2 font-semibold'
                : 'hover:text-[#1C1B18]'
            }`}
          >
            Manuscript Studio
          </button>
          <button
            onClick={() => setActiveTab('world')}
            className={`py-1 transition-colors whitespace-nowrap ${
              activeTab === 'world'
                ? 'text-[#1C1B18] underline underline-offset-8 decoration-[#1E3A5F] decoration-2 font-semibold'
                : 'hover:text-[#1C1B18]'
            }`}
          >
            World &amp; Cast
          </button>
          <button
            onClick={() => setActiveTab('export')}
            className={`py-1 transition-colors whitespace-nowrap ${
              activeTab === 'export'
                ? 'text-[#1C1B18] underline underline-offset-8 decoration-[#1E3A5F] decoration-2 font-semibold'
                : 'hover:text-[#1C1B18]'
            }`}
          >
            Compile &amp; Export
          </button>
          <button
            onClick={() => setActiveTab('go-source')}
            className={`py-1 transition-colors whitespace-nowrap ${
              activeTab === 'go-source'
                ? 'text-[#1C1B18] underline underline-offset-8 decoration-[#1E3A5F] decoration-2 font-semibold'
                : 'hover:text-[#1C1B18]'
            }`}
          >
            Go Source Code
          </button>
          <button
            onClick={() => setActiveTab('architecture')}
            className={`py-1 transition-colors whitespace-nowrap ${
              activeTab === 'architecture'
                ? 'text-[#1C1B18] underline underline-offset-8 decoration-[#1E3A5F] decoration-2 font-semibold'
                : 'hover:text-[#1C1B18]'
            }`}
          >
            Schema &amp; Build
          </button>
        </nav>

        {/* Zone 3: 2 Primary Actions */}
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
                Exit Focus
              </>
            ) : (
              <>
                <Maximize2 className="w-3.5 h-3.5" />
                Distraction-Free
              </>
            )}
          </button>
          <button
            onClick={() => setActiveTab('export')}
            className="inline-flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-semibold text-white bg-[#1E3A5F] rounded-lg hover:bg-[#162B47] transition-colors whitespace-nowrap"
          >
            <Download className="w-3.5 h-3.5" />
            Export Manuscript
          </button>
        </div>
      </header>

      {/* Mobile Navigation Bar */}
      <div className="md:hidden flex items-center gap-2 overflow-x-auto px-4 py-2 border-b border-[#E6E4DD] bg-[#F1EFEA] text-xs">
        {(
          [
            ['studio', 'Studio'],
            ['world', 'World & Cast'],
            ['export', 'Export'],
            ['go-source', 'Go Source'],
            ['architecture', 'Schema & Build'],
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

      {/* Main Content Area */}
      {activeTab === 'world' && (
        <main className="flex-1 overflow-y-auto">
          <WorldCastView
            project={activeProject}
            onAddCharacter={handleAddCharacter}
            onDeleteCharacter={handleDeleteCharacter}
            onAddLocation={handleAddLocation}
            onDeleteLocation={handleDeleteLocation}
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
          <GoSourceExplorer mode="architecture" />
        </main>
      )}

      {activeTab === 'studio' && (
        <main className="flex-1 grid grid-cols-1 lg:grid-cols-12 min-h-[calc(100vh-3.5rem)]">
          {/* 1. Left Sidebar: Hierarchical Tree Structure (Project -> Act -> Chapter -> Scene) */}
          {!distractionFree && (
            <aside className="lg:col-span-3 border-b lg:border-b-0 lg:border-r border-[#E6E4DD] bg-[#F1EFEA]/70 flex flex-col">
              {/* Project Selector & New Book */}
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
                    + Book
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
                      placeholder="Novel Title..."
                      className="w-full px-2.5 py-1.5 text-xs border border-[#DCD9D0] rounded focus:outline-none focus:border-[#1E3A5F]"
                    />
                    <input
                      type="text"
                      value={newBookAuthor}
                      onChange={(e) => setNewBookAuthor(e.target.value)}
                      placeholder="Author Name..."
                      className="w-full px-2.5 py-1.5 text-xs border border-[#DCD9D0] rounded focus:outline-none focus:border-[#1E3A5F]"
                    />
                    <div className="flex items-center justify-end gap-1.5">
                      <button
                        type="button"
                        onClick={() => setShowNewBookForm(false)}
                        className="px-2.5 py-1 text-xs text-[#68655E]"
                      >
                        Cancel
                      </button>
                      <button
                        type="submit"
                        className="px-2.5 py-1 text-xs font-semibold text-white bg-[#1E3A5F] rounded"
                      >
                        Create
                      </button>
                    </div>
                  </form>
                )}

                {/* Tree Filter & Add Act Button */}
                <div className="flex items-center gap-2">
                  <div className="relative flex-1">
                    <Search className="w-3.5 h-3.5 text-[#68655E] absolute left-2.5 top-1/2 -translate-y-1/2" />
                    <input
                      type="text"
                      value={treeSearch}
                      onChange={(e) => setTreeSearch(e.target.value)}
                      placeholder="Filter acts, chapters, scenes..."
                      className="w-full pl-8 pr-2.5 py-1.5 text-xs bg-white border border-[#DCD9D0] rounded-lg focus:outline-none focus:border-[#1E3A5F]"
                    />
                  </div>
                  <button
                    onClick={handleAddAct}
                    className="inline-flex items-center gap-1 px-2.5 py-1.5 text-xs font-semibold text-white bg-[#1E3A5F] rounded-lg hover:bg-[#162B47] transition-colors whitespace-nowrap"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    Act
                  </button>
                </div>
              </div>

              {/* Inline Rename Bar (if active) */}
              {renamingNode && (
                <div className="p-3 bg-[#FFFBEB] border-b border-[#E6E4DD] space-y-2">
                  <div className="text-xs font-medium text-[#1C1B18]">
                    Rename {renamingNode.kind}:
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
                      Save
                    </button>
                    <button
                      onClick={() => setRenamingNode(null)}
                      className="px-2 py-1 text-xs text-[#68655E]"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              )}

              {/* Hierarchical Tree List */}
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
                      {/* Act Row */}
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
                            {actWords}w
                          </span>
                          <button
                            onClick={() => handleAddChapter(act.id)}
                            title="Add Chapter to this Act"
                            className="px-1.5 py-0.5 text-[11px] font-medium text-[#1E3A5F] hover:bg-white rounded whitespace-nowrap"
                          >
                            +Ch
                          </button>
                          <button
                            onClick={() =>
                              setRenamingNode({
                                kind: 'act',
                                id: act.id,
                                title: act.title,
                              })
                            }
                            title="Rename Act"
                            className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                          >
                            <Edit3 className="w-3 h-3" />
                          </button>
                          <button
                            onClick={() => handleMoveAct(actIdx, -1)}
                            title="Move Act Up"
                            className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                          >
                            <ArrowUp className="w-3 h-3" />
                          </button>
                          <button
                            onClick={() => handleMoveAct(actIdx, 1)}
                            title="Move Act Down"
                            className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                          >
                            <ArrowDown className="w-3 h-3" />
                          </button>
                          {activeProject.acts.length > 1 && (
                            <button
                              onClick={() => handleDeleteNode('act', act.id)}
                              title="Delete Act"
                              className="p-1 text-[#68655E] hover:text-red-700"
                            >
                              <Trash2 className="w-3 h-3" />
                            </button>
                          )}
                        </div>
                      </div>

                      {/* Chapters inside Act */}
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
                                {/* Chapter Header */}
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
                                      {chapWords}w
                                    </span>
                                    <button
                                      onClick={() => handleAddScene(chapter.id)}
                                      title="Add Scene to Chapter"
                                      className="px-1.5 py-0.5 text-[11px] font-medium text-[#1E3A5F] hover:bg-white rounded whitespace-nowrap"
                                    >
                                      +Sc
                                    </button>
                                    <button
                                      onClick={() =>
                                        setRenamingNode({
                                          kind: 'chapter',
                                          id: chapter.id,
                                          title: chapter.title,
                                        })
                                      }
                                      title="Rename Chapter"
                                      className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                                    >
                                      <Edit3 className="w-3 h-3" />
                                    </button>
                                    <button
                                      onClick={() =>
                                        handleMoveChapter(act.id, chIdx, -1)
                                      }
                                      title="Move Chapter Up"
                                      className="p-1 text-[#68655E] hover:text-[#1C1B18]"
                                    >
                                      <ArrowUp className="w-3 h-3" />
                                    </button>
                                    <button
                                      onClick={() =>
                                        handleMoveChapter(act.id, chIdx, 1)
                                      }
                                      title="Move Chapter Down"
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
                                        title="Delete Chapter"
                                        className="p-1 text-[#68655E] hover:text-red-700"
                                      >
                                        <Trash2 className="w-3 h-3" />
                                      </button>
                                    )}
                                  </div>
                                </div>

                                {/* Scenes List */}
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
                                            {/* Clean unboxed metadata with typographic separator */}
                                            <div className="text-[11px] text-[#68655E] font-mono tabular-nums">
                                              <span>{scene.status}</span>
                                              <span className="mx-1.5">·</span>
                                              <span>{scene.wordCount}w</span>
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
                                              title="Rename Scene"
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
                                              title="Move Scene Up"
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
                                              title="Move Scene Down"
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
                                                title="Delete Scene"
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

              {/* Manuscript Total Word Count Footer */}
              <div className="p-3.5 border-t border-[#E6E4DD] bg-[#EAE7DF]/50 flex items-center justify-between text-xs font-mono tabular-nums text-[#3A3832]">
                <span>Manuscript Total</span>
                <span className="font-semibold">
                  {totalProjectWords.toLocaleString()} /{' '}
                  {activeProject.targetWords.toLocaleString()} words
                </span>
              </div>
            </aside>
          )}

          {/* 2. Center Panel: Rich Text & Distraction-Free Scene Editor */}
          <section
            className={`${
              distractionFree
                ? 'lg:col-span-12 max-w-4xl mx-auto w-full'
                : 'lg:col-span-6'
            } flex flex-col bg-[#FAF9F5] border-b lg:border-b-0 lg:border-r border-[#E6E4DD]`}
          >
            {activeContext ? (
              <>
                {/* Editor Top Context & Formatting Toolbar */}
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
                          Auto-saving (750ms debounce)...
                        </span>
                      ) : (
                        <span className="text-[#68655E]">
                          Saved to SQLite store · {lastSavedClock}
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Editable Scene Title */}
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
                    placeholder="Scene Title..."
                  />

                  {/* Formatting Buttons & Write/Preview Segmented Switch */}
                  <div className="flex flex-wrap items-center justify-between gap-3 pt-1">
                    <div className="flex items-center gap-1.5">
                      <button
                        onClick={() => insertProseSnippet('**bold text**')}
                        className="px-2.5 py-1 text-xs font-semibold text-[#1C1B18] bg-white border border-[#DCD9D0] rounded hover:bg-[#F1EFEA] transition-colors"
                      >
                        B
                      </button>
                      <button
                        onClick={() => insertProseSnippet('*italic text*')}
                        className="px-2.5 py-1 text-xs italic font-serif-display text-[#1C1B18] bg-white border border-[#DCD9D0] rounded hover:bg-[#F1EFEA] transition-colors"
                      >
                        I
                      </button>
                      <button
                        onClick={() =>
                          insertProseSnippet('\n\n## Section Heading\n\n')
                        }
                        className="px-2.5 py-1 text-xs font-medium text-[#1C1B18] bg-white border border-[#DCD9D0] rounded hover:bg-[#F1EFEA] transition-colors"
                      >
                        H2
                      </button>
                      <button
                        onClick={() =>
                          insertProseSnippet('\n\n> Quoted passage\n\n')
                        }
                        className="px-2.5 py-1 text-xs font-medium text-[#1C1B18] bg-white border border-[#DCD9D0] rounded hover:bg-[#F1EFEA] transition-colors"
                      >
                        Quote
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
                        Write Prose
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
                        Typeset View
                      </button>
                    </div>
                  </div>
                </div>

                {/* Main Prose Canvas */}
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
                      placeholder="Begin writing your scene prose..."
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

                {/* Real-Time Word Count & Target Progress Footer (Scene & Chapter) */}
                <div className="px-6 py-3.5 border-t border-[#E6E4DD] bg-[#F1EFEA]/60 grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div className="space-y-1.5">
                    <div className="flex items-center justify-between text-xs font-mono tabular-nums">
                      <span className="text-[#68655E]">Scene Word Progress</span>
                      <span className="font-semibold text-[#1C1B18]">
                        {activeSceneWords} / {activeContext.scene.targetWords}w (
                        {scenePercent}%)
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
                        Chapter Total Progress
                      </span>
                      <span className="font-semibold text-[#1C1B18]">
                        {activeChapterWords} /{' '}
                        {activeContext.chapter.targetWords}w ({chapterPercent}%)
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
                Select or create a scene from the hierarchy tree to begin writing.
              </div>
            )}
          </section>

          {/* 3. Right Inspector: Context & Metadata Mapping + Side Notes */}
          {!distractionFree && activeContext && (
            <aside className="lg:col-span-3 bg-[#F1EFEA]/70 flex flex-col">
              {/* Segmented Inspector Switcher */}
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
                    Context &amp; Cast
                  </button>
                  <button
                    onClick={() => setInspectorTab('notes')}
                    className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors whitespace-nowrap ${
                      inspectorTab === 'notes'
                        ? 'bg-white text-[#1C1B18] shadow-xs'
                        : 'text-[#68655E] hover:text-[#1C1B18]'
                    }`}
                  >
                    Side Notes
                  </button>
                </div>
              </div>

              {inspectorTab === 'context' ? (
                <div className="flex-1 overflow-y-auto p-5 space-y-6">
                  {/* Editorial Status Selector */}
                  <div>
                    <label className="block text-xs font-semibold text-[#1C1B18] mb-2">
                      Editorial Status
                    </label>
                    <div className="grid grid-cols-2 gap-1.5 p-1 bg-[#EAE7DF] rounded-lg">
                      {ALL_STATUSES.map((st: SceneStatus) => {
                        const active = activeContext.scene.status === st;
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

                  {/* Point of View (POV) Character Selector */}
                  <div>
                    <label className="block text-xs font-semibold text-[#1C1B18] mb-1.5">
                      Point of View (POV) Character
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
                      <option value="">(No specific POV)</option>
                      {activeProject.characters.map((c) => (
                        <option key={c.id} value={c.id}>
                          {c.name} · {c.role}
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Location / Setting Selector */}
                  <div>
                    <label className="block text-xs font-semibold text-[#1C1B18] mb-1.5">
                      Scene Location / Setting
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
                      <option value="">(Unassigned Setting)</option>
                      {activeProject.locations.map((l) => (
                        <option key={l.id} value={l.id}>
                          {l.name}
                        </option>
                      ))}
                    </select>

                    {/* Quick inline location creator */}
                    <div className="mt-2 flex items-center gap-1.5">
                      <input
                        type="text"
                        value={quickLocName}
                        onChange={(e) => setQuickLocName(e.target.value)}
                        placeholder="New location name..."
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
                        + Add
                      </button>
                    </div>
                  </div>

                  {/* Multi-Select Characters Present in Scene */}
                  <div>
                    <div className="flex items-center justify-between mb-1.5">
                      <label className="text-xs font-semibold text-[#1C1B18]">
                        Characters Present in Scene
                      </label>
                      <span className="text-[11px] font-mono text-[#68655E] tabular-nums">
                        {activeContext.scene.characterIds.length} assigned
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

                    {/* Quick inline character creator */}
                    <div className="mt-2 flex items-center gap-1.5">
                      <input
                        type="text"
                        value={quickCharName}
                        onChange={(e) => setQuickCharName(e.target.value)}
                        placeholder="New character name..."
                        className="flex-1 px-2.5 py-1 text-xs bg-white border border-[#DCD9D0] rounded focus:outline-none focus:border-[#1E3A5F]"
                      />
                      <button
                        type="button"
                        onClick={() => {
                          if (!quickCharName.trim()) return;
                          const newId = handleAddCharacter(
                            quickCharName.trim(),
                            'Supporting',
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
                        + Cast
                      </button>
                    </div>
                  </div>

                  {/* Preset Word Count Targets */}
                  <div className="pt-4 border-t border-[#E6E4DD] grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-[11px] font-medium text-[#68655E] mb-1">
                        Scene Word Target
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
                        Chapter Word Target
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
                /* Side Notes Scratchpad Tab */
                <div className="flex-1 flex flex-col p-5 space-y-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <h3 className="text-xs font-semibold text-[#1C1B18]">
                        Scene Scratchpad &amp; Continuity Notes
                      </h3>
                      <p className="text-[11px] text-[#68655E]">
                        Attached to {activeContext.scene.title}
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={() =>
                        updateActiveScene((sc) => ({
                          ...sc,
                          sideNotes:
                            (sc.sideNotes ? sc.sideNotes + '\n\n' : '') +
                            `• Note (${new Date().toLocaleTimeString([], {
                              hour: '2-digit',
                              minute: '2-digit',
                            })}): `,
                        }))
                      }
                      className="px-2.5 py-1 text-xs font-medium text-[#1E3A5F] bg-white border border-[#DCD9D0] rounded hover:bg-[#EAE7DF] whitespace-nowrap"
                    >
                      + Bullet Note
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
                    placeholder="Jot down scene beats, research fragments, sensory cues, or continuity reminders..."
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
