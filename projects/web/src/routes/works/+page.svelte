<script lang="ts">
	let { data } = $props();
</script>

<svelte:head>
	<title>Works — Shakespeare Database</title>
</svelte:head>

<h1>Works</h1>

<h2>Plays</h2>
<ul class="works">
	{#each data.plays as work}
		<li>
			<a href="/works/{work.id}">{work.title}</a>
			<span class="meta">{work.work_type}</span>
		</li>
	{/each}
</ul>

<h2>Poetry</h2>
<ul class="works">
	{#each data.poetry as work}
		<li>
			<a href="/works/{work.id}">{work.title}</a>
			<span class="meta">{work.work_type}</span>
		</li>
	{/each}
</ul>

<style>
	.works {
		list-style: none;
		padding: 0;
	}
	.works li {
		padding: 0.5rem 0;
		border-bottom: 1px solid #16213e;
		display: flex;
		justify-content: space-between;
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
	}
</style>
