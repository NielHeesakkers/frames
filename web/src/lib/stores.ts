import { writable, get } from 'svelte/store';
import { api } from './api';

export type Me = { id: number; username: string; is_admin: boolean } | null;

export const me = writable<Me>(null);

export async function refreshMe(): Promise<Me> {
  try {
    const u = await api.me();
    me.set(u);
    return u;
  } catch {
    me.set(null);
    return null;
  }
}

export async function logout() {
  await api.logout();
  me.set(null);
}

export const currentFolderPath = writable<string>('');
export const selection = writable<Set<number>>(new Set());

/**
 * `persisted` wraps a writable so the value round-trips through localStorage.
 * The key namespace is `frames.<name>.v1` so future migrations can bump it.
 */
function persisted<T>(key: string, def: T): import('svelte/store').Writable<T> {
  const storageKey = `frames.${key}.v1`;
  let initial: T = def;
  if (typeof localStorage !== 'undefined') {
    try {
      const raw = localStorage.getItem(storageKey);
      if (raw != null) initial = JSON.parse(raw) as T;
    } catch {
      /* corrupt entry, fall back */
    }
  }
  const s = writable<T>(initial);
  s.subscribe((v) => {
    if (typeof localStorage === 'undefined') return;
    try { localStorage.setItem(storageKey, JSON.stringify(v)); } catch {}
  });
  return s;
}

export const sortMode = persisted<'takenAt' | 'name' | 'size' | 'rating'>('sortMode', 'takenAt');
export const density = persisted<'small' | 'medium' | 'large'>('density', 'medium');

/** Thumbnail shape in the grid. 'square' = uniform squares (crop via object-fit:cover),
 *  'original' = preserve each photo's aspect ratio in a justified-rows layout. */
export const thumbShape = persisted<'square' | 'original'>('thumbShape', 'square');

/**
 * Paths of folders the user has expanded in the sidebar tree. Persisted to
 * localStorage so the tree keeps its shape across reloads.
 */
const EXPANDED_KEY = 'frames.expandedFolders.v1';

function loadExpanded(): Set<string> {
  if (typeof localStorage === 'undefined') return new Set();
  try {
    const raw = localStorage.getItem(EXPANDED_KEY);
    if (!raw) return new Set();
    const arr = JSON.parse(raw);
    return new Set(Array.isArray(arr) ? arr : []);
  } catch {
    return new Set();
  }
}

export const expandedFolders = writable<Set<string>>(loadExpanded());

expandedFolders.subscribe((s) => {
  if (typeof localStorage === 'undefined') return;
  try {
    localStorage.setItem(EXPANDED_KEY, JSON.stringify(Array.from(s)));
  } catch {
    /* quota or private-browsing: ignore */
  }
});

export function setExpanded(path: string, expanded: boolean) {
  expandedFolders.update((s) => {
    if (expanded) s.add(path);
    else s.delete(path);
    return new Set(s);
  });
}

export function useMe() {
  return { me, value: () => get(me) };
}
