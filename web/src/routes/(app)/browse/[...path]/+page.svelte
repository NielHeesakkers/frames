<!-- /browse/{...path} — nested folder listing. Path is derived from the URL. -->
<script lang="ts">
  import { page } from '$app/stores';
  import BrowseView from '$lib/components/BrowseView.svelte';

  $: path = decodeURIComponent(($page.params as any)?.path ?? '');
</script>

<BrowseView {path} />
