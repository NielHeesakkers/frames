<!-- web/src/routes/browse/[...path]/+page.svelte -->
<script lang="ts">
  import { page } from '$app/stores';
  import { currentFolderPath } from '$lib/stores';
  import BrowseRoot from '../+page.svelte';
  $: currentFolderPath.set(decodeURIComponent($page.params.path ?? ''));
</script>

<BrowseRoot />
