<!-- web/src/lib/components/ConfirmDialog.svelte -->
<script lang="ts">
  export let title = 'Confirm';
  export let message = '';
  export let confirmLabel = 'OK';
  export let danger = false;
  export let onConfirm: () => void;
  export let onCancel: () => void;
</script>

<div class="backdrop" on:click={onCancel}>
  <div class="dialog" on:click|stopPropagation>
    <h3>{title}</h3>
    <p>{message}</p>
    <slot name="body" />
    <div class="actions">
      <button on:click={onCancel}>Cancel</button>
      <button class:primary={!danger} class:danger on:click={onConfirm}>{confirmLabel}</button>
    </div>
  </div>
</div>

<style>
  .backdrop { position: fixed; inset: 0; background: rgba(0,0,0,0.6); z-index: 150;
    display: grid; place-items: center; }
  .dialog { background: var(--bg-2); border-radius: 8px; padding: 24px;
    min-width: 320px; border: 1px solid var(--border); }
  .actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px; }
  button.danger { background: var(--danger); color: white; border-color: var(--danger); }
</style>
