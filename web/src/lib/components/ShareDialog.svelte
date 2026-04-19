<!-- web/src/lib/components/ShareDialog.svelte -->
<script lang="ts">
  import { api } from '$lib/api';
  export let folderId: number;
  export let folderPath: string;
  export let onClose: () => void;

  let expiresDays = 30;
  let password = '';
  let allowDownload = true;
  let allowUpload = false;
  let busy = false;
  let error = '';
  let created: any = null;

  async function create() {
    busy = true; error = '';
    try {
      created = await api.createShare({
        folder_id: folderId,
        expires_in_days: Number(expiresDays) || 0,
        password: password || undefined,
        allow_download: allowDownload,
        allow_upload: allowUpload
      });
    } catch (e: any) {
      error = e.message ?? 'failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="backdrop" on:click={onClose}>
  <div class="dialog" on:click|stopPropagation>
    <h3>Share "{folderPath || 'root'}"</h3>
    {#if !created}
      <label>Expires (days, 0 = never)
        <input type="number" min="0" bind:value={expiresDays} /></label>
      <label>Password (optional)
        <input type="text" bind:value={password} /></label>
      <label><input type="checkbox" bind:checked={allowDownload} /> Allow download (incl. ZIP)</label>
      <label><input type="checkbox" bind:checked={allowUpload} /> Allow upload from external</label>
      {#if error}<p class="err">{error}</p>{/if}
      <div class="actions">
        <button on:click={onClose}>Cancel</button>
        <button class="primary" on:click={create} disabled={busy}>Create</button>
      </div>
    {:else}
      <p>Share created:</p>
      <input readonly value={created.url} on:focus={(e) => e.currentTarget.select()} style="width:100%" />
      <div class="actions">
        <button on:click={() => navigator.clipboard.writeText(created.url)}>Copy link</button>
        <button class="primary" on:click={onClose}>Done</button>
      </div>
    {/if}
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.6);
    z-index: 150; display: grid; place-items: center; }
  .dialog { background: var(--bg-2); border-radius: 8px; padding: 24px;
    min-width: 420px; border: 1px solid var(--border);
    display: flex; flex-direction: column; gap: 10px; }
  label { display: flex; flex-direction: column; gap: 4px; color: var(--fg-dim); }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 12px; }
  .err { color: var(--danger); }
</style>
