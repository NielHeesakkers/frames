<!-- web/src/routes/browse/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { currentFolderPath, sortMode, density } from '$lib/stores';
  import Grid from '$lib/components/Grid.svelte';

  let folder: any = null;
  let folders: any[] = [];
  let files: any[] = [];
  let loading = true;

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
</script>

<div class="toolbar">
  <select bind:value={$sortMode}>
    <option value="takenAt">By capture date</option>
    <option value="name">By name</option>
    <option value="size">By size</option>
  </select>
  <select bind:value={$density}>
    <option value="small">S</option>
    <option value="medium">M</option>
    <option value="large">L</option>
  </select>
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
  <Grid files={files} />
{/if}

<style>
  .toolbar { display: flex; gap: 8px; padding: 8px 16px; border-bottom: 1px solid var(--border); }
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
