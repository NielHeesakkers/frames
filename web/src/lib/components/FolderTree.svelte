<!-- web/src/lib/components/FolderTree.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { api } from '$lib/api';
  import { currentFolderPath, expandedFolders } from '$lib/stores';
  import FolderTreeNode from './FolderTreeNode.svelte';

  export type TreeNode = {
    id: number;
    path: string;
    name: string;
    has_child: boolean;
    items: number;
    children?: TreeNode[];
    expanded?: boolean;
    loading?: boolean;
  };

  let roots: TreeNode[] = [];
  let syncedPath = '<init>';

  function refresh() { roots = roots; }

  // Walk the tree and apply the expanded set: every node whose path is in
  // `wantExpanded` is expanded (and its children loaded); everything else is
  // collapsed. Runs breadth-first via a recursive helper.
  async function applyExpanded(wantExpanded: Set<string>) {
    async function walk(nodes: TreeNode[]) {
      for (const n of nodes) {
        const shouldOpen = wantExpanded.has(n.path);
        if (shouldOpen) {
          if (!n.children) {
            const kids = await api.tree(n.path);
            n.children = kids.map((k) => ({ ...k }));
          }
          n.expanded = true;
          if (n.children) await walk(n.children);
        } else {
          n.expanded = false;
        }
      }
    }
    await walk(roots);
    refresh();
  }

  // Build the set of paths that should be expanded given the current URL path.
  // = every ancestor of `currentPath` (NOT the leaf itself — expanding the
  // leaf just loads its children which we don't need until the user opens it).
  function ancestorsOf(currentPath: string): Set<string> {
    const s = new Set<string>();
    if (!currentPath) return s;
    const parts = currentPath.split('/');
    let accum = '';
    for (let i = 0; i < parts.length - 1; i++) {
      accum = accum ? `${accum}/${parts[i]}` : parts[i];
      s.add(accum);
    }
    return s;
  }

  onMount(async () => {
    const top = await api.tree('');
    roots = top.map((t) => ({ ...t }));
    // First render: combine the persisted set with the ancestors of the
    // current URL. (On hard refresh the persisted set matches the last
    // navigation, so this is effectively idempotent.)
    const persisted = get(expandedFolders);
    const wanted = new Set<string>([...persisted, ...ancestorsOf($currentFolderPath)]);
    expandedFolders.set(wanted);
    syncedPath = $currentFolderPath;
    await applyExpanded(wanted);
  });

  // React to navigation: reset the expanded set to exactly the ancestor chain
  // of the new path, so siblings of ancestors auto-collapse.
  $: if (roots.length > 0 && $currentFolderPath !== syncedPath) {
    syncedPath = $currentFolderPath;
    const wanted = ancestorsOf($currentFolderPath);
    expandedFolders.set(wanted);
    applyExpanded(wanted);
  }
</script>

<ul class="tree">
  {#each roots as n (n.id)}
    <FolderTreeNode node={n} depth={0} onUpdate={refresh} />
  {/each}
</ul>

<style>
  .tree { list-style: none; padding: 0; margin: 8px 0; flex: 1; overflow-y: auto; }
</style>
