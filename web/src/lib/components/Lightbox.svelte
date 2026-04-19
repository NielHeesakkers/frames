<!-- web/src/lib/components/Lightbox.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount, onDestroy } from 'svelte';

  export let file: any;
  export let neighbors: number[] = [];

  let index = neighbors.indexOf(file.id);

  function close() { history.back(); }
  function prev() { if (index > 0) goto(`/file/${neighbors[--index]}`); }
  function next() { if (index >= 0 && index < neighbors.length - 1) goto(`/file/${neighbors[++index]}`); }

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') close();
    if (e.key === 'ArrowLeft') prev();
    if (e.key === 'ArrowRight') next();
  }
  onMount(() => window.addEventListener('keydown', onKey));
  onDestroy(() => window.removeEventListener('keydown', onKey));

  let touchStart = 0;
  function onTouchStart(e: TouchEvent) { touchStart = e.touches[0].clientX; }
  function onTouchEnd(e: TouchEvent) {
    const dx = e.changedTouches[0].clientX - touchStart;
    if (Math.abs(dx) > 60) dx > 0 ? prev() : next();
  }
</script>

<div class="lightbox" on:click={close}>
  <button class="nav left" on:click|stopPropagation={prev} disabled={index <= 0}>‹</button>
  <div class="media" on:click|stopPropagation on:touchstart={onTouchStart} on:touchend={onTouchEnd}>
    {#if file.kind === 'video'}
      <video src={`/api/original/${file.id}`} controls autoplay></video>
    {:else if file.kind === 'other'}
      <a href={`/api/original/${file.id}`}>Download {file.name}</a>
    {:else}
      <img src={`/api/preview/${file.id}`} alt={file.name} />
    {/if}
  </div>
  <button class="nav right" on:click|stopPropagation={next} disabled={index < 0 || index >= neighbors.length - 1}>›</button>

  <aside class="info" on:click|stopPropagation>
    <h3>{file.name}</h3>
    <dl>
      <dt>Size</dt><dd>{(file.size / 1024 / 1024).toFixed(2)} MB</dd>
      {#if file.taken_at}<dt>Taken</dt><dd>{file.taken_at}</dd>{/if}
      {#if file.camera_model}<dt>Camera</dt><dd>{file.camera_make ?? ''} {file.camera_model}</dd>{/if}
      {#if file.width}<dt>Dim</dt><dd>{file.width} × {file.height}</dd>{/if}
    </dl>
    <a class="dl" href={`/api/original/${file.id}`} download={file.name}>Download</a>
  </aside>
</div>

<style>
  .lightbox { position: fixed; inset: 0; background: rgba(0,0,0,0.95);
    display: grid; grid-template-columns: 60px 1fr 300px; z-index: 100; }
  .media { display: grid; place-items: center; padding: 20px; grid-column: 2; }
  .media img, .media video { max-width: 100%; max-height: 100%; object-fit: contain; }
  .nav { background: transparent; color: #fff; border: none; font-size: 48px;
    cursor: pointer; }
  .nav:disabled { opacity: 0.3; cursor: default; }
  .info { background: var(--bg-2); padding: 20px; overflow-y: auto;
    border-left: 1px solid var(--border); }
  .info dl { display: grid; grid-template-columns: auto 1fr; gap: 6px 12px; color: var(--fg-dim); margin: 10px 0; }
  dt { text-transform: uppercase; font-size: 11px; }
  dd { margin: 0; color: var(--fg); }
  .dl { display: inline-block; background: var(--accent); color: #fff;
    padding: 8px 14px; border-radius: var(--radius); text-decoration: none; }
  @media (max-width: 768px) {
    .lightbox { grid-template-columns: 1fr; grid-template-rows: 1fr auto; }
    .nav { display: none; }
    .info { border-left: none; border-top: 1px solid var(--border); }
  }
</style>
