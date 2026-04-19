<!-- (app)/settings/+page.svelte -->
<script lang="ts">
  import { api } from '$lib/api';
  import { me } from '$lib/stores';
  import { onMount } from 'svelte';

  let oldPw = '';
  let newPw = '';
  let msg = '';
  let error = '';

  let version = '';
  let changelog = '';

  async function change() {
    msg = ''; error = '';
    try {
      await api.changePassword(oldPw, newPw);
      oldPw = ''; newPw = '';
      msg = 'Wachtwoord gewijzigd.';
    } catch (e: any) {
      error = e.message ?? 'failed';
    }
  }

  async function loadVersion() {
    try {
      const v = await api.version();
      version = v.version;
      changelog = v.changelog;
    } catch {
      version = 'unknown';
    }
  }

  onMount(loadVersion);
</script>

<div class="page">
  <h2>Settings</h2>
  {#if $me}<p>Ingelogd als <strong>{$me.username}</strong>{$me.is_admin ? ' (admin)' : ''}.</p>{/if}

  <h3>Verander wachtwoord</h3>
  <form on:submit|preventDefault={change} class="card">
    <label>Huidig wachtwoord<input type="password" bind:value={oldPw} /></label>
    <label>Nieuw wachtwoord (min 8 tekens)<input type="password" bind:value={newPw} minlength="8" /></label>
    <button class="primary">Wijzig</button>
    {#if msg}<p class="ok">{msg}</p>{/if}
    {#if error}<p class="err">{error}</p>{/if}
  </form>

  <h3>Versie</h3>
  <div class="card">
    <div class="version-head">
      <strong>Frames v{version}</strong>
    </div>
    {#if changelog}
      <details open>
        <summary>Version history</summary>
        <pre>{changelog}</pre>
      </details>
    {/if}
  </div>
</div>

<style>
  .page { padding: 24px; overflow-y: auto; height: 100%; max-width: 820px; }
  .card { display: flex; flex-direction: column; gap: 10px; margin-bottom: 22px;
    background: var(--bg-2); border: 1px solid var(--border); padding: 20px; border-radius: 8px; }
  label { display: flex; flex-direction: column; gap: 4px; color: var(--fg-dim); }
  .ok { color: #22c55e; }
  .err { color: var(--danger); }
  h2 { margin: 0 0 8px; }
  h3 { margin: 22px 0 8px; color: var(--fg-dim); font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; }
  .version-head { font-size: 16px; }
  details { margin-top: 6px; }
  summary { cursor: pointer; color: var(--fg-dim); font-size: 13px; }
  pre { background: var(--bg); padding: 14px; border-radius: 6px;
    font-size: 12px; line-height: 1.5; white-space: pre-wrap; word-break: break-word;
    max-height: 400px; overflow-y: auto; }
</style>
