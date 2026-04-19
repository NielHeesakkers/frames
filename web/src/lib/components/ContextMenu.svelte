<!-- web/src/lib/components/ContextMenu.svelte -->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  export let x = 0;
  export let y = 0;
  export let items: Array<{ label: string; onSelect: () => void; danger?: boolean }> = [];
  export let onClose: () => void;

  function handleDoc(e: MouseEvent) { onClose(); }
  onMount(() => window.addEventListener('click', handleDoc));
  onDestroy(() => window.removeEventListener('click', handleDoc));
</script>

<div class="menu" style="left: {x}px; top: {y}px" on:click|stopPropagation>
  {#each items as it}
    <button class:danger={it.danger} on:click={() => { it.onSelect(); onClose(); }}>
      {it.label}
    </button>
  {/each}
</div>

<style>
  .menu { position: fixed; z-index: 200; background: var(--bg-2);
    border: 1px solid var(--border); border-radius: var(--radius);
    min-width: 160px; box-shadow: 0 6px 24px rgba(0,0,0,0.4);
    display: flex; flex-direction: column; }
  button { background: transparent; text-align: left; border: none;
    padding: 8px 12px; color: var(--fg); border-radius: 0; }
  button:hover { background: rgba(255,255,255,0.05); }
  button.danger { color: var(--danger); }
</style>
