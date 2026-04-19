<!-- web/src/routes/file/[id]/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import Lightbox from '$lib/components/Lightbox.svelte';
  import { api } from '$lib/api';

  let file: any = null;

  async function load() {
    const id = +($page.params.id as string);
    file = await api.file(id);
  }
  onMount(load);
</script>

{#if file}<Lightbox {file} neighbors={[]} />{/if}
