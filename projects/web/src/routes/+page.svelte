<script lang="ts">
	import { onMount } from 'svelte';
	import EntryModal from '$lib/components/EntryModal.svelte';
	import type { LexiconEntryDetail } from '$lib/server/queries';

	let { data } = $props();

	const PAGE_SIZE = 100;
	let query = $state('');
	let visibleCount = $state(PAGE_SIZE);
	let loadingEntry = $state(false);

	let selectedEntry = $state<LexiconEntryDetail | null>(null);
	let selectedEntryId = $state<number | null>(null);

	// Client-side filtering: all entries are pre-loaded from build time
	let filtered = $derived.by(() => {
		const q = query.trim().toLowerCase();
		if (!q) return data.entries;
		// Prefix match first, then substring fallback
		const prefix = data.entries.filter((e) => e.key.toLowerCase().startsWith(q));
		if (prefix.length > 0) return prefix;
		return data.entries.filter((e) => e.key.toLowerCase().includes(q));
	});

	let displayEntries = $derived(filtered.slice(0, visibleCount));
	let hasMore = $derived(visibleCount < filtered.length);

	// Reset visible count when query changes
	$effect(() => {
		query;
		visibleCount = PAGE_SIZE;
	});

	let searchInput: HTMLInputElement;
	let sentinel: HTMLDivElement;

	onMount(() => {
		searchInput?.focus();

		const observer = new IntersectionObserver(
			(items) => {
				if (items[0].isIntersecting && hasMore) {
					visibleCount += PAGE_SIZE;
				}
			},
			{ rootMargin: '200px' }
		);

		if (sentinel) observer.observe(sentinel);
		return () => observer.disconnect();
	});

	function clearSearch() {
		query = '';
		searchInput?.focus();
	}

	async function openEntry(id: number) {
		selectedEntryId = id;
		loadingEntry = true;
		try {
			const res = await fetch(`/api/lexicon/entry/${id}`);
			if (res.ok) {
				selectedEntry = await res.json();
			}
		} finally {
			loadingEntry = false;
		}
	}

	function closeEntry() {
		selectedEntry = null;
		selectedEntryId = null;
	}
</script>

<svelte:head>
	<title>Lexicon &mdash; Bardbase</title>
</svelte:head>

<div class="lexicon-page">
	<div class="sticky-header">
		<header class="page-header">
			<h1 class="page-title">Lexicon</h1>
		</header>

		<div class="search-bar">
		<svg class="search-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
			<circle cx="11" cy="11" r="8" />
			<line x1="21" y1="21" x2="16.65" y2="16.65" />
		</svg>
		<input
			bind:this={searchInput}
			type="text"
			class="search-input"
			placeholder="Search words..."
			bind:value={query}
			aria-label="Search lexicon entries"
		/>
		{#if query}
			<button class="clear-btn" onclick={clearSearch} aria-label="Clear search">
				<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
					<line x1="18" y1="6" x2="6" y2="18" />
					<line x1="6" y1="6" x2="18" y2="18" />
				</svg>
			</button>
		{/if}
		</div>
	</div>

	{#if query && filtered.length === 0}
		<div class="status-text">No entries found for "{query}"</div>
	{:else}
		<div class="result-count">
			{#if query}
				{filtered.length} results
			{:else}
				{data.entries.length} entries
			{/if}
		</div>
	{/if}

	<ul class="entry-list" role="list" aria-label="Lexicon entries">
		{#each displayEntries as entry (entry.id)}
			<li>
				<button
					class="entry-item"
					class:selected={selectedEntryId === entry.id}
					onclick={() => openEntry(entry.id)}
					aria-label="View definition of {entry.key}"
				>
					<span class="entry-key">{entry.key}</span>
				</button>
			</li>
		{/each}
	</ul>

	{#if hasMore}
		<div class="status-text">Loading&hellip;</div>
	{/if}

	<!-- Infinite scroll sentinel -->
	<div bind:this={sentinel} class="scroll-sentinel"></div>
</div>

<EntryModal entry={selectedEntry} onclose={closeEntry} />

<style>
	.lexicon-page {
		max-width: 720px;
		margin: 0 auto;
		padding: 0 16px;
	}

	.sticky-header {
		position: sticky;
		top: 0;
		z-index: 50;
		background: var(--color-bg);
		padding-bottom: 8px;
	}

	.page-header {
		padding: 12px 0 8px;
	}

	.page-title {
		margin: 0;
		font-size: 1.3rem;
		font-weight: 700;
		color: var(--color-text);
	}

	/* ─── Search Bar ─── */
	.search-bar {
		position: relative;
	}

	.search-icon {
		position: absolute;
		left: 14px;
		top: 50%;
		transform: translateY(-50%);
		color: var(--color-text-muted);
		pointer-events: none;
	}

	.search-input {
		width: 100%;
		padding: 12px 40px 12px 42px;
		border: 1px solid var(--color-border);
		background: var(--color-surface);
		color: var(--color-text);
		font-family: inherit;
		font-size: 1rem;
		border-radius: 12px;
		outline: none;
		transition: border-color 0.15s, box-shadow 0.15s;
		box-sizing: border-box;
	}

	.search-input::placeholder {
		color: var(--color-text-muted);
	}

	.search-input:focus {
		border-color: var(--color-accent);
		box-shadow: 0 0 0 3px rgba(77, 182, 172, 0.15);
	}

	.clear-btn {
		position: absolute;
		right: 10px;
		top: 50%;
		transform: translateY(-50%);
		background: none;
		border: none;
		color: var(--color-text-muted);
		cursor: pointer;
		padding: 4px;
		border-radius: 4px;
		display: flex;
		align-items: center;
	}

	.clear-btn:hover {
		color: var(--color-text);
		background: var(--color-hover);
	}

	/* ─── Status ─── */
	.status-text {
		padding: 12px 0;
		font-size: 0.85rem;
		color: var(--color-text-muted);
		text-align: center;
	}

	.result-count {
		padding: 4px 0 8px;
		font-size: 0.8rem;
		color: var(--color-text-muted);
	}

	/* ─── Entry List ─── */
	.entry-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}

	.entry-item {
		display: block;
		width: 100%;
		padding: 14px 16px;
		border: none;
		background: none;
		text-align: left;
		cursor: pointer;
		font-family: inherit;
		border-radius: 8px;
		transition: background 0.15s;
		border-bottom: 1px solid var(--color-border);
		color: var(--color-text);
	}

	.entry-item:hover {
		background: var(--color-hover);
	}

	.entry-item:active {
		background: var(--color-active);
	}

	.entry-item.selected {
		background: var(--color-active);
		border-color: var(--color-accent);
	}

	.entry-key {
		font-size: 1.05rem;
		font-weight: 500;
	}

	.scroll-sentinel {
		height: 1px;
		/* pad below list so footer doesn't cover last entries */
		margin-bottom: 48px;
	}
</style>
