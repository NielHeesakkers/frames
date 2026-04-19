<!-- web/src/routes/settings/+page.svelte -->
<script lang="ts">
  import { api } from '$lib/api';
  import { me } from '$lib/stores';
  import { onMount } from 'svelte';

  let oldPw = '';
  let newPw = '';
  let msg = '';
  let error = '';

  async function change() {
    msg = ''; error = '';
    try {
      await api.changePassword(oldPw, newPw);
      oldPw = ''; newPw = '';
      msg = 'Password changed';
    } catch (e: any) {
      error = e.message ?? 'failed';
    }
  }

  onMount(() => { /* me store is hydrated by layout */ });
</script>

<div class="page">
  <h2>Settings</h2>
  {#if $me}<p>Signed in as <strong>{$me.username}</strong>{$me.is_admin ? ' (admin)' : ''}.</p>{/if}

  <h3>Change password</h3>
  <form on:submit|preventDefault={change} class="card">
    <label>Current password<input type="password" bind:value={oldPw} /></label>
    <label>New password (min 8 chars)<input type="password" bind:value={newPw} minlength="8" /></label>
    <button class="primary">Change</button>
    {#if msg}<p class="ok">{msg}</p>{/if}
    {#if error}<p class="err">{error}</p>{/if}
  </form>
</div>

<style>
  .page { padding: 24px; }
  .card { display: flex; flex-direction: column; gap: 10px; max-width: 420px;
    background: var(--bg-2); border: 1px solid var(--border); padding: 20px; border-radius: 8px; }
  label { display: flex; flex-direction: column; gap: 4px; color: var(--fg-dim); }
  .ok { color: #22c55e; }
  .err { color: var(--danger); }
</style>
