<!-- web/src/routes/file/[id]/+page.svelte -->
<script lang="ts">
  import { page } from '$app/stores';
  import Lightbox from '$lib/components/Lightbox.svelte';
  import { api } from '$lib/api';

  let file: any = null;
  let neighbors: number[] = [];

  async function load() {
    const id = +($page.params.id as string);
    file = await api.file(id);
    if (file?.folder_id != null) {
      try {
        // Derive the folder path from the file's relative_path to reuse the
        // existing /api/folder/{path} endpoint.
        const relParent = (file.relative_path ?? '').split('/').slice(0, -1).join('/');
        const folder = await api.folder(relParent, { limit: 50000 });
        neighbors = folder.files.map((f: any) => f.id);
      } catch {
        neighbors = [];
      }
    } else {
      neighbors = [];
    }
  }

  // Re-run when the route param changes (e.g. after arrow navigation).
  $: if ($page.params.id) load();
</script>

{#if file}<Lightbox {file} {neighbors} />{/if}
