import { csrfToken } from './csrf';

export class ApiError extends Error {
  status: number;
  constructor(status: number, msg: string) {
    super(msg);
    this.status = status;
  }
}

async function req<T>(method: string, path: string, body?: any, opts: RequestInit = {}): Promise<T> {
  const headers: Record<string, string> = { ...(opts.headers as any) };
  if (body && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
  }
  if (method !== 'GET' && method !== 'HEAD') {
    headers['X-CSRF-Token'] = csrfToken();
  }
  const init: RequestInit = {
    method, credentials: 'include',
    headers,
    body: body instanceof FormData ? body : body ? JSON.stringify(body) : undefined,
    ...opts
  };
  const res = await fetch(path, init);
  if (res.status === 204) return undefined as unknown as T;
  if (!res.ok) {
    let msg = res.statusText;
    try { const j = await res.json(); msg = j.error ?? msg; } catch {}
    throw new ApiError(res.status, msg);
  }
  const ct = res.headers.get('content-type') || '';
  if (ct.startsWith('application/json')) {
    const j = await res.json();
    return (j.data ?? j) as T;
  }
  return (await res.text()) as unknown as T;
}

export const api = {
  me: () => req<{ id: number; username: string; is_admin: boolean }>('GET', '/api/me'),
  login: (username: string, password: string) =>
    req<{ id: number; username: string; is_admin: boolean }>('POST', '/api/login', { username, password }),
  logout: () => req<void>('POST', '/api/logout'),

  folder: (path: string, params: { limit?: number; offset?: number; sort?: string } = {}) => {
    const q = new URLSearchParams(params as any).toString();
    const base = path ? `/api/folder/${encodeURIComponent(path).replace(/%2F/g, '/')}` : `/api/folder`;
    return req<{ folder: any; folders: any[]; files: any[]; has_more: boolean }>('GET', q ? `${base}?${q}` : base);
  },
  tree: (parent?: string) =>
    req<Array<{ id: number; path: string; name: string; has_child: boolean; items: number }>>(
      'GET', `/api/tree${parent ? `?parent=${encodeURIComponent(parent)}` : ''}`),
  file: (id: number) => req<any>('GET', `/api/file/${id}`),
  search: (q: Record<string, string>) =>
    req<{ files: any[]; has_more: boolean }>('GET', '/api/search?' + new URLSearchParams(q)),

  scan: (full = false) => req<void>('POST', `/api/scan${full ? '?full=1' : ''}`),

  mkdir: (path: string) => req<void>('POST', '/api/ops/mkdir', { path }),
  renameFile: (id: number, name: string) => req<void>('POST', '/api/ops/file/rename', { id, name }),
  moveFile: (id: number, new_folder_id: number) => req<void>('POST', '/api/ops/file/move', { id, new_folder_id }),
  deleteFile: (id: number) => req<void>('POST', '/api/ops/file/delete', { id }),
  renameFolder: (id: number, name: string) => req<void>('POST', '/api/ops/folder/rename', { id, name }),
  deleteFolder: (id: number) => req<void>('POST', '/api/ops/folder/delete', { id }),

  upload: (path: string, files: File[]) => {
    const fd = new FormData();
    fd.append('path', path);
    for (const f of files) fd.append('files', f);
    return req<{ ids: number[] }>('POST', '/api/upload', fd);
  },

  myShares: () => req<any[]>('GET', '/api/shares'),
  createShare: (body: any) => req<any>('POST', '/api/shares', body),
  revokeShare: (id: number) => req<void>('DELETE', `/api/shares/${id}/revoke`),
  deleteShare: (id: number) => req<void>('DELETE', `/api/shares/${id}`),

  sharedWithMe: () => req<any[]>('GET', '/api/shared_with_me'),
  addFolderShare: (folder_id: number, user_id: number) =>
    req<void>('POST', '/api/folder_shares', { folder_id, user_id }),
  removeFolderShare: (folder_id: number, user_id: number) =>
    req<void>('DELETE', '/api/folder_shares', { folder_id, user_id }),

  listUsers: () => req<any[]>('GET', '/api/admin/users'),
  createUser: (u: { username: string; password: string; is_admin: boolean }) =>
    req<{ id: number }>('POST', '/api/admin/users', u),
  deleteUser: (id: number) => req<void>('DELETE', `/api/admin/users/${id}`),
  scanStatus: () => req<any>('GET', '/api/admin/scan_status'),

  changePassword: (old: string, neo: string) =>
    req<void>('POST', '/api/account/password', { old, new: neo })
};
