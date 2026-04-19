<!-- web/src/lib/components/GridItem.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { createEventDispatcher } from 'svelte';
  import { api } from '$lib/api';
  import { selection } from '$lib/stores';
  import StarRating from './StarRating.svelte';

  export let file: any;
  export let size = 160;

  const dispatch = createEventDispatcher<{ context: { file: any; x: number; y: number } }>();

  function onClick(e: MouseEvent) {
    // Clicks that originate inside the hover rating widget are handled by it
    // (it stops propagation), so this only runs on clicks on the image itself.
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

  // Numeric keyboard shortcuts 1..5 / 0 when the thumb is hovered.
  let hovering = false;
  function onKey(e: KeyboardEvent) {
    if (!hovering) return;
    if (e.key >= '0' && e.key <= '5') {
      e.preventDefault();
      onRatingChange(Number(e.key));
    }
  }
</script>

<svelte:window on:keydown={onKey} />

<div class="item" style="--size: {size}px"
     on:click={onClick} on:contextmenu={onContext}
     on:mouseenter={() => (hovering = true)}
     on:mouseleave={() => (hovering = false)}
     class:selected={$selection.has(file.id)}
     class:hovered={hovering}>
  {#if file.kind === 'other'}
    <div class="icon">📄</div>
  {:else}
    <img src={`/api/thumb/${file.id}`} loading="lazy" alt={file.name} />
  {/if}

  {#if file.kind === 'video'}<span class="badge">▶</span>{/if}

  <div class="rating-overlay" class:show={hovering || (file.rating ?? 0) > 0}>
    <StarRating value={file.rating ?? 0} onChange={onRatingChange} size={Math.max(14, Math.round(size / 11))} />
  </div>
</div>

<style>
  .item { position: relative; width: var(--size); height: var(--size);
    background: var(--bg-2); border-radius: 3px; overflow: hidden;
    cursor: pointer; }
  .item.selected { outline: 3px solid var(--accent); }
  img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .icon { width: 100%; height: 100%; display: grid; place-items: center;
    font-size: 36px; color: var(--fg-dim); }
  .badge { position: absolute; bottom: 4px; right: 4px; background: rgba(0,0,0,0.6);
    color: #fff; border-radius: 50%; width: 22px; height: 22px; display: grid;
    place-items: center; font-size: 11px; }

  /* Rating overlay: always visible when rated, fades in on hover otherwise. */
  .rating-overlay { position: absolute; left: 0; right: 0; bottom: 0;
    padding: 6px 6px 4px;
    background: linear-gradient(to top, rgba(0,0,0,0.7), rgba(0,0,0,0));
    display: flex; justify-content: flex-start;
    opacity: 0; transition: opacity 0.1s;
    pointer-events: none; }
  .rating-overlay.show { opacity: 1; pointer-events: auto; }
  .item.hovered .rating-overlay { opacity: 1; pointer-events: auto; }
</style>
