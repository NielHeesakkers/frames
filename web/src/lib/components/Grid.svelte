<!-- web/src/lib/components/Grid.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
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

  // Justified layout: for each row, scale items so they fill container width
  // exactly, preserving aspect ratios and row height near `targetH`.
  const gap = 4;
  const padSide = 8; // matches grid padding

  let containerWidth = 0;
  let containerEl: HTMLDivElement;

  function measure() {
    if (containerEl) containerWidth = containerEl.clientWidth;
  }

  onMount(() => {
    measure();
    // ResizeObserver catches parent-layout changes (sidebar toggles, dialogs);
    // a window-level resize listener is the safety net for browser width
    // changes that don't always fan out to the observed node.
    const ro = new ResizeObserver(measure);
    if (containerEl) ro.observe(containerEl);
    return () => ro.disconnect();
  });

  // Also remeasure on ANY file/thumb-shape change — the container may have
  // just been mounted (switching from "square" to "original"), in which case
  // clientWidth is already correct but `containerWidth` still holds the last
  // value; forcing a read keeps the first paint right.
  $: if (containerEl && (files || $thumbShape)) measure();

  type Sized = { file: any; h: number };
  function buildRows(list: any[], W: number, targetH: number): Sized[][] {
    const avail = Math.max(40, W - 2 * padSide);
    const rows: Sized[][] = [];
    let cur: { file: any; aspect: number }[] = [];
    let curAspects = 0;
    for (const f of list) {
      const a = f.width && f.height ? f.width / f.height : 1;
      // Pre-compute the row height that this new row would get *if* we added
      // the current file to it. If that height is < 70% of targetH, the row
      // has too much content — close it before adding.
      const projectedAspects = curAspects + a;
      const projectedRowW = projectedAspects * targetH + Math.max(0, cur.length) * gap;
      if (cur.length > 0 && projectedRowW > avail * 1.35) {
        // Close current row (scale to fit width).
        rows.push(closeRow(cur, avail, targetH, /*capUpscale*/ false));
        cur = [];
        curAspects = 0;
      }
      cur.push({ file: f, aspect: a });
      curAspects += a;
      // If the row is now visually full (scaled width ~= available), close it.
      const fullW = curAspects * targetH + (cur.length - 1) * gap;
      if (fullW >= avail) {
        rows.push(closeRow(cur, avail, targetH, false));
        cur = [];
        curAspects = 0;
      }
    }
    // Trailing (incomplete) row — don't upscale beyond targetH.
    if (cur.length > 0) rows.push(closeRow(cur, avail, targetH, /*capUpscale*/ true));
    return rows;
  }

  function closeRow(row: { file: any; aspect: number }[], avail: number, targetH: number, capUpscale: boolean): Sized[] {
    const sumA = row.reduce((s, x) => s + x.aspect, 0);
    if (sumA <= 0) return [];
    const rowAvail = avail - (row.length - 1) * gap;
    let h = rowAvail / sumA;
    if (capUpscale && h > targetH * 1.25) h = targetH; // trailing row: don't stretch to fill
    return row.map((x) => ({ file: x.file, h: Math.max(40, Math.round(h)) }));
  }

  $: rowsAll = $thumbShape === 'original' && containerWidth > 0
    ? buildRows(files, containerWidth, itemSize) : null;
  $: rowsPerGroup = $thumbShape === 'original' && containerWidth > 0 && groups
    ? groups.map(g => ({ ...g, rows: buildRows(g.items, containerWidth, itemSize) }))
    : null;
</script>

<svelte:window on:resize={measure} />

<div class="timeline" bind:this={containerEl} style="--size: {itemSize}px">
  {#if groups}
    {#each groups as g, gi (g.key)}
      <h4 class="month">{g.label} <span class="count">· {g.items.length}</span></h4>
      {#if $thumbShape === 'original' && rowsPerGroup}
        {#each rowsPerGroup[gi].rows as row, ri (ri)}
          <div class="jrow">
            {#each row as it (it.file.id)}
              <GridItem file={it.file} size={it.h} on:context />
            {/each}
          </div>
        {/each}
      {:else}
        <div class="sq">
          {#each g.items as f (f.id)}
            <GridItem file={f} size={itemSize} on:context />
          {/each}
        </div>
      {/if}
    {/each}
  {:else if $thumbShape === 'original' && rowsAll}
    {#each rowsAll as row, ri (ri)}
      <div class="jrow">
        {#each row as it (it.file.id)}
          <GridItem file={it.file} size={it.h} on:context />
        {/each}
      </div>
    {/each}
  {:else}
    <div class="sq">
      {#each files as f (f.id)}
        <GridItem file={f} size={itemSize} on:context />
      {/each}
    </div>
  {/if}
</div>

<style>
  .timeline { display: flex; flex-direction: column; overflow-y: auto; flex: 1;
    content-visibility: auto; padding: 0 8px 14px; }
  .sq { display: grid; grid-template-columns: repeat(auto-fill, var(--size));
    gap: 4px; padding-bottom: 10px; }
  .jrow { display: flex; gap: 4px; margin-bottom: 4px; }
  .month { position: sticky; top: 0; z-index: 2;
    background: var(--bg); margin: 0; padding: 14px 2px 8px;
    font-size: 13px; font-weight: 600; color: var(--fg);
    backdrop-filter: blur(6px); }
  .month .count { color: var(--fg-dim); font-weight: 400; margin-left: 4px; font-size: 12px; }
</style>
