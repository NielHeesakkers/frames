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
export const sortMode = writable<'takenAt' | 'name' | 'size'>('takenAt');
export const density = writable<'small' | 'medium' | 'large'>('medium');

export function useMe() {
  return { me, value: () => get(me) };
}
