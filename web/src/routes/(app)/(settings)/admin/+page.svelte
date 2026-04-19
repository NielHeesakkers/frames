<!-- (app)/admin/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { me, refreshMe } from '$lib/stores';

  let users: any[] = [];
  let newUser = { username: '', password: '', is_admin: false };
  let error = '';
  let scanMsg = '';
  let stats: any = null;

  async function load() {
    users = await api.listUsers();
    try { stats = await api.adminStats(); } catch {}
  }

  function fmtBytes(n: number): string {
    if (!n) return '0 B';
    if (n < 1024) return `${n} B`;
    if (n < 1024*1024) return `${(n/1024).toFixed(1)} KB`;
    if (n < 1024*1024*1024) return `${(n/1024/1024).toFixed(1)} MB`;
    if (n < 1024*1024*1024*1024) return `${(n/1024/1024/1024).toFixed(2)} GB`;
    return `${(n/1024/1024/1024/1024).toFixed(2)} TB`;
  }
  async function add() {
    error = '';
    try {
      await api.createUser(newUser);
      newUser = { username: '', password: '', is_admin: false };
      load();
    } catch (e: any) { error = e.message; }
  }
  async function remove(id: number) {
    if (!confirm('Delete user?')) return;
    await api.deleteUser(id); load();
  }
  async function scanNow(full: boolean) {
    scanMsg = full ? 'Full scan gestart…' : 'Incremental scan gestart…';
    await api.scan(full);
    setTimeout(() => (scanMsg = ''), 4000);
  }

  let maintMsg = '';
  async function clearCache() {
    if (!confirm('Alle thumbnails en previews wissen? De scanner regenereert ze bij de volgende scan.')) return;
    maintMsg = 'Cache wissen…';
    try {
      const r = await api.clearCache();
      maintMsg = `Cache gewist (${r.removed_entries} entries). Start straks een full scan om alles opnieuw te genereren.`;
    } catch (e: any) {
      maintMsg = `Mislukt: ${e.message ?? e}`;
    }
  }
  async function resetIndex() {
    if (!confirm('ALLE mappen, bestanden, shares en cache wissen? Gebruik dit alleen als je de photos root-map hebt gewijzigd. Users en instellingen blijven bestaan.')) return;
    maintMsg = 'Reset bezig…';
    try {
      await api.resetIndex();
      maintMsg = 'Library gereset. Klik nu op "Run full scan" om opnieuw te indexeren.';
    } catch (e: any) {
      maintMsg = `Mislukt: ${e.message ?? e}`;
    }
  }

  onMount(async () => {
    let u = $me;
    if (!u) u = await refreshMe();
    if (!u) return goto('/login');
    if (!u.is_admin) return goto('/browse');
    load();
  });
</script>

<div class="page">
  <h2>Admin</h2>

  {#if stats}
    <section>
      <h3>Statistieken</h3>
      <div class="stat-grid">
        <div class="stat"><div class="num">{stats.files.toLocaleString()}</div><div class="lbl">Bestanden</div></div>
        <div class="stat"><div class="num">{stats.folders.toLocaleString()}</div><div class="lbl">Mappen</div></div>
        <div class="stat"><div class="num">{stats.rated.toLocaleString()}</div><div class="lbl">Met rating</div></div>
        <div class="stat"><div class="num">{fmtBytes(stats.photos_size)}</div><div class="lbl">Foto-volume</div></div>
        <div class="stat"><div class="num">{fmtBytes(stats.cache_size)}</div><div class="lbl">Cache op disk</div></div>
      </div>
      {#if stats.kind_counts}
        <div class="kind-row">
          {#each Object.entries(stats.kind_counts) as [kind, count]}
            <span class="pill"><strong>{kind}</strong> · {count.toLocaleString()}</span>
          {/each}
        </div>
      {/if}
      {#if stats.last_full?.FinishedAt}
        <p class="muted">Laatste full scan: {new Date(stats.last_full.FinishedAt).toLocaleString('nl-NL')}
          ({stats.last_full.FilesScanned.toLocaleString()} gescand)</p>
      {/if}
    </section>
  {/if}

  <section>
    <h3>Users</h3>
    <table>
      <thead><tr><th>User</th><th>Admin</th><th></th></tr></thead>
      <tbody>
        {#each users as u}
          <tr>
            <td>{u.username}</td>
            <td>{u.is_admin ? 'yes' : '-'}</td>
            <td><button class="danger" on:click={() => remove(u.id)}>Delete</button></td>
          </tr>
        {/each}
      </tbody>
    </table>
    <form on:submit|preventDefault={add} class="inline">
      <input placeholder="username" bind:value={newUser.username} />
      <input placeholder="password" type="password" bind:value={newUser.password} />
      <label><input type="checkbox" bind:checked={newUser.is_admin} /> admin</label>
      <button class="primary">Add user</button>
      {#if error}<span class="err">{error}</span>{/if}
    </form>
  </section>

  <section>
    <h3>Scan</h3>
    <div class="scan">
      <div class="scan-item">
        <p class="explain">Kijkt alleen naar mappen waarvan de <em>modification time</em> veranderd is. Snel, gebruik dit voor een tussentijdse update.</p>
        <button on:click={() => scanNow(false)}>Run incremental scan</button>
      </div>
      <div class="scan-item">
        <p class="explain">Loopt door de hele library en negeert de mtime-optimalisatie. Trager, maar vangt alles op — inclusief edge cases en correcties na configuratiewijzigingen.</p>
        <button on:click={() => scanNow(true)}>Run full scan</button>
      </div>
    </div>
    {#if scanMsg}<p class="ok">{scanMsg}</p>{/if}
  </section>

  <section>
    <h3>Onderhoud</h3>
    <div class="scan">
      <div class="scan-item">
        <p class="explain">Verwijdert alle gegenereerde thumbnails en previews. De mappen-index blijft intact; bij de volgende scan worden de thumbs opnieuw gemaakt.</p>
        <button on:click={clearCache}>Clear cache</button>
      </div>
      <div class="scan-item">
        <p class="explain">Wist mappen, bestanden, shares en cache volledig. Gebruik dit alleen als je de <code>/photos</code> root van de container wijzigt en opnieuw wil indexeren. Users en wachtwoord blijven bestaan.</p>
        <button class="danger" on:click={resetIndex}>Reset library</button>
      </div>
    </div>
    {#if maintMsg}<p class="ok">{maintMsg}</p>{/if}
  </section>
</div>

<style>
  .page { padding: 24px; overflow-y: auto; height: 100%; max-width: 900px; }
  section { margin-bottom: 28px; }
  table { width: 100%; border-collapse: collapse; margin: 10px 0; }
  th, td { padding: 8px 10px; border-bottom: 1px solid var(--border); text-align: left; }
  .inline { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
  .err { color: var(--danger); }
  .ok { color: #22c55e; margin-top: 8px; }
  button.danger { background: var(--danger); color: white; border-color: var(--danger); }
  .scan { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
  .scan-item { background: var(--bg-2); border: 1px solid var(--border);
    border-radius: 8px; padding: 16px; display: flex; flex-direction: column;
    gap: 10px; }
  .explain { margin: 0; color: var(--fg-dim); font-size: 13px; line-height: 1.5; }
  .scan-item button { align-self: flex-start; }
  .stat-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 12px; margin: 12px 0 10px; }
  .stat { background: var(--bg-2); border: 1px solid var(--border);
    border-radius: 8px; padding: 14px 16px; }
  .stat .num { font-size: 22px; font-weight: 600; color: var(--fg); }
  .stat .lbl { font-size: 12px; color: var(--fg-dim); text-transform: uppercase;
    letter-spacing: 0.3px; margin-top: 4px; }
  .kind-row { display: flex; gap: 8px; flex-wrap: wrap; margin: 10px 0 6px; }
  .pill { background: var(--bg-2); border: 1px solid var(--border);
    padding: 3px 10px; border-radius: 12px; font-size: 12px; color: var(--fg-dim); }
  .pill strong { color: var(--fg); margin-right: 4px; }
  .muted { color: var(--fg-dim); font-size: 12px; }
  @media (max-width: 720px) {
    .scan { grid-template-columns: 1fr; }
  }
</style>
