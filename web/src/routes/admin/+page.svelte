<!-- web/src/routes/admin/+page.svelte -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';
  import { me, refreshMe } from '$lib/stores';

  let users: any[] = [];
  let status: any = null;
  let newUser = { username: '', password: '', is_admin: false };
  let error = '';

  async function load() {
    users = await api.listUsers();
    status = await api.scanStatus();
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
  async function scanNow(full: boolean) { await api.scan(full); setTimeout(load, 1000); }

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
    <div class="inline">
      <button on:click={() => scanNow(false)}>Run incremental</button>
      <button on:click={() => scanNow(true)}>Run full</button>
    </div>
    {#if status}
      <pre>{JSON.stringify(status, null, 2)}</pre>
    {/if}
  </section>
</div>

<style>
  .page { padding: 24px; overflow-y: auto; height: 100%; }
  section { margin-bottom: 28px; }
  table { width: 100%; border-collapse: collapse; margin: 10px 0; }
  th, td { padding: 8px 10px; border-bottom: 1px solid var(--border); text-align: left; }
  .inline { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
  pre { background: var(--bg-2); padding: 12px; border-radius: 6px; overflow-x: auto; }
  .err { color: var(--danger); }
  button.danger { background: var(--danger); color: white; border-color: var(--danger); }
</style>
