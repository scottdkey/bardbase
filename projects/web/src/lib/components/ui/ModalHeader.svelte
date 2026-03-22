<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		children
	}: {
		children: Snippet;
	} = $props();
</script>

<div class="modal-header">
	{@render children()}
</div>

<style>
	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 16px 20px 0;
		flex-shrink: 0;
		gap: 12px;
	}
</style>
