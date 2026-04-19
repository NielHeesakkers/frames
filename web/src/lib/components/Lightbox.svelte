<!-- web/src/lib/components/Lightbox.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount, onDestroy, tick } from 'svelte';
  import { api } from '$lib/api';
  import StarRating from './StarRating.svelte';

  export let file: any;
  export let neighbors: number[] = [];

  async function onRatingChange(n: number) {
    const prev = file.rating ?? 0;
    file = { ...file, rating: n };
    try {
      await api.setRating(file.id, n);
    } catch {
      file = { ...file, rating: prev };
    }
  }

  // Shortcut overlay.
  let showHelp = false;

  // Recompute index reactively so navigation + re-renders stay in sync.
  $: index = neighbors.indexOf(file.id);
  $: hasPrev = index > 0;
  $: hasNext = index >= 0 && index < neighbors.length - 1;

  function close() {
    const rel = (file?.relative_path ?? '') as string;
    const parent = rel.split('/').slice(0, -1).join('/');
    if (parent) {
      goto('/browse/' + parent.split('/').map(encodeURIComponent).join('/'));
    } else {
      goto('/browse');
    }
  }
  function prev() { if (hasPrev) goto(`/file/${neighbors[index - 1]}`); }
  function next() { if (hasNext) goto(`/file/${neighbors[index + 1]}`); }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (showHelp) showHelp = false;
      else close();
    }
    else if (e.key === 'ArrowLeft') prev();
    else if (e.key === 'ArrowRight') next();
    else if (e.key === '+' || e.key === '=') setZoom(zoom * 1.25);
    else if (e.key === '-' || e.key === '_') setZoom(zoom / 1.25);
    else if (e.key === '0') resetZoom();
    else if (e.key === '?') showHelp = !showHelp;
  }
  onMount(() => window.addEventListener('keydown', onKey));
  onDestroy(() => window.removeEventListener('keydown', onKey));

  // Touch swipe
  let touchStart = 0;
  function onTouchStart(e: TouchEvent) { touchStart = e.touches[0].clientX; }
  function onTouchEnd(e: TouchEvent) {
    if (zoom !== 1) return; // swipe conflicts with pan when zoomed
    const dx = e.changedTouches[0].clientX - touchStart;
    if (Math.abs(dx) > 60) dx > 0 ? prev() : next();
  }

  // Zoom + pan state.
  let zoom = 1;
  let panX = 0;
  let panY = 0;
  let panning = false;
  let panStartX = 0;
  let panStartY = 0;
  let panOrigX = 0;
  let panOrigY = 0;

  function setZoom(z: number) {
    zoom = Math.min(8, Math.max(1, z));
    if (zoom === 1) { panX = 0; panY = 0; }
  }
  function resetZoom() { zoom = 1; panX = 0; panY = 0; }

  function onWheel(e: WheelEvent) {
    if (!e.ctrlKey && !e.metaKey) {
      // Plain scroll — let the browser pass through. Users hold ⌘ or Ctrl to zoom.
      return;
    }
    e.preventDefault();
    const factor = Math.exp(-e.deltaY * 0.002);
    setZoom(zoom * factor);
  }

  function onMouseDown(e: MouseEvent) {
    if (zoom === 1) return;
    e.preventDefault();
    panning = true;
    panStartX = e.clientX; panStartY = e.clientY;
    panOrigX = panX; panOrigY = panY;
  }
  function onMouseMove(e: MouseEvent) {
    if (!panning) return;
    panX = panOrigX + (e.clientX - panStartX);
    panY = panOrigY + (e.clientY - panStartY);
  }
  function onMouseUp() { panning = false; }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
    return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  }

  // Reset zoom on every navigation.
  $: if (file) { zoom = 1; panX = 0; panY = 0; }

  // Scroll the filmstrip so the active thumb is visible.
  let stripEl: HTMLDivElement | null = null;
  $: if (stripEl && index >= 0) {
    tick().then(() => {
      const activeEl = stripEl?.querySelector('.strip-thumb.active') as HTMLElement | null;
      if (activeEl) activeEl.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'center' });
    });
  }
</script>

<svelte:window on:mousemove={onMouseMove} on:mouseup={onMouseUp} />

<div class="lightbox">
  <button class="close" on:click={close} title="Sluiten (Esc)">✕</button>

  <button class="nav left" on:click={prev} disabled={!hasPrev} title="Vorige (←)">‹</button>

  <div class="media"
       on:touchstart={onTouchStart} on:touchend={onTouchEnd}
       on:wheel|nonpassive={onWheel}>
    {#if file.kind === 'video'}
      <video src={`/api/original/${file.id}`} controls autoplay></video>
    {:else if file.kind === 'other'}
      <a class="download-fallback" href={`/api/original/${file.id}`}>Download {file.name}</a>
    {:else}
      <img src={`/api/preview/${file.id}?v=${file.mtime ?? 0}`} alt={file.name}
           style="transform: translate({panX}px, {panY}px) scale({zoom}); cursor: {zoom > 1 ? (panning ? 'grabbing' : 'grab') : 'zoom-in'}"
           on:mousedown={onMouseDown}
           on:dblclick={() => (zoom === 1 ? setZoom(2) : resetZoom())}
           draggable="false" />
    {/if}

    {#if zoom !== 1}
      <div class="zoom-badge">{Math.round(zoom * 100)}%
        <button on:click={resetZoom} title="Reset zoom (0)">reset</button>
      </div>
    {/if}
  </div>

  <button class="nav right" on:click={next} disabled={!hasNext} title="Volgende (→)">›</button>

  <aside class="info">
    <h3 title={file.name}>{file.name}</h3>
    {#if neighbors.length > 0 && index >= 0}
      <p class="position">{index + 1} van {neighbors.length}</p>
    {/if}

    <div class="rating-row">
      <span class="rating-label">Rating</span>
      <StarRating value={file.rating ?? 0} onChange={onRatingChange} size={20} />
    </div>

    <dl>
      {#if file.exif?.taken_at}
        <dt>Genomen</dt><dd>{file.exif.taken_at}</dd>
      {:else if file.taken_at}
        <dt>Genomen</dt><dd>{file.taken_at.replace('T', ' ')}</dd>
      {/if}

      {#if file.exif?.camera}
        <dt>Camera</dt><dd>{file.exif.camera}</dd>
      {:else if file.camera_model}
        <dt>Camera</dt><dd>{file.camera_make ?? ''} {file.camera_model}</dd>
      {/if}

      {#if file.exif?.lens}
        <dt>Lens</dt><dd>{file.exif.lens}</dd>
      {/if}
      {#if file.exif?.focal_length}
        <dt>Focale</dt><dd>{file.exif.focal_length}</dd>
      {/if}
      {#if file.exif?.aperture}
        <dt>Diafragma</dt><dd>{file.exif.aperture}</dd>
      {/if}
      {#if file.exif?.shutter_speed}
        <dt>Sluiter</dt><dd>{file.exif.shutter_speed}</dd>
      {/if}
      {#if file.exif?.iso}
        <dt>ISO</dt><dd>{file.exif.iso}</dd>
      {/if}

      {#if file.width}
        <dt>Afmetingen</dt><dd>{file.width} × {file.height} px</dd>
      {:else if file.exif?.width}
        <dt>Afmetingen</dt><dd>{file.exif.width} × {file.exif.height} px</dd>
      {/if}

      {#if file.size}
        <dt>Grootte</dt><dd>{formatSize(file.size)}</dd>
      {/if}

      <dt>Type</dt><dd>{file.mime_type ?? file.kind}</dd>

      {#if file.exif?.gps_lat && file.exif?.gps_lon}
        <dt>GPS</dt><dd>
          <a target="_blank" rel="noopener" href={`https://www.openstreetmap.org/?mlat=${encodeURIComponent(file.exif.gps_lat)}&mlon=${encodeURIComponent(file.exif.gps_lon)}#map=16`}>
            {file.exif.gps_lat}, {file.exif.gps_lon}
          </a>
        </dd>
      {/if}
      {#if file.exif?.software}
        <dt>Software</dt><dd>{file.exif.software}</dd>
      {/if}

      {#if file.relative_path}
        <dt>Pad</dt><dd class="path" title={file.relative_path}>{file.relative_path}</dd>
      {/if}
    </dl>

    <a class="dl" href={`/api/original/${file.id}`} download={file.name}>Download origineel</a>
  </aside>

  {#if neighbors.length > 1}
    <div class="filmstrip" bind:this={stripEl}>
      {#each neighbors as id, i (id)}
        <a class="strip-thumb" class:active={i === index} href={`/file/${id}`}>
          <img src={`/api/thumb/${id}`} alt="" loading="lazy" /><!-- filmstrip uses id only -->
        </a>
      {/each}
    </div>
  {/if}

  {#if showHelp}
    <div class="help-backdrop" on:click={() => (showHelp = false)}>
      <div class="help" on:click|stopPropagation>
        <h3>Sneltoetsen</h3>
        <dl>
          <dt>← →</dt><dd>Vorige / volgende foto</dd>
          <dt>Esc</dt><dd>Sluiten</dd>
          <dt>+ / −</dt><dd>Zoom in / uit</dd>
          <dt>0</dt><dd>Zoom reset</dd>
          <dt>1 … 5</dt><dd>Rating (met muis op thumb)</dd>
          <dt>?</dt><dd>Dit overzicht</dd>
        </dl>
        <div class="help-actions">
          <button class="primary" on:click={() => (showHelp = false)}>Sluiten</button>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .lightbox { position: fixed; inset: 0; background: #000;
    display: grid;
    grid-template-columns: 60px 1fr 60px 340px;
    grid-template-rows: 1fr 84px;
    z-index: 1000; }

  .close { position: absolute; top: 12px; right: 360px; z-index: 110;
    width: 36px; height: 36px; border-radius: 50%;
    background: rgba(255,255,255,0.08); border: 1px solid rgba(255,255,255,0.15);
    color: #fff; font-size: 16px; cursor: pointer;
    display: grid; place-items: center; }
  .close:hover { background: rgba(255,255,255,0.16); }

  .topbar { position: absolute; top: 12px; left: 80px; z-index: 110;
    display: flex; gap: 8px; align-items: center; }
  .iconbtn { background: rgba(255,255,255,0.08); border: 1px solid rgba(255,255,255,0.15);
    color: #fff; border-radius: 50%; width: 36px; height: 36px;
    display: grid; place-items: center; cursor: pointer; font-size: 14px; }
  .iconbtn:hover { background: rgba(255,255,255,0.16); }
  .help-backdrop { position: absolute; inset: 0; background: rgba(0,0,0,0.7);
    z-index: 120; display: grid; place-items: center; }
  .help { background: var(--bg-2); border: 1px solid var(--border);
    border-radius: 8px; padding: 24px 28px; min-width: 320px; max-width: 440px; }
  .help h3 { margin: 0 0 14px; }
  .help dl { display: grid; grid-template-columns: 90px 1fr; gap: 8px 14px; margin: 0; }
  .help dt { color: var(--fg-dim); font-family: ui-monospace, Menlo, monospace;
    font-size: 12px; }
  .help dd { margin: 0; font-size: 13px; }
  .help-actions { display: flex; justify-content: flex-end; margin-top: 16px; }

  .nav { background: transparent; color: #fff; border: none; font-size: 48px;
    cursor: pointer; grid-row: 1; display: grid; place-items: center;
    min-width: 0; min-height: 0; }
  .nav.left { grid-column: 1; }
  .nav.right { grid-column: 3; }
  .nav:disabled { opacity: 0.25; cursor: default; }
  .nav:not(:disabled):hover { color: var(--accent); }

  .media { grid-column: 2; grid-row: 1;
    display: flex; align-items: center; justify-content: center;
    padding: 20px;
    min-width: 0; min-height: 0; overflow: hidden; position: relative; }
  .media img, .media video { max-width: 100%; max-height: 100%;
    width: auto; height: auto; object-fit: contain;
    display: block; margin: auto;
    transition: transform 0.08s linear; user-select: none; }
  .download-fallback { color: var(--accent); font-size: 18px; }

  .zoom-badge { position: absolute; top: 12px; left: 12px;
    background: rgba(0,0,0,0.6); color: #fff; padding: 5px 10px;
    border-radius: var(--radius); font-size: 12px;
    display: flex; align-items: center; gap: 8px; }
  .zoom-badge button { background: transparent; border: 1px solid rgba(255,255,255,0.3);
    color: #fff; padding: 1px 8px; border-radius: 3px; font-size: 11px; cursor: pointer; }

  .info { grid-column: 4; grid-row: 1;
    background: var(--bg-2); padding: 52px 22px 22px; overflow-y: auto;
    border-left: 1px solid var(--border); min-width: 0; }
  .info h3 { margin: 0 0 4px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .position { color: var(--fg-dim); font-size: 12px; margin: 0 0 14px; }
  .rating-row { display: flex; align-items: center; gap: 10px; margin: 0 0 16px;
    padding: 10px 0; border-top: 1px solid var(--border); border-bottom: 1px solid var(--border); }
  .rating-label { color: var(--fg-dim); font-size: 11px; text-transform: uppercase; letter-spacing: 0.3px; }

  dl { display: grid; grid-template-columns: 90px 1fr; gap: 6px 12px;
    color: var(--fg-dim); margin: 0 0 20px; font-size: 13px; }
  dt { text-transform: uppercase; font-size: 11px; letter-spacing: 0.3px; padding-top: 1px; }
  dd { margin: 0; color: var(--fg); word-break: break-word; }
  dd.path { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; }
  dd a { color: var(--accent); }

  .dl { display: inline-block; background: var(--accent); color: #fff;
    padding: 8px 14px; border-radius: var(--radius); text-decoration: none; font-size: 13px; }
  .dl:hover { opacity: 0.9; }

  .filmstrip { grid-column: 1 / 4; grid-row: 2;
    display: flex; gap: 4px; padding: 10px 16px;
    overflow-x: auto; overflow-y: hidden;
    background: rgba(0,0,0,0.7); border-top: 1px solid rgba(255,255,255,0.06);
    scrollbar-width: thin; }
  .strip-thumb { flex: 0 0 auto; width: 64px; height: 64px;
    border: 2px solid transparent; border-radius: 3px; overflow: hidden;
    opacity: 0.55; transition: opacity 0.1s, border-color 0.1s; }
  .strip-thumb img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .strip-thumb:hover { opacity: 0.85; }
  .strip-thumb.active { opacity: 1; border-color: var(--accent); }

  @media (max-width: 900px) {
    .lightbox { grid-template-columns: 1fr; grid-template-rows: 1fr auto 70px; }
    .close { top: 12px; right: 12px; }
    .nav { position: absolute; top: 42%; transform: translateY(-50%); z-index: 110; }
    .nav.left { left: 4px; grid-column: unset; }
    .nav.right { right: 4px; grid-column: unset; }
    .media { grid-column: 1; }
    .info { grid-column: 1; grid-row: 2; padding: 16px 16px 22px;
      border-left: none; border-top: 1px solid var(--border); max-height: 40vh; }
    .filmstrip { grid-column: 1; grid-row: 3; padding: 6px 10px; }
    .strip-thumb { width: 56px; height: 56px; }
  }
</style>
