<!-- $lib/components/StarRating.svelte -->
<script lang="ts">
  export let value: number = 0;
  export let onChange: ((next: number) => void) | null = null;
  export let size: number = 18;

  $: stars = [1, 2, 3, 4, 5];

  function pick(n: number) {
    if (!onChange) return;
    // Clicking the current rating clears it (toggle off).
    onChange(n === value ? 0 : n);
  }
</script>

<div class="stars" style="--s: {size}px" role="radiogroup" aria-label="Rating">
  {#each stars as n}
    <button class="star" class:filled={n <= value}
            aria-label={`${n} ${n === 1 ? 'ster' : 'sterren'}`}
            disabled={!onChange}
            on:click|stopPropagation={() => pick(n)}>★</button>
  {/each}
  {#if onChange && value > 0}
    <button class="clear" on:click|stopPropagation={() => onChange && onChange(0)} title="Rating wissen">✕</button>
  {/if}
</div>

<style>
  .stars { display: inline-flex; gap: 2px; align-items: center; }
  .star { background: none; border: none; padding: 0 2px;
    font-size: var(--s); line-height: 1; cursor: pointer;
    color: var(--fg-dim); transition: color 0.08s; }
  .star.filled { color: #f5a524; }
  .star:disabled { cursor: default; }
  .star:not(:disabled):hover,
  .star:not(:disabled):hover ~ .star { color: var(--fg-dim); }
  .star:not(:disabled):hover { color: #f5a524; }
  .clear { background: none; border: none; color: var(--fg-dim);
    font-size: 11px; cursor: pointer; padding: 0 4px; margin-left: 4px; }
  .clear:hover { color: var(--danger); }
</style>
