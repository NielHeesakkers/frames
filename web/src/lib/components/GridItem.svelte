<!-- web/src/lib/components/GridItem.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { createEventDispatcher } from 'svelte';
  import { api } from '$lib/api';
  import { selection, thumbShape } from '$lib/stores';
  import StarRating from './StarRating.svelte';

  export let file: any;
  export let size = 160;

  const dispatch = createEventDispatcher<{ context: { file: any; x: number; y: number } }>();

  function onClick(e: MouseEvent) {
    if (e.shiftKey || e.metaKey || e.ctrlKey) {
      selection.update((s) => {
        const ns = new Set(s);
        ns.has(file.id) ? ns.delete(file.id) : ns.add(file.id);
        return ns;
      });
      return;
    }
    goto(`/file/${file.id}`);
  }

  function onContext(e: MouseEvent) {
    e.preventDefault();
    dispatch('context', { file, x: e.clientX, y: e.clientY });
  }

  async function onRatingChange(n: number) {
    const prev = file.rating ?? 0;
    file = { ...file, rating: n };
    try {
      await api.setRating(file.id, n);
    } catch {
      file = { ...file, rating: prev };
    }
  }

  let hovering = false;
  function onKey(e: KeyboardEvent) {
    if (!hovering) return;
    if (e.key >= '0' && e.key <= '5') {
      e.preventDefault();
      onRatingChange(Number(e.key));
    }
  }

  function formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
    return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
  }

  // Aspect-ratio mode: width is computed from the file's natural dims and the
  // fixed row height `size`. Falls back to square when dims are unknown.
  $: aspect = file.width && file.height ? file.width / file.height : 1;
  $: itemWidth = $thumbShape === 'original' ? Math.round(size * aspect) : size;
  $: itemHeight = size;
</script>

<svelte:window on:keydown={onKey} />

<div class="item"
     class:square={$thumbShape === 'square'}
     class:original={$thumbShape === 'original'}
     style="width: {itemWidth}px; height: {itemHeight}px"
     on:click={onClick} on:contextmenu={onContext}
     on:mouseenter={() => (hovering = true)}
     on:mouseleave={() => (hovering = false)}
     class:selected={$selection.has(file.id)}
     class:hovered={hovering}>
  {#if file.kind === 'other'}
    <div class="icon">📄</div>
  {:else if file.kind === 'video' && hovering}
    <!-- Hover-preview: play the original video muted+looped. Falls back to the
         thumbnail if the browser can't decode the source. -->
    <video src={`/api/original/${file.id}`}
           poster={`/api/thumb/${file.id}`}
           muted autoplay loop playsinline></video>
  {:else}
    <img src={`/api/thumb/${file.id}?v=${file.mtime ?? 0}`} loading="lazy" alt={file.name} />
  {/if}

  {#if file.kind === 'video'}<span class="badge">▶</span>{/if}

  {#if hovering}
    <div class="meta">
      <div class="name" title={file.name}>{file.name}</div>
      <div class="row">
        {#if file.width && file.height}<span>{file.width}×{file.height}</span>{/if}
        {#if file.size}<span>{formatSize(file.size)}</span>{/if}
      </div>
    </div>
  {/if}

  <div class="rating-overlay" class:show={hovering || (file.rating ?? 0) > 0}>
    <StarRating value={file.rating ?? 0} onChange={onRatingChange} size={Math.max(14, Math.round(size / 11))} />
  </div>
</div>

<style>
  .item { position: relative;
    background: var(--bg-2); border-radius: 3px; overflow: hidden;
    cursor: pointer; }
  .item.selected { outline: 3px solid var(--accent); }
  img, video { width: 100%; height: 100%; display: block; }
  .item.square img, .item.square video { object-fit: cover; }
  .item.original img, .item.original video { object-fit: cover; }
  .icon { width: 100%; height: 100%; display: grid; place-items: center;
    font-size: 36px; color: var(--fg-dim); }
  .badge { position: absolute; bottom: 4px; right: 4px; background: rgba(0,0,0,0.6);
    color: #fff; border-radius: 50%; width: 22px; height: 22px; display: grid;
    place-items: center; font-size: 11px; }

  .meta { position: absolute; left: 0; right: 0; top: 0;
    padding: 8px 10px 12px;
    background: linear-gradient(to bottom, rgba(0,0,0,0.8), rgba(0,0,0,0));
    color: #fff; font-size: 11px; pointer-events: none; }
  .meta .name { font-weight: 500; white-space: nowrap; overflow: hidden;
    text-overflow: ellipsis; margin-bottom: 2px; }
  .meta .row { display: flex; gap: 8px; color: rgba(255,255,255,0.8); font-size: 10px; }

  .rating-overlay { position: absolute; left: 0; right: 0; bottom: 0;
    padding: 14px 6px 4px;
    background: linear-gradient(to top, rgba(0,0,0,0.8), rgba(0,0,0,0));
    display: flex; justify-content: flex-start;
    opacity: 0; transition: opacity 0.1s;
    pointer-events: none; }
  .rating-overlay.show { opacity: 1; pointer-events: auto; }
  .item.hovered .rating-overlay { opacity: 1; pointer-events: auto; }
</style>
