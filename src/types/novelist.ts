export type SceneStatus = 'Ý tưởng' | 'Đang viết' | 'Hoàn thành' | 'Đã biên tập';

export const ALL_STATUSES: SceneStatus[] = [
  'Ý tưởng',
  'Đang viết',
  'Hoàn thành',
  'Đã biên tập',
];

export function normalizeStatus(raw: string): SceneStatus {
  switch (raw) {
    case 'Idea':
    case 'Ý tưởng':
      return 'Ý tưởng';
    case 'Drafting':
    case 'Đang viết':
      return 'Đang viết';
    case 'Completed':
    case 'Hoàn thành':
      return 'Hoàn thành';
    case 'Edited':
    case 'Đã biên tập':
      return 'Đã biên tập';
    default:
      return 'Ý tưởng';
  }
}

export interface Tag {
  id: number;
  bookId: number;
  name: string;
  color?: string;
  entityType?: 'character' | 'location' | 'prop' | 'event' | string;
}

export interface Character {
  id: number;
  projectId: number;
  name: string;
  role: string;
  description: string;
  tagIds?: number[];
}

export interface Location {
  id: number;
  projectId: number;
  name: string;
  description: string;
  tagIds?: number[];
}

export interface Prop {
  id: number;
  bookId: number;
  name: string;
  category: string;
  description: string;
  significance: string;
  tagIds?: number[];
}

export interface WorldEvent {
  id: number;
  bookId: number;
  title: string;
  timelineOrder: number;
  description: string;
  tagIds?: number[];
}

export interface Scene {
  id: number;
  chapterId: number;
  title: string;
  summary: string;
  content: string;
  sideNotes: string;
  status: SceneStatus;
  povCharacterId: number | null;
  locationId: number | null;
  targetWords: number;
  wordCount: number;
  position: number;
  updatedAt: string;
  characterIds: number[];
  propIds?: number[];
  eventIds?: number[];
}

export interface Chapter {
  id: number;
  actId: number;
  title: string;
  position: number;
  targetWords: number;
  scenes: Scene[];
}

export interface Act {
  id: number;
  projectId: number;
  title: string;
  position: number;
  chapters: Chapter[];
}

export interface Project {
  id: number;
  title: string;
  author: string;
  genre: string;
  synopsis: string;
  targetWords: number;
  updatedAt: string;
  tags?: Tag[];
  characters: Character[];
  locations: Location[];
  props?: Prop[];
  events?: WorldEvent[];
  acts: Act[];
}

export function countWords(text: string): number {
  const trimmed = text.trim();
  if (!trimmed) return 0;
  return trimmed.split(/\s+/).length;
}
