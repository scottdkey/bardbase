<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		variant = 'primary',
		disabled = false,
		onclick,
		children
	}: {
		variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
		disabled?: boolean;
		onclick?: (e: MouseEvent) => void;
		children: Snippet;
	} = $props();
</script>

<button class="btn btn-{variant}" {disabled} {onclick}>
	{@render children()}
</button>

<style>
	.btn {
		padding: 6px 14px;
		border-radius: 6px;
		font-family: inherit;
		font-size: 0.75rem;
		font-weight: 600;
		cursor: pointer;
		white-space: nowrap;
		transition: background 0.15s, color 0.15s;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-primary {
		border: none;
		background: var(--color-accent);
		color: var(--color-bg);
	}

	.btn-primary:not(:disabled):hover {
		background: var(--color-accent-hover);
	}

	.btn-secondary {
		border: 1px solid var(--color-border);
		background: none;
		color: var(--color-text-muted);
	}

	.btn-secondary:hover {
		background: var(--color-hover);
		color: var(--color-text);
	}

	.btn-danger {
		border: 1px solid var(--color-border);
		background: none;
		color: var(--color-text-muted);
	}

	.btn-danger:hover {
		color: var(--color-danger);
		border-color: var(--color-danger);
	}

	.btn-ghost {
		border: 1px solid var(--color-border);
		background: none;
		color: var(--color-text-muted);
		font-size: 0.75rem;
	}

	.btn-ghost:hover {
		color: var(--color-text);
		background: var(--color-hover);
	}
</style>
