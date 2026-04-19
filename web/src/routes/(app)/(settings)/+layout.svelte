<!-- (app)/(settings)/+layout.svelte — shared shell for /settings, /admin, /shares -->
<script lang="ts">
  import { page } from '$app/stores';
  import { me } from '$lib/stores';

  $: activePath = $page.url.pathname;
  $: tabs = [
    { href: '/settings', label: 'Settings' },
    { href: '/shares', label: 'Shares' },
    ...($me?.is_admin ? [{ href: '/admin', label: 'Admin' }] : []),
  ];
</script>

<div class="settings-root">
  <nav class="tabs">
    {#each tabs as t}
      <a class="tab" class:active={activePath === t.href} href={t.href}>{t.label}</a>
    {/each}
  </nav>
  <div class="content">
    <slot />
  </div>
</div>

<style>
  .settings-root { display: flex; flex-direction: column; flex: 1; min-height: 0; }
  .tabs { display: flex; gap: 2px; border-bottom: 1px solid var(--border);
    padding: 0 16px; background: var(--bg); }
  .tab { padding: 12px 16px; color: var(--fg-dim); text-decoration: none;
    border-bottom: 2px solid transparent; font-size: 14px;
    margin-bottom: -1px; }
  .tab:hover { color: var(--fg); }
  .tab.active { color: var(--fg); border-bottom-color: var(--accent); }
  .content { flex: 1; min-height: 0; overflow-y: auto; }
</style>
