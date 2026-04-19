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

  function aspect(f: any): number {
    return f.width && f.height ? f.width / f.height : 1;
  }
</script>

<div class="timeline" style="--size: {itemSize}px">
  {#if groups}
    {#each groups as g (g.key)}
      <h4 class="month">{g.label} <span class="count">· {g.items.length}</span></h4>
      {#if $thumbShape === 'original'}
        <div class="jgrid">
          {#each g.items as f (f.id)}
            <div class="jslot"
                 style="flex-grow: {aspect(f)}; flex-basis: calc({aspect(f)} * var(--size));
                        min-width: calc({aspect(f)} * var(--size) * 0.55);
                        aspect-ratio: {aspect(f)};">
              <GridItem file={f} size={itemSize} on:context />
            </div>
          {/each}
          <!-- A final greedy "invisible" item keeps the last row from
               stretching items absurdly wide when it's half-full. -->
          <div class="jfiller" />
        </div>
      {:else}
        <div class="sq">
          {#each g.items as f (f.id)}
            <GridItem file={f} size={itemSize} on:context />
          {/each}
        </div>
      {/if}
    {/each}
  {:else if $thumbShape === 'original'}
    <div class="jgrid">
      {#each files as f (f.id)}
        <div class="jslot"
             style="flex-grow: {aspect(f)}; flex-basis: calc({aspect(f)} * var(--size));
                    min-width: calc({aspect(f)} * var(--size) * 0.55);
                    aspect-ratio: {aspect(f)};">
          <GridItem file={f} size={itemSize} on:context />
        </div>
      {/each}
      <div class="jfiller" />
    </div>
  {:else}
    <div class="sq">
      {#each files as f (f.id)}
        <GridItem file={f} size={itemSize} on:context />
      {/each}
    </div>
  {/if}
</div>

<style>
  .timeline { display: flex; flex-direction: column;
    content-visibility: auto; padding: 0 8px 14px; width: 100%; }
  .sq { display: grid; grid-template-columns: repeat(auto-fill, var(--size));
    gap: 4px; padding-bottom: 10px; }
  /* Justified rows via flexbox: each slot's flex-grow + flex-basis are its
     aspect ratio. Rows wrap and distribute leftover space proportionally,
     so they always fill container width exactly. */
  .jgrid { display: flex; flex-wrap: wrap; gap: 4px; padding-bottom: 10px; }
  .jslot { min-height: 60px; position: relative; }
  /* Force the nested GridItem's root div (.item) to fill the slot exactly,
     overriding its own inline width/height. Svelte needs `>` outside :global. */
  .jslot > :global(.item) { width: 100% !important; height: 100% !important; }
  /* Absorbs slack so a half-full last row doesn't stretch items to 2×+ width. */
  .jfiller { flex: 99 1 0; height: 0; pointer-events: none; }
  .month { position: sticky; top: 0; z-index: 2;
    background: var(--bg); margin: 0; padding: 14px 2px 8px;
    font-size: 13px; font-weight: 600; color: var(--fg);
    backdrop-filter: blur(6px); }
  .month .count { color: var(--fg-dim); font-weight: 400; margin-left: 4px; font-size: 12px; }
</style>
