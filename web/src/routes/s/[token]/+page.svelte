<!-- web/src/routes/s/[token]/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';

  const token = $page.params.token as string;
  let meta: any = null;
  let folder: any = null;
  let files: any[] = [];
  let folders: any[] = [];
  let needsPassword = false;
  let password = '';
  let error = '';
  let currentSub = '';

  async function fetchJSON(path: string, opts: RequestInit = {}) {
    const r = await fetch(path, { credentials: 'include', ...opts });
    if (r.status === 401) { needsPassword = true; return null; }
    if (r.status === 410) { error = 'Share expired or revoked.'; return null; }
    if (!r.ok) { error = `Error ${r.status}`; return null; }
    return (await r.json()).data;
  }

  async function loadMeta() {
    meta = await fetchJSON(`/api/s/${token}`);
    if (meta) loadFolder('');
  }
  async function loadFolder(sub: string) {
    currentSub = sub;
    const d = await fetchJSON(`/api/s/${token}/folder${sub ? `?path=${encodeURIComponent(sub)}` : ''}`);
    if (d) { folder = d.folder; files = d.files; folders = d.folders; }
  }
  async function unlock() {
    const r = await fetch(`/api/s/${token}/unlock`, {
      method: 'POST', credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ password })
    });
    if (r.ok) { needsPassword = false; loadMeta(); }
    else error = 'Wrong password';
  }

  onMount(loadMeta);
</script>

{#if needsPassword}
  <div class="center">
    <form on:submit|preventDefault={unlock} class="card">
      <h2>Password required</h2>
      <input type="password" bind:value={password} autofocus />
      {#if error}<p class="err">{error}</p>{/if}
      <button class="primary">Unlock</button>
    </form>
  </div>
{:else if error}
  <div class="center"><p class="err">{error}</p></div>
{:else if folder}
  <div class="share">
    <header>
      <strong>{meta.folder.name || 'Shared'}</strong>
      <span class="sep">/</span>
      {#each currentSub.split('/').filter(Boolean) as p, i}
        <a href="#" on:click|preventDefault={() => loadFolder(meta.folder.path + '/' + currentSub.split('/').slice(0, i + 1).join('/'))}>{p}</a>
        <span class="sep">›</span>
      {/each}
      <div class="spacer" />
      {#if meta.allow_download}
        <a class="dl" href={`/api/s/${token}/zip`}>Download all (ZIP)</a>
      {/if}
    </header>

    {#if folders.length > 0}
      <h3>Subfolders</h3>
      <div class="fcards">
        {#each folders as sub}
          <a href="#" class="fcard" on:click|preventDefault={() => loadFolder(sub.path)}>
            📁 {sub.name}
          </a>
        {/each}
      </div>
    {/if}

    <div class="jgrid" style="--size: 180px">
      {#each files as f (f.id)}
        {@const a = f.width && f.height ? f.width / f.height : 1}
        <a class="jslot" href={`/api/s/${token}/original/${f.id}`} target="_blank"
           style="flex-grow: {a}; flex-basis: calc({a} * var(--size));
                  min-width: calc({a} * var(--size) * 0.55);
                  aspect-ratio: {a};">
          <img src={`/api/s/${token}/thumb/${f.id}`} alt={f.name} loading="lazy" />
        </a>
      {/each}
      <div class="jfiller" />
    </div>

    {#if meta.allow_upload}
      <form class="upload" method="post" enctype="multipart/form-data"
            action={`/api/s/${token}/upload`}>
        <input type="text" name="name" placeholder="Your name" />
        <input type="file" name="files" multiple />
        <button class="primary">Upload</button>
      </form>
    {/if}
  </div>
{/if}

<style>
  .center { min-height: 100vh; display: grid; place-items: center; }
  .card { background: var(--bg-2); padding: 24px; border-radius: 8px;
    border: 1px solid var(--border); display: flex; flex-direction: column; gap: 10px; }
  .share { padding: 16px; max-width: 1200px; margin: 0 auto; }
  header { display: flex; align-items: center; gap: 8px; padding: 8px 0 16px;
    border-bottom: 1px solid var(--border); }
  .spacer { flex: 1; }
  .dl { background: var(--accent); color: white; padding: 6px 12px; border-radius: 4px;
    text-decoration: none; }
  .fcards { display: grid; grid-template-columns: repeat(auto-fill, 180px); gap: 8px; padding: 8px 0; }
  .fcard { background: var(--bg-2); border-radius: 6px; padding: 12px; text-decoration: none;
    color: var(--fg); border: 1px solid var(--border); }
  /* Justified rows zodat foto's hun oorspronkelijke verhouding houden. Elke
     tegel heeft flex-grow + flex-basis gelijk aan zijn aspect ratio, zodat
     rijen de containerbreedte precies vullen zonder croppen. */
  .jgrid { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 12px; }
  .jslot { display: block; min-height: 60px; overflow: hidden; border-radius: 3px;
    background: var(--bg-2); position: relative; }
  .jslot img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .jfiller { flex: 99 1 0; height: 0; pointer-events: none; }
  .upload { margin-top: 20px; display: flex; gap: 8px; }
  .err { color: var(--danger); }
</style>
