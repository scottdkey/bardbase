<script lang="ts">
	import PageHeader from '$lib/components/ui/PageHeader.svelte';
	import { readingPosition } from '$lib/stores/reading-position.svelte';
	import type { WorkDivision } from '$lib/server/api';
	import { goto } from '$app/navigation';

	let { data } = $props();

	let expandedWorkId = $state<number | null>(null);
	let tocCache = $state<Record<number, WorkDivision[]>>({});
	let tocLoading = $state(false);

	const TYPE_LABELS: Record<string, string> = {
		comedy: 'Comedies',
		history: 'Histories',
		tragedy: 'Tragedies'
	};

	function slugify(title: string): string {
		return title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
	}

	function getSlug(work: { slug?: string; title: string }): string {
		return work.slug || slugify(work.title);
	}

	let playGroups = $derived.by(() => {
		const groups = new Map<string, typeof data.works.plays>();
		for (const play of data.works.plays) {
			const list = groups.get(play.work_type) ?? [];
			list.push(play);
			groups.set(play.work_type, list);
		}
		return groups;
	});

	async function toggleWork(workId: number) {
		if (expandedWorkId === workId) {
			expandedWorkId = null;
			return;
		}
		expandedWorkId = workId;
		if (!tocCache[workId]) {
			tocLoading = true;
			try {
				const res = await fetch(`/api/works/${workId}/toc`);
				if (res.ok) {
					tocCache = { ...tocCache, [workId]: await res.json() };
				}
			} catch (err) {
				console.error('[texts] failed to load TOC:', err);
			} finally {
				tocLoading = false;
			}
		}
	}

	function gotoScene(work: { slug?: string; title: string; id: number }, act: number, scene: number) {
		goto(`/text/${getSlug(work)}/${act}/${scene}`);
	}

	function continueReading(work: { slug?: string; title: string; id: number }) {
		const pos = readingPosition.get(work.id);
		if (!pos) return;
		goto(`/text/${getSlug(work)}/${pos.act}/${pos.scene}`);
	}

	function tocByAct(toc: WorkDivision[]): Map<number, WorkDivision[]> {
		const acts = new Map<number, WorkDivision[]>();
		for (const d of toc) {
			const list = acts.get(d.act) ?? [];
			list.push(d);
			acts.set(d.act, list);
		}
		return acts;
	}

	function formatPosition(workId: number): string {
		const pos = readingPosition.get(workId);
		if (!pos) return '';
		if (pos.act === 0) return `at ${pos.scene}`;
		return `Act ${pos.act}, Scene ${pos.scene}`;
	}
</script>

{#snippet workToc(work: { slug?: string; title: string; id: number })}
	{@const saved = readingPosition.get(work.id)}
	<div class="work-toc">
		{#if saved}
			<button class="continue-btn" onclick={() => continueReading(work)}>
				Continue reading &mdash; {formatPosition(work.id)}
			</button>
		{/if}
		{#if tocLoading && !tocCache[work.id]}
			<div class="toc-loading">Loading&hellip;</div>
		{:else if tocCache[work.id]}
			{@const acts = tocByAct(tocCache[work.id])}
			{#each [...acts.entries()] as [actNum, scenes] (actNum)}
				<div class="toc-act">
					{#if actNum > 0}
						<h3 class="toc-act-title">Act {actNum}</h3>
					{/if}
					<ul class="toc-scene-list">
						{#each scenes as sc (sc.scene)}
							<li>
								<button
									class="toc-scene-btn"
									onclick={() => gotoScene(work, sc.act, sc.scene)}
								>
									{#if actNum === 0}
										{sc.description ?? `${sc.scene}`}
									{:else}
										Scene {sc.scene}
									{/if}
									<span class="toc-line-count">{sc.line_count} lines</span>
								</button>
							</li>
						{/each}
					</ul>
				</div>
			{/each}
		{/if}
	</div>
{/snippet}

<svelte:head>
	<title>Bardbase</title>
</svelte:head>

<div class="texts-page">
	<PageHeader title="Bardbase" />

	{#each ['tragedy', 'comedy', 'history'] as workType (workType)}
		{@const works = playGroups.get(workType) ?? []}
		{#if works.length > 0}
			<section class="work-group">
				<h2 class="group-title">{TYPE_LABELS[workType] ?? workType}</h2>
				<ul class="work-list">
					{#each works as work (work.id)}
						{@const saved = readingPosition.get(work.id)}
						<li class="work-item" class:expanded={expandedWorkId === work.id}>
							<button class="work-btn" onclick={() => toggleWork(work.id)}>
								<span class="work-title">{work.title}</span>
								{#if saved}
									<span class="reading-badge">{formatPosition(work.id)}</span>
								{/if}
								<span class="expand-icon">{expandedWorkId === work.id ? '\u25B4' : '\u25BE'}</span>
							</button>
							{#if expandedWorkId === work.id}
								{@render workToc(work)}
							{/if}
						</li>
					{/each}
				</ul>
			</section>
		{/if}
	{/each}

	{#if data.works.poetry.length > 0}
		<section class="work-group">
			<h2 class="group-title">Poetry</h2>
			<ul class="work-list">
				{#each data.works.poetry as work (work.id)}
					{@const saved = readingPosition.get(work.id)}
					<li class="work-item" class:expanded={expandedWorkId === work.id}>
						<button class="work-btn" onclick={() => toggleWork(work.id)}>
							<span class="work-title">{work.title}</span>
							{#if saved}
								<span class="reading-badge">{formatPosition(work.id)}</span>
							{/if}
							<span class="expand-icon">{expandedWorkId === work.id ? '\u25B4' : '\u25BE'}</span>
						</button>
						{#if expandedWorkId === work.id}
							{@render workToc(work)}
						{/if}
					</li>
				{/each}
			</ul>
		</section>
	{/if}
</div>

<style>
	.texts-page {
		max-width: 720px;
		margin: 0 auto;
		padding: 0 16px 60px;
	}

	.work-group {
		margin-bottom: 24px;
	}

	.group-title {
		margin: 0 0 8px;
		font-size: 0.8rem;
		font-weight: 700;
		color: var(--color-accent);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		padding-bottom: 6px;
		border-bottom: 1px solid var(--color-border);
	}

	.work-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}

	.work-item {
		border-bottom: 1px solid color-mix(in srgb, var(--color-border) 50%, transparent);
	}

	.work-item.expanded {
		background: var(--color-surface);
		border-radius: 8px;
		margin: 4px 0;
		border-bottom: none;
	}

	.work-btn {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 12px;
		border: none;
		background: none;
		color: var(--color-text);
		font-family: inherit;
		font-size: 0.95rem;
		cursor: pointer;
		text-align: left;
		border-radius: 8px;
		transition: background 0.15s;
		gap: 8px;
	}

	.work-btn:hover {
		background: var(--color-hover);
	}

	.work-title {
		flex: 1;
		font-weight: 500;
	}

	.reading-badge {
		font-size: 0.65rem;
		font-weight: 600;
		color: var(--color-accent);
		background: var(--color-active);
		padding: 2px 8px;
		border-radius: 10px;
		white-space: nowrap;
	}

	.expand-icon {
		font-size: 0.8rem;
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.work-toc {
		padding: 0 12px 12px;
	}

	.continue-btn {
		display: block;
		width: 100%;
		padding: 10px 12px;
		margin-bottom: 8px;
		border: 1px solid var(--color-accent);
		background: none;
		color: var(--color-accent);
		font-family: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		cursor: pointer;
		border-radius: 8px;
		text-align: center;
		transition: background 0.15s;
	}

	.continue-btn:hover {
		background: var(--color-hover);
	}

	.toc-loading {
		padding: 8px 12px;
		font-size: 0.8rem;
		color: var(--color-text-muted);
	}

	.toc-act {
		margin-bottom: 8px;
	}

	.toc-act-title {
		margin: 0 0 2px;
		font-size: 0.7rem;
		font-weight: 700;
		color: var(--color-accent);
		text-transform: uppercase;
		letter-spacing: 0.04em;
		padding: 4px 12px;
	}

	.toc-scene-list {
		list-style: none;
		padding: 0;
		margin: 0;
	}

	.toc-scene-btn {
		display: flex;
		justify-content: space-between;
		align-items: center;
		width: 100%;
		padding: 6px 12px;
		border: none;
		background: none;
		color: var(--color-text);
		font-family: inherit;
		font-size: 0.85rem;
		cursor: pointer;
		border-radius: 6px;
		text-align: left;
		transition: background 0.15s;
	}

	.toc-scene-btn:hover {
		background: var(--color-hover);
	}

	.toc-line-count {
		font-size: 0.6rem;
		color: var(--color-text-muted);
		flex-shrink: 0;
		margin-left: 8px;
	}
</style>
