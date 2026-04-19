<!-- web/src/lib/components/Grid.svelte -->
<script lang="ts">
  import GridItem from './GridItem.svelte';
  import { density, sortMode, thumbShape } from '$lib/stores';

  export let files: any[] = [];
  export let groupBy: 'none' | 'day' | 'week' | 'month' = 'month';

  $: itemSize = $density === 'small' ? 120 : $density === 'large' ? 220 : 160;
  // Grouping is only meaningful when sorting by capture date.
  $: effectiveGroupBy = $sortMode === 'takenAt' ? groupBy : 'none';

  type Group = { key: string; label: string; items: any[] };
  const monthFmt = new Intl.DateTimeFormat('nl-NL', { month: 'long', year: 'numeric' });
  const dayFmt = new Intl.DateTimeFormat('nl-NL', { weekday: 'short', day: 'numeric', month: 'long', year: 'numeric' });

  function dateOf(f: any): Date | null {
    const raw: string | undefined = f.taken_at;
    if (raw) {
      const d = new Date(raw.replace(' ', 'T'));
      if (!isNaN(d.getTime())) return d;
    }
    if (f.mtime) return new Date(f.mtime * 1000);
    return null;
  }

  // ISO-week helpers — Monday-based week, year-week like "2024-W14".
  function isoWeek(d: Date): { year: number; week: number } {
    const target = new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()));
    const dayNum = (target.getUTCDay() + 6) % 7; // 0 = Monday
    target.setUTCDate(target.getUTCDate() - dayNum + 3);
    const firstThursday = new Date(Date.UTC(target.getUTCFullYear(), 0, 4));
    const diff = target.getTime() - firstThursday.getTime();
    const week = 1 + Math.round(diff / (7 * 24 * 3600 * 1000));
    return { year: target.getUTCFullYear(), week };
  }

  function groupKeyOf(f: any, mode: string): { key: string; label: string } {
    const d = dateOf(f);
    if (!d) return { key: 'unknown', label: 'Zonder datum' };
    if (mode === 'day') {
      const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
      return { key, label: dayFmt.format(d).replace(/^./, (c) => c.toUpperCase()) };
    }
    if (mode === 'week') {
      const { year, week } = isoWeek(d);
      return { key: `${year}-W${week}`, label: `Week ${week} · ${year}` };
    }
    // month (default)
    const key = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`;
    return { key, label: monthFmt.format(d).replace(/^./, (c) => c.toUpperCase()) };
  }

  $: groups = (() => {
    if (effectiveGroupBy === 'none') return null as Group[] | null;
    const out: Group[] = [];
    let cur: Group | null = null;
    for (const f of files) {
      const { key, label } = groupKeyOf(f, effectiveGroupBy);
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
