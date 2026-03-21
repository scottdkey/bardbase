<script lang="ts">
	import { onMount } from 'svelte';
	import { theme } from '$lib/stores/theme.svelte';
	import { corrections } from '$lib/stores/corrections.svelte';
	import type { LayoutProps } from './$types';

	let { children, data }: LayoutProps = $props();
	let footerOpen = $state(false);

	onMount(() => {
		theme.init();
	});
</script>

<svelte:head>
	<title>Bardbase</title>
</svelte:head>

<div class="app">
	<!-- <header class="top-bar">
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
	</header> -->

	<div class="content">
		{@render children()}
	</div>

	<!-- Fixed footer -->
	<footer class="site-footer" class:drawer-open={footerOpen}>
		{#if footerOpen}
			<div class="footer-drawer">
				<div class="drawer-header">
					<h3 class="drawer-title">Acknowledgements & Legal</h3>
					<button class="drawer-close" onclick={() => footerOpen = false} aria-label="Close">
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<line x1="18" y1="6" x2="6" y2="18" />
							<line x1="6" y1="6" x2="18" y2="18" />
						</svg>
					</button>
				</div>
				<div class="drawer-body">
					{#each data.attributions as attr}
						<section class="attribution-section">
							<h4>{attr.source_name}</h4>
							<p>{@html attr.attribution_html}</p>
							{#if attr.license_notice_text}
								<p class="license-notice">{attr.license_notice_text}</p>
							{/if}
						</section>
					{/each}

					<section class="attribution-section">
						<h4>License</h4>
						<p>
							The compiled database is released under <strong>CC BY-SA 4.0</strong> due to Perseus content.
							Build tooling is released under the <strong>MIT License</strong>.
						</p>
					</section>
				</div>
			</div>
		{/if}
		<div class="footer-bar">
			<nav class="footer-nav">
				<a href="/" class="nav-link">Lexicon</a>
				<a href="/editions" class="nav-link">Editions</a>
				<a href="/glossary" class="nav-link">Glossary</a>
				<a href="/corrections" class="nav-link">
					Corrections
					{#if corrections.pendingCount > 0}
						<span class="corrections-badge">{corrections.pendingCount}</span>
					{/if}
				</a>
			</nav>
			<div class="footer-actions">
				<button class="footer-toggle" onclick={() => footerOpen = !footerOpen}>
					{footerOpen ? 'Close' : 'Legal'}
				</button>
				<button
					class="theme-toggle"
					onclick={() => theme.toggle()}
					aria-label={theme.isDark ? 'Switch to light mode' : 'Switch to dark mode'}
				>
					{#if theme.isDark}
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
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
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
							<path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
						</svg>
					{/if}
				</button>
			</div>
		</div>
	</footer>
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

	:global(html) {
		font-size: 150%;
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

	.theme-toggle {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 6px;
		transition: color 0.15s, background 0.15s;
	}

	.theme-toggle:hover {
		color: var(--color-text);
		background: var(--color-hover);
	}

	.content {
		flex: 1;
		padding-bottom: 48px;
	}

	/* ─── Fixed Footer ─── */
	.site-footer {
		position: fixed;
		bottom: 0;
		left: 0;
		right: 0;
		z-index: 200;
		background: var(--color-bg);
		border-top: 1px solid var(--color-border);
	}

	.footer-bar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 10px 16px;
		gap: 12px;
	}

	.footer-actions {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;
	}

	.footer-toggle {
		background: none;
		border: 1px solid var(--color-border);
		color: var(--color-text-muted);
		font-family: inherit;
		font-size: 0.75rem;
		font-weight: 600;
		padding: 4px 10px;
		border-radius: 6px;
		cursor: pointer;
		white-space: nowrap;
	}

	.footer-toggle:hover {
		color: var(--color-text);
		background: var(--color-hover);
	}

	.footer-nav {
		display: flex;
		align-items: center;
		gap: 6px;
	}

	.nav-link {
		display: flex;
		align-items: center;
		gap: 4px;
		font-size: 0.55rem;
		font-weight: 600;
		color: var(--color-text-muted) !important;
		text-decoration: none;
		padding: 4px 8px;
		border-radius: 4px;
	}

	.nav-link:hover {
		color: var(--color-text) !important;
		background: var(--color-hover);
	}

	.corrections-badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 14px;
		height: 14px;
		padding: 0 3px;
		border-radius: 7px;
		background: #e8a735;
		color: #1a1a2e;
		font-size: 0.4rem;
		font-weight: 800;
	}

	/* ─── Footer Drawer ─── */
	.footer-drawer {
		max-height: 60dvh;
		display: flex;
		flex-direction: column;
		border-bottom: 1px solid var(--color-border);
	}

	.drawer-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px 0;
		flex-shrink: 0;
	}

	.drawer-title {
		margin: 0;
		font-size: 0.9rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.drawer-close {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 28px;
		height: 28px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		cursor: pointer;
		border-radius: 6px;
	}

	.drawer-close:hover {
		background: var(--color-hover);
		color: var(--color-text);
	}

	.drawer-body {
		overflow-y: auto;
		padding: 12px 16px 16px;
	}

	.attribution-section {
		margin-bottom: 12px;
		padding-bottom: 12px;
		border-bottom: 1px solid var(--color-border);
	}

	.attribution-section:last-child {
		margin-bottom: 0;
		padding-bottom: 0;
		border-bottom: none;
	}

	.attribution-section h4 {
		margin: 0 0 4px;
		font-size: 0.75rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.attribution-section p {
		margin: 0 0 4px;
		font-size: 0.7rem;
		color: var(--color-text-secondary);
		line-height: 1.5;
	}

	.attribution-section p:last-child {
		margin-bottom: 0;
	}

	.license-notice {
		font-size: 0.65rem;
		color: var(--color-text-muted);
	}
</style>
