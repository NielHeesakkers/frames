<!-- web/src/lib/components/ShareDialog.svelte -->
<script lang="ts">
  import { api } from '$lib/api';
  export let folderId: number;
  export let folderPath: string;
  export let fileIds: number[] | null = null;
  export let onClose: () => void;

  let expiresDays = 30;
  let password = '';
  let allowDownload = true;
  let allowUpload = false;
  let busy = false;
  let error = '';
  let created: any = null;

  $: isFileShare = fileIds && fileIds.length > 0;

  async function create() {
    busy = true; error = '';
    try {
      created = await api.createShare({
        folder_id: folderId,
        expires_in_days: Number(expiresDays) || 0,
        password: password || undefined,
        allow_download: allowDownload,
        allow_upload: allowUpload,
        file_ids: isFileShare ? fileIds : undefined
      });
    } catch (e: any) {
      error = e.message ?? 'failed';
    } finally {
      busy = false;
    }
  }

  async function copyAndClose() {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(created.url);
      } else {
        // Fallback for non-HTTPS contexts: legacy execCommand path.
        const ta = document.createElement('textarea');
        ta.value = created.url;
        ta.style.position = 'fixed';
        ta.style.opacity = '0';
        document.body.appendChild(ta);
        ta.select();
        document.execCommand('copy');
        document.body.removeChild(ta);
      }
    } catch {
      // Even if copying fails, still close; user can select the URL manually.
    }
    onClose();
  }
</script>

<div class="backdrop" on:click={onClose}>
  <div class="dialog" on:click|stopPropagation>
    <h3>
      {#if isFileShare}
        Share {fileIds?.length} geselecteerde foto{(fileIds?.length ?? 0) === 1 ? '' : '’s'}
      {:else}
        Share "{folderPath || 'root'}"
      {/if}
    </h3>
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
        <button on:click={onClose}>Done</button>
        <button class="primary" on:click={copyAndClose}>Copy link</button>
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
