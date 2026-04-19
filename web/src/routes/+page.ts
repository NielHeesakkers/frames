import { redirect } from '@sveltejs/kit';
import { refreshMe } from '$lib/stores';

export const load = async () => {
  const u = await refreshMe();
  if (!u) throw redirect(307, '/login');
  throw redirect(307, '/browse');
};
