<!-- web/src/lib/components/GridItem.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { createEventDispatcher } from 'svelte';
  import { selection } from '$lib/stores';

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
</script>

<div class="item" style="--size: {size}px" on:click={onClick} on:contextmenu={onContext}
     class:selected={$selection.has(file.id)}>
  {#if file.kind === 'other'}
    <div class="icon">📄</div>
  {:else}
    <img src={`/api/thumb/${file.id}`} loading="lazy" alt={file.name} />
  {/if}
  {#if file.kind === 'video'}<span class="badge">▶</span>{/if}
  {#if (file.rating ?? 0) > 0}
    <span class="stars">{'★'.repeat(file.rating)}</span>
  {/if}
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
  .stars { position: absolute; bottom: 4px; left: 4px;
    color: #f5a524; text-shadow: 0 1px 2px rgba(0,0,0,0.6);
    font-size: 11px; letter-spacing: -1px; }
</style>
