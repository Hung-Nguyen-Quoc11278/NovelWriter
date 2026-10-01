export type SceneStatus = 'Idea' | 'Drafting' | 'Completed' | 'Edited';

export const ALL_STATUSES: SceneStatus[] = ['Idea', 'Drafting', 'Completed', 'Edited'];

export interface Character {
  id: number;
  projectId: number;
  name: string;
  role: 'Protagonist' | 'Deuteragonist' | 'Antagonist' | 'Supporting';
  bio: string;
}

export interface Location {
  id: number;
  projectId: number;
  name: string;
  description: string;
}

export interface Scene {
  id: number;
  chapterId: number;
  title: string;
  content: string;
  sideNotes: string;
  status: SceneStatus;
  povCharacterId: number | null;
  locationId: number | null;
  characterIds: number[];
  targetWords: number;
  wordCount: number;
  sortOrder: number;
  updatedAt: string;
}

export interface Chapter {
  id: number;
  actId: number;
  title: string;
  targetWords: number;
  sortOrder: number;
  scenes: Scene[];
}

export interface Act {
  id: number;
  projectId: number;
  title: string;
  sortOrder: number;
  chapters: Chapter[];
}

export interface Project {
  id: number;
  title: string;
  author: string;
  genre: string;
  synopsis: string;
  targetWords: number;
  acts: Act[];
  characters: Character[];
  locations: Location[];
  updatedAt: string;
}

export function countWords(text: string): number {
  const trimmed = text.trim();
  if (!trimmed) return 0;
  return trimmed.split(/\s+/).filter(Boolean).length;
}
