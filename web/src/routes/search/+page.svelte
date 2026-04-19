<!-- web/src/routes/search/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { api } from '$lib/api';
  import Grid from '$lib/components/Grid.svelte';

  let files: any[] = [];
  let loading = true;

  async function run() {
    loading = true;
    const q: Record<string, string> = {};
    const qp = $page.url.searchParams;
    for (const k of ['q', 'date_from', 'date_to', 'camera', 'kind']) {
      const v = qp.get(k); if (v) q[k] = v;
    }
    const r = await api.search(q);
    files = r.files;
    loading = false;
  }

  onMount(run);
  $: $page.url, run();
</script>

<div class="page">
  <h2>Search {$page.url.searchParams.get('q') ? `"${$page.url.searchParams.get('q')}"` : ''}</h2>
  {#if loading}<p>Loading…</p>
  {:else if files.length === 0}<p>No results.</p>
  {:else}<Grid {files} />{/if}
</div>

<style>
  .page { padding: 16px; display: flex; flex-direction: column; min-height: 0; flex: 1; }
  h2 { margin: 0 0 10px; }
</style>
