<!-- web/src/lib/components/FolderTreeNode.svelte -->
<script lang="ts">
  import { api } from '$lib/api';
  import { currentFolderPath, setExpanded } from '$lib/stores';
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

  // Single-click: navigate only. Expansion follows the auto-expand logic
  // in FolderTree (and the chevron toggle below).
  function onRowClick() {
    // href-based navigation is provided by the <a> element; no JS needed here.
  }

  // Clicking the arrow toggles expansion and persists it to the store so the
  // tree keeps its shape across page refreshes.
  async function onToggle(e: MouseEvent) {
    e.stopPropagation();
    e.preventDefault();
    const nowExpanded = !node.expanded;
    node.expanded = nowExpanded;
    setExpanded(node.path, nowExpanded);
    if (nowExpanded) await loadChildren();
    onUpdate();
  }
</script>

<li>
  <a class="row" class:active={$currentFolderPath === node.path}
     style="padding-left: {10 + depth * 14}px"
     href={'/browse/' + node.path.split('/').map(encodeURIComponent).join('/')}>
    {#if node.has_child}
      <button type="button" class="toggle" on:click={onToggle}>{node.expanded ? '▾' : '▸'}</button>
    {:else}
      <span class="toggle" />
    {/if}
    <span class="name">{node.name || 'Photos'}</span>
    <span class="count">{node.items}</span>
  </a>

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
    gap: 4px; padding: 4px 10px; cursor: pointer; border-radius: 3px;
    text-decoration: none; color: inherit; }
  .row:hover { background: rgba(255,255,255,0.05); }
  .row.active { background: rgba(74,124,255,0.15); color: var(--accent); }
  .toggle { width: 20px; height: 20px; background: transparent; border: none;
    color: var(--fg-dim); font-size: 12px; padding: 0; cursor: pointer; }
  .count { color: var(--fg-dim); font-size: 11px; }
  .loading { color: var(--fg-dim); font-size: 11px; padding: 2px 10px; }
</style>
