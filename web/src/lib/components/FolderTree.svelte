<!-- web/src/lib/components/FolderTree.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { get } from 'svelte/store';
  import { api } from '$lib/api';
  import { currentFolderPath, expandedFolders, setExpanded } from '$lib/stores';
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
  let syncedPath = '';

  function refresh() { roots = roots; }

  // Ensure every path in `paths` is expanded and its children are loaded.
  // Paths are processed in dependency order (shorter first) so parents are
  // populated before children are visited.
  async function expandPaths(paths: Iterable<string>) {
    const sorted = [...paths].filter(Boolean).sort((a, b) => a.length - b.length);
    for (const p of sorted) {
      const parts = p.split('/');
      let container: TreeNode[] = roots;
      let accum = '';
      for (let i = 0; i < parts.length; i++) {
        accum = accum ? `${accum}/${parts[i]}` : parts[i];
        const node = container.find((n) => n.path === accum);
        if (!node) break;
        if (!node.children) {
          const kids = await api.tree(node.path);
          node.children = kids.map((k) => ({ ...k }));
        }
        node.expanded = true;
        container = node.children ?? [];
      }
    }
    refresh();
  }

  // Add all ancestor paths of `targetPath` to the expanded set (so a
  // navigate-to also opens the chain), then apply the full set.
  async function expandToPath(targetPath: string) {
    if (!targetPath) return;
    const parts = targetPath.split('/');
    const ancestors: string[] = [];
    let accum = '';
    // Every ancestor except the leaf itself.
    for (let i = 0; i < parts.length - 1; i++) {
      accum = accum ? `${accum}/${parts[i]}` : parts[i];
      ancestors.push(accum);
    }
    for (const a of ancestors) setExpanded(a, true);
    await expandPaths(get(expandedFolders));
  }

  onMount(async () => {
    const top = await api.tree('');
    roots = top.map((t) => ({ ...t }));
    // Restore the user's manually-expanded tree shape …
    await expandPaths(get(expandedFolders));
    // … then ensure the current path is reachable on top of that.
    if ($currentFolderPath) {
      syncedPath = $currentFolderPath;
      await expandToPath($currentFolderPath);
    }
  });

  // React to later path changes without ever *collapsing* anything.
  $: if ($currentFolderPath && roots.length > 0 && $currentFolderPath !== syncedPath) {
    syncedPath = $currentFolderPath;
    expandToPath($currentFolderPath);
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
