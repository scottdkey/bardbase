<script lang="ts">
	import { onMount } from 'svelte';
	import { theme } from '$lib/stores/theme.svelte';

	let { children } = $props();

	onMount(() => {
		theme.init();
	});
</script>

<svelte:head>
	<title>Bardbase</title>
</svelte:head>

<div class="app">
	<header class="top-bar">
		<a href="/" class="logo">Bardbase</a>
		<button
			class="theme-toggle"
			onclick={() => theme.toggle()}
			aria-label={theme.isDark ? 'Switch to light mode' : 'Switch to dark mode'}
		>
			{#if theme.isDark}
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<circle cx="12" cy="12" r="5" />
					<line x1="12" y1="1" x2="12" y2="3" />
					<line x1="12" y1="21" x2="12" y2="23" />
					<line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
					<line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
					<line x1="1" y1="12" x2="3" y2="12" />
					<line x1="21" y1="12" x2="23" y2="12" />
					<line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
					<line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
				</svg>
			{:else}
				<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
				</svg>
			{/if}
		</button>
	</header>

	<div class="content">
		{@render children()}
	</div>
</div>

<style>
	:global(*) {
		box-sizing: border-box;
	}

	:global(html) {
		--top-bar-height: 48px;
	}

	/* ─── Light theme ─── */
	:global([data-theme='light']) {
		--color-bg: #FAF7F2;
		--color-surface: #F0EBE1;
		--color-elevated: #FFFFFF;
		--color-text: #1A1209;
		--color-text-secondary: #5C4A37;
		--color-text-muted: #8A7968;
		--color-accent: #7B5B3A;
		--color-accent-hover: #634A2E;
		--color-border: #DDD5C7;
		--color-overlay: rgba(26, 18, 9, 0.4);
		--color-hover: rgba(123, 91, 58, 0.08);
		--color-active: rgba(123, 91, 58, 0.14);
		--color-focus: #7B5B3A;
		color-scheme: light;
	}

	/* ─── Dark theme: deeper, higher contrast ─── */
	:global([data-theme='dark']) {
		--color-bg: #080E10;
		--color-surface: #0E1519;
		--color-elevated: #141E24;
		--color-text: #E8F0EC;
		--color-text-secondary: #A0C4B0;
		--color-text-muted: #607A6E;
		--color-accent: #6DDAD0;
		--color-accent-hover: #8EECE4;
		--color-border: #1A2A30;
		--color-overlay: rgba(0, 0, 0, 0.65);
		--color-hover: rgba(109, 218, 208, 0.07);
		--color-active: rgba(109, 218, 208, 0.13);
		--color-focus: #6DDAD0;
		color-scheme: dark;
	}

	:global(body) {
		margin: 0;
		font-family: 'Georgia', 'Times New Roman', 'Noto Serif', serif;
		background: var(--color-bg);
		color: var(--color-text);
		line-height: 1.6;
		-webkit-font-smoothing: antialiased;
	}

	:global(a) {
		color: var(--color-accent);
		text-decoration: none;
	}

	:global(a:hover) {
		color: var(--color-accent-hover);
	}

	:global(:focus-visible) {
		outline: 2px solid var(--color-focus);
		outline-offset: 2px;
	}

	@media (prefers-reduced-motion: reduce) {
		:global(*) {
			transition-duration: 0.01ms !important;
			animation-duration: 0.01ms !important;
		}
	}

	.app {
		min-height: 100dvh;
		display: flex;
		flex-direction: column;
	}

	/* ─── Top Bar ─── */
	.top-bar {
		position: sticky;
		top: 0;
		z-index: 100;
		height: var(--top-bar-height);
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 16px;
		background: var(--color-bg);
		border-bottom: 1px solid var(--color-border);
	}

	.logo {
		font-size: 1.1rem;
		font-weight: 700;
		color: var(--color-text) !important;
		letter-spacing: -0.01em;
	}

	.theme-toggle {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 36px;
		height: 36px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 8px;
		transition: color 0.15s, background 0.15s;
	}

	.theme-toggle:hover {
		color: var(--color-text);
		background: var(--color-hover);
	}

	.content {
		flex: 1;
	}
</style>
