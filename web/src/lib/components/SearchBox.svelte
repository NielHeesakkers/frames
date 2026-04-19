<!-- web/src/lib/components/SearchBox.svelte -->
<script lang="ts">
  import { goto } from '$app/navigation';
  let q = '';
  let timer: any = null;
  function change() {
    clearTimeout(timer);
    timer = setTimeout(() => {
      if (q.trim()) goto(`/search?q=${encodeURIComponent(q.trim())}`);
    }, 250);
  }
</script>

<input class="search" type="search" placeholder="Search filename, date (YYYY-MM-DD), camera…"
       bind:value={q} on:input={change} />

<style>
  .search { width: 100%; max-width: 420px; }
</style>
