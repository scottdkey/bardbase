<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		label,
		children
	}: {
		label: string;
		children: Snippet;
	} = $props();
</script>

<label class="field">
	<span class="field-label">{label}</span>
	{@render children()}
</label>

<style>
	.field {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.field-label {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.04em;
	}
</style>
