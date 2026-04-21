<script lang="ts">
	import { onMount } from 'svelte';
	import { theme } from '$lib/stores/theme.svelte';
	import { sidebar } from '$lib/stores/sidebar.svelte';
	import type { LayoutProps } from './$types';
	import Sidebar from '$lib/components/Sidebar.svelte';
	import IconClose from '$lib/components/icons/IconClose.svelte';
	import IconSun from '$lib/components/icons/IconSun.svelte';
	import IconMoon from '$lib/components/icons/IconMoon.svelte';
	import IconButton from '$lib/components/ui/IconButton.svelte';

	let { children, data }: LayoutProps = $props();
	let legalOpen = $state(false);

	onMount(() => {
		theme.init();
		checkServerVersion();
	});

	async function checkServerVersion() {
		try {
			const res = await fetch('/api/version');
			if (!res.ok) return;
			const { build } = await res.json();
			const stored = localStorage.getItem('bardbase-server-version');
			if (stored && stored !== build) {
				if ('caches' in window) {
					await caches.delete('bardbase-meta');
					await caches.delete('bardbase-data');
					await caches.delete('bardbase-search');
				}
			}
			localStorage.setItem('bardbase-server-version', build);
		} catch {
			// best-effort
		}
	}
</script>

<svelte:head>
	<title>Bardbase</title>
</svelte:head>

<!-- Sidebar overlay (mobile) -->
{#if sidebar.open}
	<div
		class="sidebar-overlay"
		role="presentation"
		onclick={() => sidebar.close()}
		aria-hidden="true"
	></div>
{/if}

<!-- Left sidebar -->
<aside
	class="sidebar-panel"
	class:open={sidebar.open}
	aria-label="Navigation sidebar"
	aria-hidden={!sidebar.open}
>
	<div class="sidebar-header">
		<span class="sidebar-wordmark">Bardbase</span>
		<div class="sidebar-header-actions">
			<IconButton
				onclick={() => theme.toggle()}
				label={theme.isDark ? 'Switch to light mode' : 'Switch to dark mode'}
				size={28}
			>
				{#if theme.isDark}
					<IconSun size={14} />
				{:else}
					<IconMoon size={14} />
				{/if}
			</IconButton>
			<IconButton onclick={() => sidebar.close()} label="Close navigation" size={28}>
				<IconClose size={14} />
			</IconButton>
		</div>
	</div>

	<Sidebar works={data.works} onclose={() => sidebar.close()} />

	<div class="sidebar-footer">
		<button class="legal-toggle" onclick={() => legalOpen = !legalOpen}>
			{legalOpen ? 'Close' : 'Legal & Attribution'}
		</button>
		{#if legalOpen}
			<div class="legal-body">
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
		{/if}
	</div>
</aside>

<!-- Hamburger toggle (always visible) -->
{#if !sidebar.open}
	<button
		class="sidebar-toggle"
		onclick={() => sidebar.toggle()}
		aria-label="Open navigation"
		aria-expanded={sidebar.open}
	>
		<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
			<line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="18" x2="21" y2="18"/>
		</svg>
	</button>
{/if}

<!-- Main content -->
<div class="app-content" class:sidebar-open={sidebar.open}>
	{@render children()}
</div>

<style>
	:global(*) {
		box-sizing: border-box;
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

	/* ─── Dark theme ─── */
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
		--hamburger-clearance: 56px;
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

	/* ─── Sidebar panel ─── */
	.sidebar-panel {
		position: fixed;
		top: 0;
		left: 0;
		width: clamp(375px, 30vw, 600px);
		height: 100dvh;
		background: var(--color-surface);
		border-right: 1px solid var(--color-border);
		z-index: 300;
		display: flex;
		flex-direction: column;
		transform: translateX(-100%);
		transition: transform 0.22s ease;
		overflow: hidden;
	}

	.sidebar-panel.open {
		transform: translateX(0);
	}

	.sidebar-overlay {
		position: fixed;
		inset: 0;
		background: var(--color-overlay);
		z-index: 299;
	}

	.sidebar-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 10px 10px 10px 14px;
		border-bottom: 1px solid var(--color-border);
		flex-shrink: 0;
	}

	.sidebar-wordmark {
		font-size: 0.9rem;
		font-weight: 700;
		color: var(--color-text);
		letter-spacing: 0.02em;
	}

	.sidebar-header-actions {
		display: flex;
		align-items: center;
		gap: 2px;
	}

	.sidebar-footer {
		border-top: 1px solid var(--color-border);
		flex-shrink: 0;
	}

	.legal-toggle {
		display: block;
		width: 100%;
		padding: 10px 14px;
		background: none;
		border: none;
		color: var(--color-text-muted);
		font-family: inherit;
		font-size: 0.7rem;
		font-weight: 600;
		cursor: pointer;
		text-align: left;
		transition: color 0.15s;
	}

	.legal-toggle:hover {
		color: var(--color-text);
	}

	.legal-body {
		max-height: 40dvh;
		overflow-y: auto;
		padding: 8px 14px 14px;
		border-top: 1px solid var(--color-border);
	}

	.attribution-section {
		margin-bottom: 10px;
		padding-bottom: 10px;
		border-bottom: 1px solid var(--color-border);
	}

	.attribution-section:last-child {
		margin-bottom: 0;
		padding-bottom: 0;
		border-bottom: none;
	}

	.attribution-section h4 {
		margin: 0 0 3px;
		font-size: 0.65rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.attribution-section p {
		margin: 0 0 3px;
		font-size: 0.62rem;
		color: var(--color-text-secondary);
		line-height: 1.5;
	}

	.license-notice {
		font-size: 0.58rem;
		color: var(--color-text-muted);
	}

	/* ─── Hamburger toggle ─── */
	.sidebar-toggle {
		position: fixed;
		top: 10px;
		left: 10px;
		z-index: 298;
		width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 8px;
		color: var(--color-text-secondary);
		cursor: pointer;
		transition: background 0.15s, color 0.15s;
	}

	.sidebar-toggle:hover {
		background: var(--color-hover);
		color: var(--color-text);
	}

	/* ─── Main content ─── */
	.app-content {
		min-height: 100dvh;
		padding-top: var(--hamburger-clearance);
		transition: margin-left 0.22s ease;
	}

	@media (min-width: 900px) {
		.sidebar-panel {
			transform: translateX(-100%);
		}

		.sidebar-panel.open {
			transform: translateX(0);
		}

		.sidebar-overlay {
			display: none;
		}

		.app-content.sidebar-open {
			margin-left: clamp(375px, 30vw, 600px);
		}
	}
</style>
