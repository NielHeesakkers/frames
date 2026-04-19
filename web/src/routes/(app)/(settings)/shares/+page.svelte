<!-- web/src/routes/shares/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { refreshMe } from '$lib/stores';

  let shares: any[] = [];
  let loading = true;
  let error = '';

  async function load() {
    loading = true;
    try {
      shares = await api.myShares();
    } catch (e: any) { error = e.message; }
    loading = false;
  }

  async function revoke(id: number) { await api.revokeShare(id); load(); }
  async function del(id: number) { await api.deleteShare(id); load(); }

  onMount(async () => { await refreshMe(); load(); });
</script>

<div class="page">
  <h2>My shares</h2>
  {#if loading}<p>Loading…</p>
  {:else if error}<p class="err">{error}</p>
  {:else if shares.length === 0}<p>No shares yet.</p>
  {:else}
    <table>
      <thead><tr><th>Folder</th><th>URL</th><th>Status</th><th>Expires</th><th>Flags</th><th></th></tr></thead>
      <tbody>
        {#each shares as s}
          <tr>
            <td>{s.folder_path || 'root'}</td>
            <td><input readonly value={s.url} /></td>
            <td>{s.status}</td>
            <td>{s.expires_at ?? 'never'}</td>
            <td>
              {s.has_password ? '🔒 ' : ''}{s.allow_download ? '⬇' : ''}{s.allow_upload ? ' ⬆' : ''}
            </td>
            <td class="actions">
              {#if s.status === 'active'}
                <button on:click={() => revoke(s.id)}>Revoke</button>
              {/if}
              <button class="danger" on:click={() => del(s.id)}>Delete</button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
</div>

<style>
  .page { padding: 24px; overflow-y: auto; height: 100%; }
  table { width: 100%; border-collapse: collapse; margin-top: 12px; }
  th, td { padding: 8px 10px; border-bottom: 1px solid var(--border); font-size: 13px;
    text-align: left; vertical-align: middle; }
  th { color: var(--fg-dim); font-weight: 500; }
  input { width: 100%; }
  .actions { display: flex; gap: 6px; }
  button.danger { background: var(--danger); color: white; border-color: var(--danger); }
  .err { color: var(--danger); }
</style>
