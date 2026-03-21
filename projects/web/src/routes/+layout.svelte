<script lang="ts">
	import { onMount } from 'svelte';
	import { theme } from '$lib/stores/theme.svelte';

	let { children } = $props();
	let menuOpen = $state(false);

	onMount(() => {
		theme.init();
	});

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && menuOpen) {
			menuOpen = false;
		}
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<svelte:head>
	<title>Bardbase</title>
</svelte:head>

<div class="app">
	<div class="content">
		{@render children()}
	</div>

	<nav class="bottom-nav" aria-label="Main navigation">
		<a href="/" class="nav-item active" aria-current="page">
			<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
				<path d="M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20" />
			</svg>
			<span>Lexicon</span>
		</a>
		<button
			class="nav-item"
			onclick={() => (menuOpen = !menuOpen)}
			aria-expanded={menuOpen}
			aria-controls="app-menu"
		>
			<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
				<circle cx="12" cy="12" r="3" />
				<path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
			</svg>
			<span>Settings</span>
		</button>
	</nav>

	{#if menuOpen}
		<div
			class="menu-backdrop"
			onclick={() => (menuOpen = false)}
			onkeydown={(e) => e.key === 'Enter' && (menuOpen = false)}
			role="button"
			tabindex="-1"
			aria-label="Close menu"
		></div>
		<div class="menu-panel" id="app-menu" role="dialog" aria-label="Settings menu">
			<div class="menu-header">
				<h2>Settings</h2>
			</div>
			<button class="menu-item" onclick={() => theme.toggle()}>
				{#if theme.isDark}
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
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
					<span>Switch to Light Mode</span>
				{:else}
					<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
						<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
					</svg>
					<span>Switch to Dark Mode</span>
				{/if}
			</button>
		</div>
	{/if}
</div>

<style>
	:global(*) {
		box-sizing: border-box;
	}

	:global(html) {
		--nav-height: 56px;
	}

	/* ─── Light theme: tans & browns ─── */
	:global([data-theme='light']) {
		--color-bg: #F5F0E8;
		--color-surface: #EDE4D3;
		--color-elevated: #E3D8C4;
		--color-text: #2C1810;
		--color-text-secondary: #5C4033;
		--color-text-muted: #8A7560;
		--color-accent: #7B5B3A;
		--color-accent-hover: #634A2E;
		--color-border: #D1C4A9;
		--color-nav: #E3D8C4;
		--color-overlay: rgba(44, 24, 16, 0.4);
		--color-hover: rgba(123, 91, 58, 0.1);
		--color-active: rgba(123, 91, 58, 0.18);
		--color-focus: #7B5B3A;
		color-scheme: light;
	}

	/* ─── Dark theme: greens & teals ─── */
	:global([data-theme='dark']) {
		--color-bg: #0A1A1F;
		--color-surface: #0F2429;
		--color-elevated: #152E34;
		--color-text: #E0EFE6;
		--color-text-secondary: #8EC5A8;
		--color-text-muted: #5A8A70;
		--color-accent: #4DB6AC;
		--color-accent-hover: #6DCEC5;
		--color-border: #1C3E45;
		--color-nav: #0C1E23;
		--color-overlay: rgba(10, 26, 31, 0.6);
		--color-hover: rgba(77, 182, 172, 0.08);
		--color-active: rgba(77, 182, 172, 0.15);
		--color-focus: #4DB6AC;
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

	.content {
		flex: 1;
		padding-bottom: var(--nav-height);
	}

	/* ─── Bottom Nav ─── */
	.bottom-nav {
		position: fixed;
		bottom: 0;
		left: 0;
		right: 0;
		height: var(--nav-height);
		background: var(--color-nav);
		border-top: 1px solid var(--color-border);
		display: flex;
		align-items: center;
		justify-content: space-around;
		z-index: 100;
		padding-bottom: env(safe-area-inset-bottom, 0);
	}

	.nav-item {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 2px;
		padding: 6px 16px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		font-family: inherit;
		font-size: 0.7rem;
		cursor: pointer;
		border-radius: 8px;
		transition: color 0.15s, background 0.15s;
		text-decoration: none;
		letter-spacing: 0.02em;
	}

	.nav-item:hover {
		color: var(--color-text);
		background: var(--color-hover);
	}

	.nav-item.active {
		color: var(--color-accent);
	}

	/* ─── Menu ─── */
	.menu-backdrop {
		position: fixed;
		inset: 0;
		background: var(--color-overlay);
		z-index: 200;
	}

	.menu-panel {
		position: fixed;
		bottom: calc(var(--nav-height) + env(safe-area-inset-bottom, 0));
		left: 8px;
		right: 8px;
		background: var(--color-elevated);
		border: 1px solid var(--color-border);
		border-radius: 12px;
		z-index: 300;
		padding: 8px;
		max-width: 360px;
		margin: 0 auto;
	}

	.menu-header {
		padding: 8px 12px 4px;
	}

	.menu-header h2 {
		margin: 0;
		font-size: 0.85rem;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.menu-item {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		padding: 12px;
		border: none;
		background: none;
		color: var(--color-text);
		font-family: inherit;
		font-size: 0.95rem;
		cursor: pointer;
		border-radius: 8px;
		transition: background 0.15s;
		text-align: left;
	}

	.menu-item:hover {
		background: var(--color-hover);
	}

	.menu-item:active {
		background: var(--color-active);
	}
</style>
