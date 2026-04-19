<!-- (app)/+layout.svelte — shared shell for all authenticated pages -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { refreshMe, me, logout } from '$lib/stores';
  import FolderTree from '$lib/components/FolderTree.svelte';
  import Breadcrumb from '$lib/components/Breadcrumb.svelte';
  import SearchBox from '$lib/components/SearchBox.svelte';

  let sidebarOpen = false;

  onMount(async () => {
    const u = await refreshMe();
    if (!u) goto('/login');
  });
</script>

{#if $me}
  <button class="menu-btn" on:click={() => (sidebarOpen = !sidebarOpen)}>☰</button>
  <div class="shell">
    <aside class:open={sidebarOpen}>
      <div class="brand">
        <a href="/browse">Frames</a>
      </div>
      <FolderTree />
      <div class="sidebar-footer">
        <div class="nav-links">
          <a href="/settings">⚙ Settings</a>
        </div>
        <div class="account">
          <span>{$me.username}</span>
          <button on:click={async () => { await logout(); goto('/login'); }}>Logout</button>
        </div>
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
  .brand a { color: inherit; text-decoration: none; }
  .sidebar-footer { border-top: 1px solid var(--border); padding: 10px;
    display: flex; flex-direction: column; gap: 10px; color: var(--fg-dim); }
  .nav-links { display: flex; gap: 12px; flex-wrap: wrap; }
  .nav-links a { color: var(--fg); text-decoration: none; }
  .nav-links a:hover { color: var(--accent); }
  .account { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
  main { display: flex; flex-direction: column; min-height: 0; }
  header { border-bottom: 1px solid var(--border); padding: 10px 16px;
    display: flex; align-items: center; gap: 12px; }
  .main-inner { flex: 1; overflow: hidden; display: flex; flex-direction: column; min-height: 0; }
  .menu-btn { display: none; position: fixed; top: 8px; left: 8px; z-index: 20;
    background: var(--bg-2); border: 1px solid var(--border); padding: 6px 10px; }
  @media (max-width: 768px) {
    .shell { grid-template-columns: 1fr; }
    .menu-btn { display: block; }
    aside { display: flex; position: fixed; inset: 0 40% 0 0; transform: translateX(-100%);
      transition: transform 0.2s; z-index: 15; }
    aside.open { transform: translateX(0); }
  }
</style>
