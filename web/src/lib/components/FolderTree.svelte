<!-- web/src/lib/components/FolderTree.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { currentFolderPath } from '$lib/stores';

  type Node = { id: number; path: string; name: string; has_child: boolean; items: number;
                children?: Node[]; expanded?: boolean };

  let roots: Node[] = [];

  async function loadChildren(n: Node) {
    if (n.children) return;
    const kids = await api.tree(n.path);
    n.children = kids.map((k) => ({ ...k }));
    roots = roots;
  }

  async function toggle(n: Node) {
    n.expanded = !n.expanded;
    if (n.expanded) await loadChildren(n);
    roots = roots;
  }

  function select(n: Node) {
    currentFolderPath.set(n.path);
    goto('/browse/' + n.path.split('/').map(encodeURIComponent).join('/'));
  }

  onMount(async () => {
    const top = await api.tree('');
    roots = top.map((t) => ({ ...t }));
  });
</script>

<ul class="tree">
  {#each roots as n}
    <li>
      <div class="row" class:active={$currentFolderPath === n.path} on:click={() => select(n)}>
        {#if n.has_child}
          <button class="toggle" on:click|stopPropagation={() => toggle(n)}>{n.expanded ? '▾' : '▸'}</button>
        {:else}
          <span class="toggle" />
        {/if}
        <span class="name">{n.name || 'Photos'}</span>
        <span class="count">{n.items}</span>
      </div>
      {#if n.expanded && n.children}
        <ul class="tree sub">
          {#each n.children as c}
            <li>
              <div class="row" class:active={$currentFolderPath === c.path} on:click={() => select(c)}>
                {#if c.has_child}
                  <button class="toggle" on:click|stopPropagation={() => toggle(c)}>{c.expanded ? '▾' : '▸'}</button>
                {:else}
                  <span class="toggle" />
                {/if}
                <span class="name">{c.name}</span>
                <span class="count">{c.items}</span>
              </div>
            </li>
          {/each}
        </ul>
      {/if}
    </li>
  {/each}
</ul>

<style>
  .tree { list-style: none; padding: 0; margin: 8px 0; flex: 1; overflow-y: auto; }
  .sub { padding-left: 14px; margin: 0; }
  .row { display: grid; grid-template-columns: 20px 1fr auto; align-items: center;
    gap: 4px; padding: 4px 10px; cursor: pointer; border-radius: 3px; }
  .row:hover { background: rgba(255,255,255,0.05); }
  .row.active { background: rgba(74,124,255,0.15); color: var(--accent); }
  .toggle { width: 20px; height: 20px; background: transparent; border: none;
    color: var(--fg-dim); font-size: 12px; padding: 0; }
  .count { color: var(--fg-dim); font-size: 11px; }
</style>
