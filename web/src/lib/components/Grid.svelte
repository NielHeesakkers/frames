<!-- web/src/lib/components/Grid.svelte -->
<script lang="ts">
  import GridItem from './GridItem.svelte';
  import { density, sortMode, thumbShape } from '$lib/stores';

  export let files: any[] = [];

  $: itemSize = $density === 'small' ? 120 : $density === 'large' ? 220 : 160;
  $: groupByMonth = $sortMode === 'takenAt';

  type Group = { key: string; label: string; items: any[] };
  const monthFmt = new Intl.DateTimeFormat('nl-NL', { month: 'long', year: 'numeric' });

  function monthKeyOf(f: any): { key: string; label: string } {
    const raw: string | undefined = f.taken_at;
    let d: Date | null = null;
    if (raw) {
      d = new Date(raw.replace(' ', 'T'));
      if (isNaN(d.getTime())) d = null;
    }
    if (!d && f.mtime) d = new Date(f.mtime * 1000);
    if (!d) return { key: 'unknown', label: 'Zonder datum' };
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
    const label = monthFmt.format(d).replace(/^./, (c) => c.toUpperCase());
    return { key, label };
  }

  $: groups = (() => {
    if (!groupByMonth) return null as Group[] | null;
    const out: Group[] = [];
    let cur: Group | null = null;
    for (const f of files) {
      const { key, label } = monthKeyOf(f);
      if (!cur || cur.key !== key) {
        cur = { key, label, items: [] };
        out.push(cur);
      }
      cur.items.push(f);
    }
    return out;
  })();

  // For aspect-ratio mode, flex-wrap on a parent with fixed row height gives
  // a justified-rows layout automatically.
  $: layoutClass = $thumbShape === 'original' ? 'justified' : 'square';
</script>

{#if groups}
  <div class="timeline" style="--size: {itemSize}px">
    {#each groups as g (g.key)}
      <h4 class="month">{g.label} <span class="count">· {g.items.length}</span></h4>
      <div class={`grid ${layoutClass}`}>
        {#each g.items as f (f.id)}
          <GridItem file={f} size={itemSize} on:context />
        {/each}
      </div>
    {/each}
  </div>
{:else}
  <div class={`grid solo ${layoutClass}`} style="--size: {itemSize}px">
    {#each files as f (f.id)}
      <GridItem file={f} size={itemSize} on:context />
    {/each}
  </div>
{/if}

<style>
  .timeline { display: flex; flex-direction: column; overflow-y: auto; flex: 1;
    content-visibility: auto; padding-bottom: 8px; }
  /* Square grid: uniform cells. */
  .grid.square { display: grid;
    grid-template-columns: repeat(auto-fill, var(--size));
    gap: 4px; padding: 0 8px 14px; }
  /* Justified rows: flex-wrap gives rows of fixed height with natural widths. */
  .grid.justified { display: flex; flex-wrap: wrap; gap: 4px;
    padding: 0 8px 14px; }
  .grid.solo { overflow-y: auto; flex: 1; padding: 8px;
    content-visibility: auto; }
  .grid.solo.square { padding: 8px; }
  .month { position: sticky; top: 0; z-index: 2;
    background: var(--bg); margin: 0; padding: 14px 10px 8px;
    font-size: 13px; font-weight: 600; color: var(--fg);
    backdrop-filter: blur(6px); }
  .month .count { color: var(--fg-dim); font-weight: 400; margin-left: 4px; font-size: 12px; }
</style>
