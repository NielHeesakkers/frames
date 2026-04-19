<!-- web/src/routes/browse/+layout.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { refreshMe, me, logout } from '$lib/stores';
  import FolderTree from '$lib/components/FolderTree.svelte';
  import Breadcrumb from '$lib/components/Breadcrumb.svelte';
  import SearchBox from '$lib/components/SearchBox.svelte';

  onMount(async () => {
    const u = await refreshMe();
    if (!u) goto('/login');
  });
</script>

{#if $me}
  <div class="shell">
    <aside>
      <div class="brand">Frames</div>
      <FolderTree />
      <div class="sidebar-footer">
        <span>{$me.username}</span>
        <a href="/shares">Shares</a>
        <a href="/settings">Settings</a>
        {#if $me?.is_admin}<a href="/admin">Admin</a>{/if}
        <button on:click={async () => { await logout(); goto('/login'); }}>Logout</button>
      </div>
    </aside>
    <main>
      <header>
        <Breadcrumb />
        <SearchBox />
      </header>
      <div class="main-inner">
        <slot />
      </div>
    </main>
  </div>
{/if}

<style>
  .shell { display: grid; grid-template-columns: 280px 1fr; height: 100vh; }
  aside { background: var(--bg-2); border-right: 1px solid var(--border);
    display: flex; flex-direction: column; min-height: 0; }
  .brand { padding: 14px 16px; font-weight: 600; border-bottom: 1px solid var(--border); }
  .sidebar-footer { border-top: 1px solid var(--border); padding: 10px;
    display: flex; justify-content: space-between; align-items: center; gap: 8px; color: var(--fg-dim); }
  main { display: flex; flex-direction: column; min-height: 0; }
  header { border-bottom: 1px solid var(--border); padding: 10px 16px; }
  .main-inner { flex: 1; overflow: hidden; display: flex; flex-direction: column; min-height: 0; }
  @media (max-width: 768px) {
    .shell { grid-template-columns: 1fr; }
    aside { display: none; }
  }
</style>
