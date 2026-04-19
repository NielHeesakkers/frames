<!-- (app)/browse/[...path]/+page.svelte — render same component as root /browse.
     The inner component derives the folder path from $page.params itself. -->
<script lang="ts">
  import BrowseRoot from '../+page.svelte';
</script>

<BrowseRoot />
