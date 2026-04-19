<!-- web/src/routes/login/+page.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { me } from '$lib/stores';

  let username = '';
  let password = '';
  let error = '';
  let busy = false;

  async function submit() {
    error = '';
    busy = true;
    try {
      // Ensure CSRF cookie is seeded by a GET first.
      await fetch('/api/me', { credentials: 'include' });
      const u = await api.login(username, password);
      me.set(u);
      goto('/browse');
    } catch (e: any) {
      error = e.message || 'login failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="center">
  <form class="card" on:submit|preventDefault={submit}>
    <h1>Frames</h1>
    <label>
      Username
      <input bind:value={username} required autofocus />
    </label>
    <label>
      Password
      <input type="password" bind:value={password} required />
    </label>
    {#if error}<p class="err">{error}</p>{/if}
    <button class="primary" disabled={busy}>{busy ? '...' : 'Login'}</button>
  </form>
</div>

<style>
  .center { min-height: 100vh; display: grid; place-items: center; }
  .card { background: var(--bg-2); padding: 24px; border-radius: 8px;
    width: 320px; display: flex; flex-direction: column; gap: 12px;
    border: 1px solid var(--border); }
  label { display: flex; flex-direction: column; gap: 4px; color: var(--fg-dim); }
  h1 { margin: 0 0 4px; }
  .err { color: var(--danger); margin: 0; }
</style>
