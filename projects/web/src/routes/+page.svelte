<script lang="ts">
	let { data } = $props();
</script>

<svelte:head>
	<title>Shakespeare Database</title>
</svelte:head>

<h1>Shakespeare Database</h1>

<section class="stats">
	<div class="stat">
		<span class="number">{data.stats.work_count}</span>
		<span class="label">Works</span>
	</div>
	<div class="stat">
		<span class="number">{data.stats.character_count.toLocaleString()}</span>
		<span class="label">Characters</span>
	</div>
	<div class="stat">
		<span class="number">{data.stats.line_count.toLocaleString()}</span>
		<span class="label">Text Lines</span>
	</div>
	<div class="stat">
		<span class="number">{data.stats.lexicon_count.toLocaleString()}</span>
		<span class="label">Lexicon Entries</span>
	</div>
</section>

<h2>Works</h2>
<ul class="works">
	{#each data.works as work}
		<li>
			<a href="/works/{work.id}">{work.title}</a>
			<span class="meta">{work.work_type}</span>
		</li>
	{/each}
</ul>

<style>
	.stats {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
		gap: 1rem;
		margin: 2rem 0;
	}

	.stat {
		background: #16213e;
		padding: 1.5rem;
		border-radius: 8px;
		text-align: center;
	}

	.number {
		display: block;
		font-size: 2rem;
		font-weight: bold;
		color: #e94560;
	}

	.label {
		font-size: 0.85rem;
		color: #a0a0a0;
	}

	.works {
		list-style: none;
		padding: 0;
	}

	.works li {
		padding: 0.5rem 0;
		border-bottom: 1px solid #16213e;
		display: flex;
		justify-content: space-between;
		align-items: center;
	}

	.works a {
		color: #e0e0e0;
		text-decoration: none;
	}

	.works a:hover {
		color: #e94560;
	}

	.meta {
		font-size: 0.8rem;
		color: #a0a0a0;
		text-transform: capitalize;
	}
</style>
