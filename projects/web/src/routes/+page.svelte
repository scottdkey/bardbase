<script lang="ts">
	import { onMount } from 'svelte';
	import EntryModal from '$lib/components/EntryModal.svelte';
	import PageHeader from '$lib/components/ui/PageHeader.svelte';
	import SearchInput from '$lib/components/ui/SearchInput.svelte';
	import type { LexiconEntryDetail } from '$lib/server/queries';

	let { data } = $props();

	const PAGE_SIZE = 100;
	let query = $state('');
	let visibleCount = $state(PAGE_SIZE);

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
		void query;
		visibleCount = PAGE_SIZE;
	});

	let sentinel: HTMLDivElement;

	onMount(() => {
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

	async function openEntry(id: number) {
		selectedEntryId = id;
		try {
			const res = await fetch(`/api/lexicon/entry/${id}`);
			if (res.ok) {
				selectedEntry = await res.json();
			}
		} finally {
			// loading done
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
		<PageHeader title="Bardbase" />

		<div class="search-bar">
			<SearchInput bind:value={query} placeholder="Search entries..." autofocus />
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

	/* ─── Search Bar ─── */
	.search-bar {
		position: relative;
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
