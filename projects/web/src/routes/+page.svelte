<script lang="ts">
	import { onMount } from 'svelte';
	import EntryModal from '$lib/components/EntryModal.svelte';
	import type { LexiconEntryDetail } from '$lib/server/queries';

	let { data } = $props();

	let activeLetter = $state('A');
	let entries = $state([] as typeof data.entries);
	let offset = $state(0);
	let hasMore = $state(true);
	let loading = $state(false);
	let loadingEntry = $state(false);

	// Initialize from server data
	$effect(() => {
		entries = data.entries.slice();
		offset = data.entries.length;
		hasMore = data.entries.length === 50;
	});

	let selectedEntry = $state<LexiconEntryDetail | null>(null);
	let selectedEntryId = $state<number | null>(null);

	let sentinel: HTMLDivElement;

	onMount(() => {
		const observer = new IntersectionObserver(
			(items) => {
				if (items[0].isIntersecting && hasMore && !loading) {
					loadMore();
				}
			},
			{ rootMargin: '200px' }
		);
		observer.observe(sentinel);
		return () => observer.disconnect();
	});

	async function switchLetter(letter: string) {
		if (letter === activeLetter) return;
		activeLetter = letter;
		loading = true;
		try {
			const res = await fetch(`/api/lexicon/entries?letter=${letter}&offset=0&limit=50`);
			const json = await res.json();
			entries = json.entries;
			offset = json.entries.length;
			hasMore = json.hasMore;
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		if (loading || !hasMore) return;
		loading = true;
		try {
			const res = await fetch(
				`/api/lexicon/entries?letter=${activeLetter}&offset=${offset}&limit=50`
			);
			const json = await res.json();
			entries = [...entries, ...json.entries];
			offset += json.entries.length;
			hasMore = json.hasMore;
		} finally {
			loading = false;
		}
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
	<header class="page-header">
		<h1 class="page-title">Lexicon</h1>
		<p class="page-subtitle">Shakespeare's complete vocabulary</p>
	</header>

	<nav class="letter-bar" aria-label="Filter by letter">
		{#each data.letters as { letter, count }}
			<button
				class="letter-btn"
				class:active={letter === activeLetter}
				onclick={() => switchLetter(letter)}
				aria-label="{letter} ({count} entries)"
				aria-pressed={letter === activeLetter}
			>
				{letter}
			</button>
		{/each}
	</nav>

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

	<div bind:this={sentinel} class="sentinel" aria-hidden="true">
		{#if loading}
			<span class="loading-text">Loading&hellip;</span>
		{/if}
	</div>
</div>

<EntryModal entry={selectedEntry} onclose={closeEntry} />

<style>
	.lexicon-page {
		max-width: 720px;
		margin: 0 auto;
		padding: 0 16px;
	}

	.page-header {
		padding: 24px 0 16px;
	}

	.page-title {
		margin: 0;
		font-size: 1.6rem;
		font-weight: 700;
		color: var(--color-text);
	}

	.page-subtitle {
		margin: 4px 0 0;
		font-size: 0.85rem;
		color: var(--color-text-muted);
	}

	/* ─── Letter Bar ─── */
	.letter-bar {
		display: flex;
		gap: 4px;
		overflow-x: auto;
		padding: 0 0 12px;
		scrollbar-width: none;
		-webkit-overflow-scrolling: touch;
	}

	.letter-bar::-webkit-scrollbar {
		display: none;
	}

	.letter-btn {
		flex-shrink: 0;
		width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
		border: 1px solid transparent;
		background: none;
		color: var(--color-text-muted);
		font-family: inherit;
		font-size: 0.85rem;
		font-weight: 600;
		cursor: pointer;
		border-radius: 8px;
		transition: background 0.15s, color 0.15s, border-color 0.15s;
	}

	.letter-btn:hover {
		background: var(--color-hover);
		color: var(--color-text);
	}

	.letter-btn:active {
		background: var(--color-active);
	}

	.letter-btn.active {
		background: var(--color-accent);
		color: var(--color-bg);
		border-color: var(--color-accent);
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

	/* ─── Loading ─── */
	.sentinel {
		padding: 24px;
		text-align: center;
		min-height: 48px;
	}

	.loading-text {
		font-size: 0.85rem;
		color: var(--color-text-muted);
	}
</style>
