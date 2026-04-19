<!-- web/src/lib/components/Breadcrumb.svelte -->
<script lang="ts">
  import { currentFolderPath } from '$lib/stores';
  $: parts = $currentFolderPath === '' ? [] : $currentFolderPath.split('/');
</script>

<nav>
  <a href="/browse">Photos</a>
  {#each parts as p, i}
    <span class="sep">›</span>
    <a href={'/browse/' + parts.slice(0, i + 1).map(encodeURIComponent).join('/')}>{p}</a>
  {/each}
</nav>

<style>
  nav { display: flex; align-items: center; gap: 6px; font-size: 14px; }
  a { color: var(--fg); text-decoration: none; }
  a:hover { color: var(--accent); }
  .sep { color: var(--fg-dim); }
</style>
