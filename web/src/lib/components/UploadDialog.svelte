<!-- web/src/lib/components/UploadDialog.svelte -->
<script lang="ts">
  import { api } from '$lib/api';
  export let path = '';
  export let onClose: () => void;
  export let onDone: () => void;

  let files: FileList | null = null;
  let busy = false;
  let progress = 0;
  let error = '';

  async function upload() {
    if (!files || files.length === 0) return;
    busy = true; error = ''; progress = 0;
    try {
      await api.upload(path, Array.from(files));
      onDone();
      onClose();
    } catch (e: any) {
      error = e.message ?? 'upload failed';
    } finally {
      busy = false;
    }
  }
</script>

<div class="backdrop" on:click={onClose}>
  <div class="dialog" on:click|stopPropagation>
    <h3>Upload to {path || 'root'}</h3>
    <input type="file" multiple bind:files />
    {#if error}<p class="err">{error}</p>{/if}
    <div class="actions">
      <button on:click={onClose}>Cancel</button>
      <button class="primary" on:click={upload} disabled={busy}>
        {busy ? 'Uploading…' : 'Upload'}
      </button>
    </div>
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.6); z-index: 150;
    display: grid; place-items: center; }
  .dialog { background: var(--bg-2); border-radius: 8px; padding: 24px;
    min-width: 420px; border: 1px solid var(--border); }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px; }
  .err { color: var(--danger); }
</style>
