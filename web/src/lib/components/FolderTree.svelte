<!-- web/src/lib/components/FolderTree.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api';
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

  onMount(async () => {
    const top = await api.tree('');
    roots = top.map((t) => ({ ...t }));
  });

  function refresh() { roots = roots; }
</script>

<ul class="tree">
  {#each roots as n (n.id)}
    <FolderTreeNode node={n} depth={0} onUpdate={refresh} />
  {/each}
</ul>

<style>
  .tree { list-style: none; padding: 0; margin: 8px 0; flex: 1; overflow-y: auto; }
</style>
