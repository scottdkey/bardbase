<script lang="ts">
	import { searchText, type SearchResult } from '$lib/client/search-db';
	import { browser } from '$app/environment';

	let query = $state('');
	let results = $state<SearchResult[]>([]);
	let searching = $state(false);
	let error = $state('');
	let dbReady = $state(false);

	async function handleSearch() {
		if (!query.trim()) return;
		searching = true;
		error = '';
		try {
			results = await searchText(query);
			dbReady = true;
		} catch (e) {
			error = e instanceof Error ? e.message : 'Search failed';
		} finally {
			searching = false;
		}
	}
</script>

<svelte:head>
	<title>Search — Shakespeare Database</title>
</svelte:head>

<h1>Search</h1>
<p class="note">Full-text search powered by SQLite WASM — works offline.</p>

<form onsubmit={e => { e.preventDefault(); handleSearch(); }}>
	<input
		type="search"
		bind:value={query}
		placeholder="Search Shakespeare's texts..."
		disabled={searching}
	/>
	<button type="submit" disabled={searching}>
		{searching ? 'Searching...' : 'Search'}
	</button>
</form>

{#if error}
	<p class="error">{error}</p>
{/if}

{#if results.length > 0}
	<ul class="results">
		{#each results as r}
			<li>
				<div class="result-header">
					<strong>{r.work_title}</strong>
					<span class="location">Act {r.act}, Scene {r.scene}, Line {r.line_number}</span>
				</div>
				{#if r.character_name}
					<span class="speaker">{r.character_name}:</span>
				{/if}
				<span class="content">{r.content}</span>
			</li>
		{/each}
	</ul>
{:else if dbReady && !searching}
	<p>No results found.</p>
{/if}

<style>
	form {
		display: flex;
		gap: 0.5rem;
		margin: 1rem 0;
	}
	input {
		flex: 1;
		padding: 0.75rem;
		border: 1px solid #0f3460;
		border-radius: 4px;
		background: #16213e;
		color: #e0e0e0;
		font-size: 1rem;
	}
	button {
		padding: 0.75rem 1.5rem;
		background: #e94560;
		color: white;
		border: none;
		border-radius: 4px;
		cursor: pointer;
		font-size: 1rem;
	}
	button:disabled {
		opacity: 0.6;
	}
	.note {
		color: #a0a0a0;
		font-size: 0.85rem;
	}
	.error {
		color: #e94560;
	}
	.results {
		list-style: none;
		padding: 0;
	}
	.results li {
		padding: 0.75rem 0;
		border-bottom: 1px solid #16213e;
	}
	.result-header {
		display: flex;
		justify-content: space-between;
		margin-bottom: 0.25rem;
	}
	.location {
		color: #a0a0a0;
		font-size: 0.8rem;
	}
	.speaker {
		color: #e94560;
		font-weight: bold;
	}
</style>
