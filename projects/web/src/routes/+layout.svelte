<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { theme } from '$lib/stores/theme.svelte';
	import { corrections } from '$lib/stores/corrections.svelte';
	import type { LayoutProps } from './$types';
	import IconClose from '$lib/components/icons/IconClose.svelte';
	import IconSun from '$lib/components/icons/IconSun.svelte';
	import IconMoon from '$lib/components/icons/IconMoon.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';

	let { children, data }: LayoutProps = $props();
	let footerOpen = $state(false);

	let currentPath = $derived(page.url.pathname);

	function isActive(href: string): boolean {
		if (href === '/') return currentPath === '/';
		return currentPath.startsWith(href);
	}

	onMount(() => {
		theme.init();
	});
</script>

<svelte:head>
	<title>Bardbase</title>
</svelte:head>

<div class="app">
	<div class="content">
		{@render children()}
	</div>

	<!-- Fixed footer -->
	<footer class="site-footer" class:drawer-open={footerOpen}>
		{#if footerOpen}
			<div class="footer-drawer">
				<div class="drawer-header">
					<h3 class="drawer-title">Acknowledgements & Legal</h3>
					<IconButton onclick={() => footerOpen = false} label="Close" size={28}>
						<IconClose size={16} />
					</IconButton>
				</div>
				<div class="drawer-body">
					{#each data.attributions as attr (attr.source_name)}
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
				<a href="/" class="nav-link" class:active={isActive('/') && !isActive('/references') && !isActive('/reference') && !isActive('/corrections') && !isActive('/lexicon')}>Texts</a>
				<a href="/references" class="nav-link" class:active={isActive('/references') || isActive('/reference') || isActive('/lexicon')}>References</a>
				<a href="/corrections" class="nav-link" class:active={isActive('/corrections')}>
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
				<IconButton
					onclick={() => theme.toggle()}
					label={theme.isDark ? 'Switch to light mode' : 'Switch to dark mode'}
					size={32}
				>
					{#if theme.isDark}
						<IconSun size={16} />
					{:else}
						<IconMoon size={16} />
					{/if}
				</IconButton>
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
		--color-bg: #EDE8DF;
		--color-surface: #E3DDD3;
		--color-elevated: #F2EDE5;
		--color-text: #2C2418;
		--color-text-secondary: #4A3F30;
		--color-text-muted: #7A6E5D;
		--color-accent: #6B4D2E;
		--color-accent-hover: #523A21;
		--color-border: #CFC7B8;
		--color-overlay: rgba(26, 18, 9, 0.4);
		--color-hover: rgba(107, 77, 46, 0.10);
		--color-active: rgba(107, 77, 46, 0.18);
		--color-focus: #6B4D2E;
		--color-warning: #c48a1a;
		--color-danger: #c44428;
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
		--color-warning: #e8a735;
		--color-danger: #e85535;
		color-scheme: dark;
	}

	:global(html) {
		font-size: 128%;
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
		transition: color 0.15s;
	}

	.nav-link.active {
		color: var(--color-accent) !important;
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
		background: var(--color-warning);
		color: var(--color-bg);
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
