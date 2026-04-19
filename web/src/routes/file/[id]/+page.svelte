<!-- web/src/routes/file/[id]/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import Lightbox from '$lib/components/Lightbox.svelte';
  import { api } from '$lib/api';

  let file: any = null;
  let neighbors: number[] = [];

  async function load() {
    const id = +($page.params.id as string);
    // For now, fetch only the file metadata via folder listing heuristic:
    // fall back to a lightweight endpoint or just show the file alone.
    // A future task could add a /api/file/{id} metadata endpoint.
    // Here we rely on thumb endpoint + no metadata; a richer version arrives in Task 51.
    file = { id, name: `#${id}`, size: 0, kind: 'image' };
  }
  onMount(load);
</script>

{#if file}<Lightbox {file} {neighbors} />{/if}
