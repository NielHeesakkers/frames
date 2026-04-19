<!-- web/src/lib/components/FolderTree.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
  import { currentFolderPath } from '$lib/stores';
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
  let lastExpandedFor = '';

  function refresh() { roots = roots; }

  // Walk from the roots and make sure every ancestor of `targetPath` is
  // expanded and its children are loaded. Leaves the tail node (the target
  // itself) collapsed — it becomes the active row without forcing its own
  // children to load.
  async function ensurePathExpanded(targetPath: string) {
    if (!targetPath) return;
    const parts = targetPath.split('/');
    let container: TreeNode[] = roots;
    let accum = '';
    for (let i = 0; i < parts.length; i++) {
      accum = accum ? `${accum}/${parts[i]}` : parts[i];
      const node = container.find((n) => n.path === accum);
      if (!node) break;
      // Expand every ancestor (everything except the final target).
      if (i < parts.length - 1) {
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

  onMount(async () => {
    const top = await api.tree('');
    roots = top.map((t) => ({ ...t }));
    if ($currentFolderPath) {
      lastExpandedFor = $currentFolderPath;
      await ensurePathExpanded($currentFolderPath);
    }
  });

  // React to later path changes (breadcrumb clicks, folder-card nav, etc.).
  $: if ($currentFolderPath && roots.length > 0 && $currentFolderPath !== lastExpandedFor) {
    lastExpandedFor = $currentFolderPath;
    ensurePathExpanded($currentFolderPath);
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
