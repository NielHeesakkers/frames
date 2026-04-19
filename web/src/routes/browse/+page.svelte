<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { currentFolderPath, sortMode, density, selection } from '$lib/stores';
  import Grid from '$lib/components/Grid.svelte';
  import ContextMenu from '$lib/components/ContextMenu.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import UploadDialog from '$lib/components/UploadDialog.svelte';
  import MovePicker from '$lib/components/MovePicker.svelte';
  import ShareDialog from '$lib/components/ShareDialog.svelte';

  let folder: any = null;
  let folders: any[] = [];
  let files: any[] = [];
  let loading = true;

  let menu: { file: any; x: number; y: number } | null = null;
  let confirmDelete: any = null;
  let renaming: any = null;
  let moving: any = null;
  let uploading = false;
  let newFolderName = '';
  let showNewFolder = false;
  let sharing: any = null;

  let dragging = false;
  let droppedFiles: File[] = [];

  async function load() {
    loading = true;
    const sort = $sortMode === 'takenAt' ? 'taken' : $sortMode;
    const r = await api.folder($currentFolderPath, { sort, limit: 500 });
    folder = r.folder;
    folders = r.folders;
    files = r.files;
    loading = false;
  }

  onMount(() => { currentFolderPath.set(''); });
  $: $currentFolderPath, load();

  function onContext(e: CustomEvent<{ file: any; x: number; y: number }>) { menu = e.detail; }

  async function doDelete(f: any) { await api.deleteFile(f.id); load(); }
  async function doRename(f: any, name: string) { await api.renameFile(f.id, name); load(); }
  async function doMove(f: any, folderId: number) { await api.moveFile(f.id, folderId); load(); }
  async function doMkdir() {
    if (!newFolderName) return;
    const sub = $currentFolderPath ? `${$currentFolderPath}/${newFolderName}` : newFolderName;
    await api.mkdir(sub);
    showNewFolder = false;
    newFolderName = '';
    load();
  }

  function onDragOver(e: DragEvent) {
    e.preventDefault();
    dragging = true;
  }
  function onDragLeave(e: DragEvent) {
    // Ignore leaves that land on child nodes inside the container.
    const t = e.currentTarget as HTMLElement | null;
    const rel = e.relatedTarget as Node | null;
    if (t && rel && t.contains(rel)) return;
    dragging = false;
  }
  function onDrop(e: DragEvent) {
    e.preventDefault();
    dragging = false;
    const fl = e.dataTransfer?.files;
    if (fl && fl.length > 0) {
      droppedFiles = Array.from(fl);
      uploading = true;
    }
  }

  function contextItems(f: any) {
    return [
      { label: 'Open', onSelect: () => location.assign(`/file/${f.id}`) },
      { label: 'Download', onSelect: () => location.assign(`/api/original/${f.id}`) },
      { label: 'Rename…', onSelect: () => { const n = prompt('New name', f.name); if (n && n !== f.name) doRename(f, n); } },
      { label: 'Move…', onSelect: () => (moving = f) },
      { label: 'Share folder containing…', onSelect: () => (sharing = { id: folder.id, path: folder.path }) },
      { label: 'Delete', danger: true, onSelect: () => (confirmDelete = f) }
    ];
  }
</script>

<div class="browse-root" class:dragging
     on:dragover={onDragOver}
     on:dragleave={onDragLeave}
     on:drop={onDrop}>
  <div class="toolbar">
    <select bind:value={$sortMode}>
      <option value="takenAt">By capture date</option>
      <option value="name">By name</option>
      <option value="size">By size</option>
    </select>
    <select bind:value={$density}>
      <option value="small">S</option><option value="medium">M</option><option value="large">L</option>
    </select>
    <div class="spacer" />
    <button on:click={() => (showNewFolder = true)}>New folder</button>
    <button on:click={() => (sharing = { id: folder.id, path: folder.path })}>Share</button>
    <button class="primary" on:click={() => (uploading = true)}>+ Upload</button>
  </div>

  {#if loading}
    <div class="loading">Loading…</div>
  {:else}
    {#if folders.length > 0}
      <section>
        <h3>Subfolders</h3>
        <div class="folder-cards">
          {#each folders as sub}
            <a class="fcard" href={`/browse/${sub.path.split('/').map(encodeURIComponent).join('/')}`}
               on:click|preventDefault={() => currentFolderPath.set(sub.path)}>
              <div class="ico">📁</div>
              <div class="name">{sub.name}</div>
              <div class="cnt">{sub.items} items</div>
            </a>
          {/each}
        </div>
      </section>
    {/if}
    <Grid files={files} on:context={onContext} />
  {/if}
</div>

{#if menu}
  <ContextMenu x={menu.x} y={menu.y} items={contextItems(menu.file)} onClose={() => (menu = null)} />
{/if}

{#if confirmDelete}
  <ConfirmDialog
    title="Delete {confirmDelete.name}?"
    message="This removes the file from disk."
    confirmLabel="Delete" danger
    onConfirm={async () => { await doDelete(confirmDelete); confirmDelete = null; }}
    onCancel={() => (confirmDelete = null)}
  />
{/if}

{#if moving}
  <MovePicker
    onPick={async (id, _path) => { await doMove(moving, id); moving = null; }}
    onClose={() => (moving = null)}
  />
{/if}

{#if uploading}
  <UploadDialog path={$currentFolderPath} initialFiles={droppedFiles}
    onClose={() => { uploading = false; droppedFiles = []; }} onDone={load} />
{/if}

{#if showNewFolder}
  <ConfirmDialog
    title="New folder"
    message=""
    confirmLabel="Create"
    onConfirm={doMkdir}
    onCancel={() => { showNewFolder = false; newFolderName = ''; }}
  >
    <input slot="body" bind:value={newFolderName} placeholder="Folder name" />
  </ConfirmDialog>
{/if}

{#if sharing}
  <ShareDialog folderId={sharing.id} folderPath={sharing.path} onClose={() => (sharing = null)} />
{/if}

<style>
  .browse-root { flex: 1; display: flex; flex-direction: column; min-height: 0; }
  .browse-root.dragging { outline: 2px dashed var(--accent); outline-offset: -8px; }
  .toolbar { display: flex; gap: 8px; padding: 8px 16px; border-bottom: 1px solid var(--border); align-items: center; }
  .spacer { flex: 1; }
  .loading { padding: 20px; color: var(--fg-dim); }
  h3 { margin: 16px 16px 8px; color: var(--fg-dim); font-size: 12px; text-transform: uppercase; }
  .folder-cards { display: grid; grid-template-columns: repeat(auto-fill, 180px);
    gap: 8px; padding: 0 16px; }
  .fcard { background: var(--bg-2); border-radius: 6px; padding: 12px;
    text-align: center; text-decoration: none; color: var(--fg); border: 1px solid var(--border); }
  .fcard:hover { border-color: var(--accent); }
  .ico { font-size: 26px; }
  .cnt { color: var(--fg-dim); font-size: 11px; }
</style>
