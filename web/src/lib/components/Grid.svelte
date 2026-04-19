<!-- web/src/lib/components/Grid.svelte -->
<script lang="ts">
  import GridItem from './GridItem.svelte';
  import { density } from '$lib/stores';

  export let files: any[] = [];

  $: itemSize = $density === 'small' ? 120 : $density === 'large' ? 220 : 160;
</script>

<div class="grid" style="--size: {itemSize}px">
  {#each files as f (f.id)}
    <GridItem file={f} size={itemSize} />
  {/each}
</div>

<style>
  .grid { display: grid; grid-template-columns: repeat(auto-fill, var(--size));
    gap: 4px; padding: 8px; overflow-y: auto; flex: 1;
    content-visibility: auto; }
</style>
