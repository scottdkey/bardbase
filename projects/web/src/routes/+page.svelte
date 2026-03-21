<script lang="ts">
	import { onMount } from 'svelte';
	import EntryModal from '$lib/components/EntryModal.svelte';
	import type { LexiconEntryDetail } from '$lib/server/queries';

	let { data } = $props();

	let query = $state('');
	let entries = $state([] as typeof data.entries);
	let loading = $state(false);
	let loadingMore = $state(false);
	let loadingEntry = $state(false);
	let hasMore = $state(true);
	let searchTimeout: ReturnType<typeof setTimeout> | null = null;

	// Initialize with first page of entries
	$effect(() => {
		if (query === '') {
			entries = data.entries.slice();
			hasMore = data.entries.length >= data.pageSize;
		}
	});

	let selectedEntry = $state<LexiconEntryDetail | null>(null);
	let selectedEntryId = $state<number | null>(null);

	let searchInput: HTMLInputElement;
	let sentinel: HTMLDivElement;

	onMount(() => {
		searchInput?.focus();

		// IntersectionObserver for infinite scroll
		const observer = new IntersectionObserver(
			(entries) => {
				if (entries[0].isIntersecting && hasMore && !loading && !loadingMore && query === '') {
					loadMore();
				}
			},
			{ rootMargin: '200px' }
		);

		if (sentinel) observer.observe(sentinel);
		return () => observer.disconnect();
	});

	async function loadMore() {
		if (loadingMore || !hasMore || query !== '') return;
		loadingMore = true;
		try {
			const res = await fetch(`/api/lexicon/entries?offset=${entries.length}&limit=${data.pageSize}`);
			const json = await res.json();
			if (json.entries.length > 0) {
				entries = [...entries, ...json.entries];
			}
			hasMore = json.hasMore;
		} finally {
			loadingMore = false;
		}
	}

	function handleSearch(e: Event) {
		const value = (e.target as HTMLInputElement).value;
		query = value;

		if (searchTimeout) clearTimeout(searchTimeout);

		if (value.trim() === '') {
			entries = data.entries.slice();
			hasMore = data.entries.length >= data.pageSize;
			return;
		}

		hasMore = false;
		searchTimeout = setTimeout(async () => {
			loading = true;
			try {
				const res = await fetch(`/api/lexicon/search?q=${encodeURIComponent(value.trim())}&limit=100`);
				const json = await res.json();
				entries = json.entries;
			} finally {
				loading = false;
			}
		}, 150);
	}

	function clearSearch() {
		query = '';
		entries = data.entries.slice();
		hasMore = data.entries.length >= data.pageSize;
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
			value={query}
			oninput={handleSearch}
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

	{#if loading}
		<div class="status-text">Searching&hellip;</div>
	{:else if query && entries.length === 0}
		<div class="status-text">No entries found for "{query}"</div>
	{:else if entries.length > 0}
		<div class="result-count">
			{#if query}
				{entries.length} results
			{:else}
				{entries.length} of {data.total} entries
			{/if}
		</div>
	{/if}

	<ul class="entry-list" role="list" aria-label="Lexicon entries">
		{#each entries as entry (entry.id)}
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

	{#if loadingMore}
		<div class="status-text">Loading more&hellip;</div>
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
