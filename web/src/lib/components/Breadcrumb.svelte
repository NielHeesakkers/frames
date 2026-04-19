<!-- web/src/lib/components/Breadcrumb.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { currentFolderPath } from '$lib/stores';
  $: parts = $currentFolderPath === '' ? [] : $currentFolderPath.split('/');
</script>

<nav>
  <a href="#" on:click|preventDefault={() => { currentFolderPath.set(''); goto('/browse'); }}>Photos</a>
  {#each parts as p, i}
    <span class="sep">›</span>
    <a href="#" on:click|preventDefault={() => {
         const sub = parts.slice(0, i + 1).join('/');
         currentFolderPath.set(sub);
         goto('/browse/' + parts.slice(0, i + 1).map(encodeURIComponent).join('/'));
       }}>{p}</a>
  {/each}
</nav>

<style>
  nav { display: flex; align-items: center; gap: 6px; font-size: 14px; }
  a { color: var(--fg); text-decoration: none; }
  a:hover { color: var(--accent); }
  .sep { color: var(--fg-dim); }
</style>
