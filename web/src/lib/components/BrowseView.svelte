<!-- $lib/components/BrowseView.svelte
     Both /browse and /browse/[...path] render this with an explicit `path`
     prop. The component writes the path into the shared `currentFolderPath`
     store so the folder-tree highlight stays in sync. -->
<script lang="ts">
  import { api } from '$lib/api';
  import { currentFolderPath, sortMode, density, thumbShape, selection } from '$lib/stores';
  import Grid from '$lib/components/Grid.svelte';
  import ContextMenu from '$lib/components/ContextMenu.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import UploadDialog from '$lib/components/UploadDialog.svelte';
  import MovePicker from '$lib/components/MovePicker.svelte';
  import ShareDialog from '$lib/components/ShareDialog.svelte';

  export let path: string = '';

  let folder: any = null;
  let folders: any[] = [];
  let files: any[] = [];
  let loading = true;

  let latestFiles: any[] = [];
  let subtreeLatest: any[] = [];

  let menu: { file: any; x: number; y: number } | null = null;
  let confirmDelete: any = null;
  let moving: any = null;
  let uploading = false;
  let newFolderName = '';
  let showNewFolder = false;
  let sharing: any = null;

  let dragging = false;
  let droppedFiles: File[] = [];

  $: atRoot = path === '';
  // Keep the folder tree in sync with whatever path we were rendered with.
  $: currentFolderPath.set(path);

  async function load() {
    loading = true;
    const sort = $sortMode === 'takenAt' ? 'taken' : $sortMode;  // backend expects taken|name|size|rating
    try {
      const r = await api.folder(path, { sort, limit: 50000 });
      folder = r.folder;
      folders = r.folders;
      files = r.files;
    } catch {
      folder = null;
      folders = [];
      files = [];
    }
    if (atRoot) {
      try {
        const l = await api.latest(10, 0);
        latestFiles = l.files;
      } catch {
        latestFiles = [];
      }
      subtreeLatest = [];
    } else {
      latestFiles = [];
      if (files.length === 0) {
        try {
          const l = await api.latest(10, 0, path);
          subtreeLatest = l.files;
        } catch {
          subtreeLatest = [];
        }
      } else {
        subtreeLatest = [];
      }
    }
    loading = false;
  }

  // Re-run load() whenever the path prop changes (including the initial render).
  $: path, load();

  function onContext(e: CustomEvent<{ file: any; x: number; y: number }>) { menu = e.detail; }

  async function doDelete(f: any) { await api.deleteFile(f.id); load(); }
  async function doRename(f: any, name: string) { await api.renameFile(f.id, name); load(); }
  async function doMove(f: any, folderId: number) { await api.moveFile(f.id, folderId); load(); }
  async function doMkdir() {
    if (!newFolderName) return;
    const sub = path ? `${path}/${newFolderName}` : newFolderName;
    await api.mkdir(sub);
    showNewFolder = false;
    newFolderName = '';
    load();
  }

  function onDragOver(e: DragEvent) { e.preventDefault(); dragging = true; }
  function onDragLeave(e: DragEvent) {
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
      <option value="takenAt">Op opnamedatum</option>
      <option value="rating">Op rating</option>
      <option value="name">Op naam</option>
      <option value="size">Op grootte</option>
    </select>
    <select bind:value={$density}>
      <option value="small">S</option><option value="medium">M</option><option value="large">L</option>
    </select>
    <select bind:value={$thumbShape} title="Thumb shape">
      <option value="square">Squares</option>
      <option value="original">Oorspronkelijke verhouding</option>
    </select>
    <div class="spacer" />
    {#if $selection.size > 0}
      <span class="sel-count">{$selection.size} geselecteerd</span>
      <button on:click={() => selection.set(new Set())}>Deselect</button>
      <button on:click={() => (sharing = { id: folder?.id ?? 0, path: folder?.path ?? '', fileIds: Array.from($selection) })}>Share selected</button>
    {/if}
    <button on:click={() => (showNewFolder = true)}>New folder</button>
    {#if folder}<button on:click={() => (sharing = { id: folder.id, path: folder.path })}>Share folder</button>{/if}
    <button class="primary" on:click={() => (uploading = true)}>+ Upload</button>
  </div>

  {#if loading}
    <div class="loading">Loading…</div>
  {:else}
    <div class="scroll">
      {#if atRoot && latestFiles.length > 0}
        <section>
          <h3>Laatste toegevoegde foto's</h3>
          <div class="latest-grid">
            {#each latestFiles as lf}
              <a class="latest-cell" href={`/file/${lf.id}`}>
                <img src={`/api/thumb/${lf.id}`} alt={lf.name} loading="lazy" />
              </a>
            {/each}
          </div>
        </section>
      {/if}

      {#if !atRoot && folders.length > 0}
        <section>
          <h3>Submappen</h3>
          <div class="folder-cards">
            {#each folders as sub}
              <a class="fcard" href={`/browse/${sub.path.split('/').map(encodeURIComponent).join('/')}`}>
                <div class="ico">📁</div>
                <div class="name">{sub.name}</div>
                <div class="cnt">{sub.items} items</div>
              </a>
            {/each}
          </div>
        </section>
      {/if}

      {#if !atRoot}
        <section class="files-section">
          <h3>Foto's{files.length > 0 ? ` (${files.length})` : ''}</h3>
          {#if files.length > 0}
            <Grid files={files} on:context={onContext} />
          {:else if subtreeLatest.length > 0}
            <p class="sub-caption">Laatste toegevoegd in deze map (submappen inbegrepen)</p>
            <div class="latest-grid">
              {#each subtreeLatest as lf}
                <a class="latest-cell" href={`/file/${lf.id}`}>
                  <img src={`/api/thumb/${lf.id}`} alt={lf.name} loading="lazy" />
                </a>
              {/each}
            </div>
          {:else}
            <div class="empty">Geen foto's in deze map. Kies een submap of upload nieuwe bestanden.</div>
          {/if}
        </section>
      {/if}
    </div>
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
  <UploadDialog path={path} initialFiles={droppedFiles}
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
  <ShareDialog folderId={sharing.id} folderPath={sharing.path}
               fileIds={sharing.fileIds ?? null}
               onClose={() => (sharing = null)} />
{/if}

<style>
  .browse-root { flex: 1; display: flex; flex-direction: column; min-height: 0; }
  .browse-root.dragging { outline: 2px dashed var(--accent); outline-offset: -8px; }
  .toolbar { display: flex; gap: 8px; padding: 8px 16px; border-bottom: 1px solid var(--border); align-items: center; }
  .spacer { flex: 1; }
  .loading { padding: 20px; color: var(--fg-dim); }
  .scroll { flex: 1; overflow-y: auto; }
  h3 { margin: 16px 16px 8px; color: var(--fg-dim); font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; }
  .folder-cards { display: grid; grid-template-columns: repeat(auto-fill, 180px);
    gap: 8px; padding: 0 16px 4px; }
  .fcard { background: var(--bg-2); border-radius: 6px; padding: 12px;
    text-align: center; text-decoration: none; color: var(--fg); border: 1px solid var(--border); }
  .fcard:hover { border-color: var(--accent); }
  .ico { font-size: 26px; }
  .cnt { color: var(--fg-dim); font-size: 11px; }
  .latest-grid { display: grid; grid-template-columns: repeat(auto-fill, 160px);
    gap: 4px; padding: 0 16px 8px; }
  .latest-cell { display: block; aspect-ratio: 1; overflow: hidden; border-radius: 4px;
    background: var(--bg-2); border: 1px solid var(--border); }
  .latest-cell img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .files-section { display: flex; flex-direction: column; min-height: 0; }
  .empty { padding: 16px; color: var(--fg-dim); font-style: italic; }
  .sub-caption { margin: 0 16px 6px; color: var(--fg-dim); font-size: 12px; font-style: italic; }
  .sel-count { color: var(--accent); font-size: 13px; margin-right: 4px; }
</style>
