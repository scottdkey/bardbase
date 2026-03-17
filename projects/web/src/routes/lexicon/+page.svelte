<script lang="ts">
	let { data } = $props();
</script>

<svelte:head>
	<title>Lexicon — Variorum</title>
</svelte:head>

<h1>Shakespeare Lexicon</h1>

<nav class="letters">
	{#each data.letters as { letter, count }}
		<span class="letter-link" title="{count} entries">{letter}</span>
	{/each}
</nav>

<ul class="entries">
	{#each data.entries as entry}
		<li>
			<strong>{entry.key}</strong>
			{#if entry.orthography}
				<span class="orth">({entry.orthography})</span>
			{/if}
			<p class="definition">{(entry.full_text ?? '').slice(0, 200)}{(entry.full_text ?? '').length > 200 ? '...' : ''}</p>
		</li>
	{/each}
</ul>

<style>
	.letters {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
		margin: 1rem 0;
	}
	.letter-link {
		background: #16213e;
		padding: 0.4rem 0.7rem;
		border-radius: 4px;
		cursor: pointer;
		color: #e0e0e0;
	}
	.letter-link:hover {
		background: #e94560;
	}
	.entries {
		list-style: none;
		padding: 0;
	}
	.entries li {
		padding: 0.75rem 0;
		border-bottom: 1px solid #16213e;
	}
	.orth {
		color: #a0a0a0;
		font-style: italic;
	}
	.definition {
		margin: 0.25rem 0 0;
		font-size: 0.9rem;
		color: #c0c0c0;
	}
</style>
