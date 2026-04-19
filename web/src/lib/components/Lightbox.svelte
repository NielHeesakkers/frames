<!-- web/src/lib/components/Lightbox.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount, onDestroy } from 'svelte';

  export let file: any;
  export let neighbors: number[] = [];

  // Recompute index reactively so navigation + re-renders stay in sync.
  $: index = neighbors.indexOf(file.id);
  $: hasPrev = index > 0;
  $: hasNext = index >= 0 && index < neighbors.length - 1;

  function close() {
    if (window.history.length > 1) window.history.back();
    else goto('/browse');
  }
  function prev() { if (hasPrev) goto(`/file/${neighbors[index - 1]}`); }
  function next() { if (hasNext) goto(`/file/${neighbors[index + 1]}`); }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') close();
    else if (e.key === 'ArrowLeft') prev();
    else if (e.key === 'ArrowRight') next();
  }
  onMount(() => window.addEventListener('keydown', onKey));
  onDestroy(() => window.removeEventListener('keydown', onKey));

  let touchStart = 0;
  function onTouchStart(e: TouchEvent) { touchStart = e.touches[0].clientX; }
  function onTouchEnd(e: TouchEvent) {
    const dx = e.changedTouches[0].clientX - touchStart;
    if (Math.abs(dx) > 60) dx > 0 ? prev() : next();
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MB`;
    return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  }
</script>

<div class="lightbox">
  <button class="close" on:click={close} title="Sluiten (Esc)">✕</button>

  <button class="nav left" on:click={prev} disabled={!hasPrev} title="Vorige (←)">‹</button>

  <div class="media" on:touchstart={onTouchStart} on:touchend={onTouchEnd}>
    {#if file.kind === 'video'}
      <video src={`/api/original/${file.id}`} controls autoplay></video>
    {:else if file.kind === 'other'}
      <a class="download-fallback" href={`/api/original/${file.id}`}>Download {file.name}</a>
    {:else}
      <img src={`/api/preview/${file.id}`} alt={file.name} />
    {/if}
  </div>

  <button class="nav right" on:click={next} disabled={!hasNext} title="Volgende (→)">›</button>

  <aside class="info">
    <h3 title={file.name}>{file.name}</h3>
    {#if neighbors.length > 0 && index >= 0}
      <p class="position">{index + 1} van {neighbors.length}</p>
    {/if}

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
</div>

<style>
  .lightbox { position: fixed; inset: 0; background: #000;
    display: grid;
    grid-template-columns: 60px 1fr 60px 340px;
    grid-template-rows: 1fr;
    z-index: 1000; }

  .close { position: absolute; top: 12px; right: 360px; z-index: 110;
    width: 36px; height: 36px; border-radius: 50%;
    background: rgba(255,255,255,0.08); border: 1px solid rgba(255,255,255,0.15);
    color: #fff; font-size: 16px; cursor: pointer;
    display: grid; place-items: center; }
  .close:hover { background: rgba(255,255,255,0.16); }

  .nav { background: transparent; color: #fff; border: none; font-size: 48px;
    cursor: pointer; grid-row: 1; display: grid; place-items: center;
    min-width: 0; min-height: 0; }
  .nav.left { grid-column: 1; }
  .nav.right { grid-column: 3; }
  .nav:disabled { opacity: 0.25; cursor: default; }
  .nav:not(:disabled):hover { color: var(--accent); }

  .media { grid-column: 2; grid-row: 1;
    display: grid; place-items: center; padding: 20px;
    min-width: 0; min-height: 0; overflow: hidden; }
  .media img, .media video { max-width: 100%; max-height: 100%; object-fit: contain;
    display: block; }
  .download-fallback { color: var(--accent); font-size: 18px; }

  .info { grid-column: 4; grid-row: 1;
    background: var(--bg-2); padding: 52px 22px 22px; overflow-y: auto;
    border-left: 1px solid var(--border); min-width: 0; }
  .info h3 { margin: 0 0 4px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .position { color: var(--fg-dim); font-size: 12px; margin: 0 0 14px; }

  dl { display: grid; grid-template-columns: 90px 1fr; gap: 6px 12px;
    color: var(--fg-dim); margin: 0 0 20px; font-size: 13px; }
  dt { text-transform: uppercase; font-size: 11px; letter-spacing: 0.3px; padding-top: 1px; }
  dd { margin: 0; color: var(--fg); word-break: break-word; }
  dd.path { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; }
  dd a { color: var(--accent); }

  .dl { display: inline-block; background: var(--accent); color: #fff;
    padding: 8px 14px; border-radius: var(--radius); text-decoration: none; font-size: 13px; }
  .dl:hover { opacity: 0.9; }

  @media (max-width: 900px) {
    .lightbox { grid-template-columns: 1fr; grid-template-rows: 1fr auto; }
    .close { top: 12px; right: 12px; }
    .nav { position: absolute; top: 50%; transform: translateY(-50%); z-index: 110; }
    .nav.left { left: 4px; grid-column: unset; }
    .nav.right { right: 4px; grid-column: unset; }
    .media { grid-column: 1; }
    .info { grid-column: 1; grid-row: 2; padding: 16px 16px 22px;
      border-left: none; border-top: 1px solid var(--border); max-height: 40vh; }
  }
</style>
