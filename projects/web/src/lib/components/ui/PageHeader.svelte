<script lang="ts">
	import type { Snippet } from 'svelte';

	let {
		title,
		subtitle,
		children
	}: {
		title: string;
		subtitle?: string;
		children?: Snippet;
	} = $props();
</script>

<header class="page-header">
	<div class="header-text">
		<h1 class="page-title">{title}</h1>
		{#if subtitle}
			<p class="page-subtitle">{subtitle}</p>
		{/if}
	</div>
	{#if children}
		{@render children()}
	{/if}
</header>

<style>
	.page-header {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		padding: 16px 0 12px;
	}

	.header-text {
		flex: 1;
	}

	.page-title {
		margin: 0;
		font-size: 1.3rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.page-subtitle {
		margin: 2px 0 0;
		font-size: 0.85rem;
		color: var(--color-text-muted);
	}
</style>
