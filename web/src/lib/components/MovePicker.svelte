<!-- web/src/lib/components/MovePicker.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';

  export let onPick: (folderId: number, path: string) => void;
  export let onClose: () => void;

  type Node = { id: number; path: string; name: string; has_child: boolean; expanded?: boolean; children?: Node[] };
  let nodes: Node[] = [];

  async function loadChildren(n: Node) {
    if (n.children) return;
    n.children = (await api.tree(n.path)) as any;
    nodes = nodes;
  }

  onMount(async () => {
    nodes = (await api.tree('')) as any;
  });
</script>

<div class="backdrop" on:click={onClose}>
  <div class="dialog" on:click|stopPropagation>
    <h3>Move to…</h3>
    <ul class="tree">
      <li>
        <button on:click={() => onPick(0, '')}>Photos (root)</button>
      </li>
      {#each nodes as n}
        <li>
          <div class="row">
            <button class="toggle" on:click={() => { n.expanded = !n.expanded; loadChildren(n); nodes = nodes; }}>
              {n.has_child ? (n.expanded ? '▾' : '▸') : '•'}
            </button>
            <button class="name" on:click={() => onPick(n.id, n.path)}>{n.name}</button>
          </div>
          {#if n.expanded && n.children}
            <ul class="sub">
              {#each n.children as c}
                <li>
                  <button on:click={() => onPick(c.id, c.path)}>{c.name}</button>
                </li>
              {/each}
            </ul>
          {/if}
        </li>
      {/each}
    </ul>
    <div class="actions"><button on:click={onClose}>Cancel</button></div>
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.6); z-index: 150;
    display: grid; place-items: center; }
  .dialog { background: var(--bg-2); border-radius: 8px; padding: 20px;
    min-width: 420px; max-height: 70vh; overflow-y: auto; border: 1px solid var(--border); }
  .tree { list-style: none; padding: 0; margin: 12px 0; }
  .sub { padding-left: 18px; list-style: none; }
  .row { display: flex; gap: 4px; align-items: center; }
  .toggle { width: 24px; background: transparent; border: none; color: var(--fg-dim); }
  .name { background: transparent; border: none; color: var(--fg); text-align: left; padding: 4px 6px; }
  .name:hover { background: rgba(255,255,255,0.05); }
  .actions { display: flex; justify-content: flex-end; }
</style>
