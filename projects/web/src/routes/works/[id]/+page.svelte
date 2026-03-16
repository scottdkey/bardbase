<script lang="ts">
	let { data } = $props();
	let currentAct = $state(0);
	let currentScene = $state(0);

	const scenes = $derived(() => {
		const map = new Map<string, typeof data.lines>();
		for (const line of data.lines) {
			const key = `${line.act}.${line.scene}`;
			if (!map.has(key)) map.set(key, []);
			map.get(key)!.push(line);
		}
		return map;
	});
</script>

<svelte:head>
	<title>{data.work.title} — Shakespeare Database</title>
</svelte:head>

<h1>{data.work.title}</h1>
<p class="meta">{data.work.work_type} &middot; {data.work.year}</p>

{#if data.characters.length > 0}
	<details>
		<summary>Characters ({data.characters.length})</summary>
		<ul class="characters">
			{#each data.characters as char}
				<li>
					<strong>{char.name}</strong>
					{#if char.description} — {char.description}{/if}
					<span class="speech-count">({char.speech_count} speeches)</span>
				</li>
			{/each}
		</ul>
	</details>
{/if}

{#if data.editions.length > 1}
	<p class="edition-note">
		Showing: {data.currentEdition?.name} ({data.currentEdition?.year})
	</p>
{/if}

<section class="text">
	{#each data.lines as line}
		{#if line.act !== currentAct || line.scene !== currentScene}
			{@const _ = (() => { currentAct = line.act; currentScene = line.scene; })()}
			<h3>Act {line.act}, Scene {line.scene}</h3>
		{/if}
		<div class="line" class:stage-direction={line.is_stage_direction}>
			{#if line.character_name}
				<span class="speaker">{line.character_name}</span>
			{/if}
			<span class="content">{line.content}</span>
			<span class="line-num">{line.line_number}</span>
		</div>
	{/each}
</section>

<style>
	.meta {
		color: #a0a0a0;
		margin-top: -0.5rem;
	}
	.characters {
		list-style: none;
		padding: 0;
	}
	.characters li {
		padding: 0.25rem 0;
	}
	.speech-count {
		color: #a0a0a0;
		font-size: 0.85rem;
	}
	.edition-note {
		color: #a0a0a0;
		font-style: italic;
	}
	.text {
		margin-top: 2rem;
	}
	.line {
		display: grid;
		grid-template-columns: 120px 1fr 40px;
		gap: 0.5rem;
		padding: 2px 0;
		font-size: 0.95rem;
	}
	.speaker {
		color: #e94560;
		font-weight: bold;
		text-align: right;
	}
	.line-num {
		color: #555;
		text-align: right;
		font-size: 0.8rem;
	}
	.stage-direction .content {
		font-style: italic;
		color: #a0a0a0;
	}
	h3 {
		margin-top: 2rem;
		color: #e94560;
		border-bottom: 1px solid #0f3460;
		padding-bottom: 0.5rem;
	}
</style>
