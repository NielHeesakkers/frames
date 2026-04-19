export function csrfToken(): string {
  const name = 'frames_csrf=';
  for (const part of document.cookie.split(';')) {
    const p = part.trim();
    if (p.startsWith(name)) return decodeURIComponent(p.slice(name.length));
  }
  return '';
}
