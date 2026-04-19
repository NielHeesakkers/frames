<!-- web/src/lib/components/FolderTreeNode.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { currentFolderPath } from '$lib/stores';
  import Self from './FolderTreeNode.svelte';

  type Node = {
    id: number;
    path: string;
    name: string;
    has_child: boolean;
    items: number;
    children?: Node[];
    expanded?: boolean;
    loading?: boolean;
  };

  export let node: Node;
  export let depth = 0;
  export let onUpdate: () => void;

  async function loadChildren() {
    if (node.children) return;
    node.loading = true;
    onUpdate();
    try {
      const kids = await api.tree(node.path);
      node.children = kids.map((k) => ({ ...k }));
    } finally {
      node.loading = false;
      onUpdate();
    }
  }

  // Single-click: navigate AND expand (if collapsible).
  async function onRowClick() {
    currentFolderPath.set(node.path);
    goto('/browse/' + node.path.split('/').map(encodeURIComponent).join('/'));
    if (node.has_child) {
      if (!node.expanded) {
        node.expanded = true;
        await loadChildren();
      }
      onUpdate();
    }
  }

  // Clicking the arrow alone only toggles (without navigating).
  async function onToggle(e: MouseEvent) {
    e.stopPropagation();
    node.expanded = !node.expanded;
    if (node.expanded) await loadChildren();
    onUpdate();
  }
</script>

<li>
  <div class="row" class:active={$currentFolderPath === node.path}
       style="padding-left: {10 + depth * 14}px"
       on:click={onRowClick}>
    {#if node.has_child}
      <button class="toggle" on:click={onToggle}>{node.expanded ? '▾' : '▸'}</button>
    {:else}
      <span class="toggle" />
    {/if}
    <span class="name">{node.name || 'Photos'}</span>
    <span class="count">{node.items}</span>
  </div>

  {#if node.expanded && node.children}
    <ul class="sub">
      {#each node.children as c (c.id)}
        <Self node={c} depth={depth + 1} {onUpdate} />
      {/each}
    </ul>
  {/if}

  {#if node.loading}
    <div class="loading" style="padding-left: {14 + depth * 14}px">loading…</div>
  {/if}
</li>

<style>
  li { list-style: none; }
  .sub { list-style: none; padding: 0; margin: 0; }
  .row { display: grid; grid-template-columns: 20px 1fr auto; align-items: center;
    gap: 4px; padding: 4px 10px; cursor: pointer; border-radius: 3px; }
  .row:hover { background: rgba(255,255,255,0.05); }
  .row.active { background: rgba(74,124,255,0.15); color: var(--accent); }
  .toggle { width: 20px; height: 20px; background: transparent; border: none;
    color: var(--fg-dim); font-size: 12px; padding: 0; cursor: pointer; }
  .count { color: var(--fg-dim); font-size: 11px; }
  .loading { color: var(--fg-dim); font-size: 11px; padding: 2px 10px; }
</style>
