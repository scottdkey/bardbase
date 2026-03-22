<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		onclick,
		label,
		size = 32,
		title: btnTitle,
		variant = 'default',
		opacity,
		children
	}: {
		onclick?: (e: MouseEvent) => void;
		label: string;
		size?: number;
		title?: string;
		variant?: 'default' | 'danger';
		opacity?: number;
		children: Snippet;
	} = $props();
</script>

<button
	class="icon-btn"
	class:danger={variant === 'danger'}
	style="width: {size}px; height: {size}px;{opacity != null ? ` opacity: ${opacity}` : ''}"
	{onclick}
	aria-label={label}
	title={btnTitle}
>
	{@render children()}
</button>

<style>
	.icon-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		border: none;
		background: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 6px;
		flex-shrink: 0;
		padding: 0;
		transition: color 0.15s, background 0.15s;
	}

	.icon-btn:hover {
		color: var(--color-text);
		background: var(--color-hover);
	}

	.icon-btn.danger:hover {
		color: var(--color-danger);
		background: rgba(232, 85, 53, 0.1);
	}
</style>
