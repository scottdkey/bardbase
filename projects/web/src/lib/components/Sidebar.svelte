<script lang="ts">
	import { slide } from 'svelte/transition';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { readingPosition } from '$lib/stores/reading-position.svelte';
	import { corrections } from '$lib/stores/corrections.svelte';
	import type { WorkDivision } from '$lib/server/api';

	interface Work {
		id: number;
		title: string;
		slug?: string;
		work_type?: string;
	}

	interface Works {
		plays: Work[];
		poetry: Work[];
	}

	let { works, onclose }: { works: Works; onclose?: () => void } = $props();

	const TYPE_ORDER = ['tragedy', 'comedy', 'history'];
	const SOURCE_LABELS: Record<string, string> = {
		schmidt: 'Schmidt Lexicon',
		onions: 'Onions Glossary',
		abbott: 'Abbott Grammar',
		bartlett: 'Bartlett Concordance',
		henley_farmer: 'Henley & Farmer'
	};
	const SOURCE_ORDER = ['schmidt', 'onions', 'abbott', 'bartlett', 'henley_farmer'];
	const TYPE_LABELS: Record<string, string> = {
		comedy: 'Comedies',
		history: 'Histories',
		tragedy: 'Tragedies'
	};

	let searchQuery = $state('');
	let expandedWorkId = $state<number | null>(null);
	let workTocs = $state<Record<number, WorkDivision[]>>({});
	let loadingToc = $state<number | null>(null);
	let expandedGenres = $state<Set<string>>(new Set(['tragedy', 'comedy', 'history', 'poetry']));

	let currentPath = $derived(page.url.pathname);

	function isActive(href: string): boolean {
		if (href === '/') return currentPath === '/';
		return currentPath.startsWith(href);
	}

	function slugify(title: string): string {
		return title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '');
	}

	function getSlug(work: Work): string {
		return work.slug || slugify(work.title);
	}

	function formatPosition(work: Work): { label: string; state: 'unread' | 'reading' | 'completed' } {
		const pos = readingPosition.get(work.id);
		if (!pos) return { label: '', state: 'unread' };
		if (pos.completed) return { label: 'Completed', state: 'completed' };
		if (work.title === 'Sonnets') return { label: `Sonnet ${pos.scene}`, state: 'reading' };
		if (pos.act === 0) return { label: pos.scene.toString(), state: 'reading' };
		return { label: `Act ${pos.act}, Scene ${pos.scene}`, state: 'reading' };
	}

	async function toggleWork(work: Work) {
		if (expandedWorkId === work.id) {
			expandedWorkId = null;
			return;
		}
		expandedWorkId = work.id;
		if (!workTocs[work.id]) {
			loadingToc = work.id;
			try {
				const res = await fetch(`/api/works/${work.id}/toc`);
				if (res.ok) workTocs = { ...workTocs, [work.id]: await res.json() };
			} finally {
				loadingToc = null;
			}
		}
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

	function navigate(path: string) {
		goto(path);
		onclose?.();
	}

	function toggleGenre(type: string) {
		const next = new Set(expandedGenres);
		if (next.has(type)) next.delete(type);
		else next.add(type);
		expandedGenres = next;
	}

	let playGroups = $derived.by(() => {
		const groups = new Map<string, Work[]>();
		for (const play of works.plays) {
			const list = groups.get(play.work_type ?? '') ?? [];
			list.push(play);
			groups.set(play.work_type ?? '', list);
		}
		return groups;
	});

	function handleSearchKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' && searchQuery.trim()) {
			navigate(`/references?q=${encodeURIComponent(searchQuery.trim())}`);
		}
	}
</script>

{#snippet workRow(work: Work)}
	{@const position = formatPosition(work)}
	<div class="work-item">
		<button
			class="work-btn"
			class:expanded={expandedWorkId === work.id}
			onclick={() => toggleWork(work)}
			aria-expanded={expandedWorkId === work.id}
		>
			<span class="work-meta">
				<span class="work-name">{work.title}</span>
				{#if position.label}
					<span class="pos-badge pos-{position.state}" aria-label="Reading position: {position.label}">{position.label}</span>
				{/if}
			</span>
			<span class="chevron" aria-hidden="true">{expandedWorkId === work.id ? '▴' : '▾'}</span>
		</button>

		{#if expandedWorkId === work.id}
			<div class="work-toc" role="region" aria-label="Table of contents for {work.title}" transition:slide={{ duration: 180 }}>
				{#if loadingToc === work.id}
					<div class="toc-loading" aria-live="polite">Loading…</div>
				{:else if workTocs[work.id]}
					{@const acts = tocByAct(workTocs[work.id])}
					{#if position.state === 'reading'}
						<button class="continue-btn" onclick={() => {
							const p = readingPosition.get(work.id);
							if (p) navigate(`/text/${getSlug(work)}/${p.act}/${p.scene}`);
						}}>
							Continue — {position.label}
						</button>
					{/if}
					{#each [...acts.entries()] as [actNum, scenes] (actNum)}
						<div class="toc-act">
							{#if actNum > 0}
								<span class="act-label">Act {actNum}</span>
							{/if}
							<ul class="scene-list">
								{#each scenes as sc (sc.scene)}
									<li>
										<button
											class="scene-btn"
											onclick={() => navigate(`/text/${getSlug(work)}/${sc.act}/${sc.scene}`)}
										>
											{#if actNum === 0}
												{sc.description ?? `${sc.scene}`}
											{:else if work.title === 'Sonnets'}
												Sonnet {sc.scene}
											{:else}
												Scene {sc.scene}
											{/if}
											<span class="line-count">{sc.line_count}</span>
										</button>
									</li>
								{/each}
							</ul>
						</div>
					{/each}
				{/if}
			</div>
		{/if}
	</div>
{/snippet}

<nav class="sidebar" aria-label="Site navigation">
	<div class="sidebar-search">
		<svg class="search-icon" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
			<circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
		</svg>
		<input
			class="search-input"
			type="search"
			placeholder="Press Enter to search"
			bind:value={searchQuery}
			onkeydown={handleSearchKeydown}
			aria-label="Search references — press Enter to go to results"
		/>
	</div>

	<div class="sidebar-body">
		<section class="sidebar-section">
			{#each TYPE_ORDER as workType (workType)}
				{@const groupWorks = playGroups.get(workType) ?? []}
				{#if groupWorks.length > 0}
					{@const genreOpen = expandedGenres.has(workType)}
					<div class="work-type-group">
						<button class="genre-btn" onclick={() => toggleGenre(workType)} aria-expanded={genreOpen}>
							<span class="genre-label">{TYPE_LABELS[workType] ?? workType}</span>
							<span class="genre-chevron" aria-hidden="true">{genreOpen ? '▴' : '▾'}</span>
						</button>
						{#if genreOpen}
							<div transition:slide={{ duration: 180 }}>
								{#each groupWorks as work (work.id)}
									{@render workRow(work)}
								{/each}
							</div>
						{/if}
					</div>
				{/if}
			{/each}

			{#if works.poetry.length > 0}
				{@const poetryOpen = expandedGenres.has('poetry')}
				<div class="work-type-group">
					<button class="genre-btn" onclick={() => toggleGenre('poetry')} aria-expanded={poetryOpen}>
						<span class="genre-label">Poetry</span>
						<span class="genre-chevron" aria-hidden="true">{poetryOpen ? '▴' : '▾'}</span>
					</button>
					{#if poetryOpen}
						<div transition:slide={{ duration: 180 }}>
							{#each works.poetry as work (work.id)}
								{@render workRow(work)}
							{/each}
						</div>
					{/if}
				</div>
			{/if}
		</section>

		<section class="sidebar-section">
			<a
				href="/references"
				class="section-link"
				class:active={isActive('/references') || isActive('/reference') || isActive('/lexicon')}
				onclick={(e) => { e.preventDefault(); navigate('/references'); }}
			>
				References
			</a>
			{#each SOURCE_ORDER as src (src)}
				<a
					href="/references?source={src}"
					class="source-link"
					onclick={(e) => { e.preventDefault(); navigate(`/references?source=${src}`); }}
				>
					{SOURCE_LABELS[src]}
				</a>
			{/each}
		</section>

		<section class="sidebar-section">
			<a
				href="/corrections"
				class="section-link"
				class:active={isActive('/corrections')}
				onclick={(e) => { e.preventDefault(); navigate('/corrections'); }}
			>
				Corrections
				{#if corrections.pendingCount > 0}
					<span class="badge" aria-label="{corrections.pendingCount} pending">{corrections.pendingCount}</span>
				{/if}
			</a>
			<a
				href="/help"
				class="section-link"
				class:active={isActive('/help')}
				onclick={(e) => { e.preventDefault(); navigate('/help'); }}
			>
				Help
			</a>
			<a
				href="/about"
				class="section-link"
				class:active={isActive('/about')}
				onclick={(e) => { e.preventDefault(); navigate('/about'); }}
			>
				About
			</a>
		</section>
	</div>
</nav>

<style>
	.sidebar {
		display: flex;
		flex-direction: column;
		height: 100%;
		overflow: hidden;
	}

	.sidebar-search {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 12px 14px;
		border-bottom: 1px solid var(--color-border);
		flex-shrink: 0;
	}

	.search-icon {
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.search-input {
		flex: 1;
		background: none;
		border: none;
		outline: none;
		color: var(--color-text);
		font-family: inherit;
		font-size: 0.8rem;
		min-width: 0;
	}

	.search-input::placeholder { color: var(--color-text-muted); }
	.search-input::-webkit-search-cancel-button { opacity: 0.5; }

	.sidebar-body {
		flex: 1;
		overflow-y: auto;
		padding: 4px 0 16px;
	}

	.sidebar-section {
		border-bottom: 1px solid var(--color-border);
		padding: 4px 0;
	}

	.sidebar-section:last-child { border-bottom: none; }

	.section-link {
		display: flex;
		align-items: center;
		gap: 6px;
		width: 100%;
		padding: 8px 14px;
		font-family: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		color: var(--color-text-secondary);
		background: none;
		border: none;
		text-decoration: none;
		cursor: pointer;
		text-align: left;
		transition: color 0.15s, background 0.15s;
	}

	.section-link:hover { color: var(--color-text); background: var(--color-hover); }
	.section-link.active { color: var(--color-accent); }

	.source-link {
		display: block;
		padding: 4px 14px 4px 26px;
		font-family: inherit;
		font-size: 0.75rem;
		color: var(--color-text-secondary);
		text-decoration: none;
		transition: color 0.15s, background 0.15s;
	}

	.source-link:hover { color: var(--color-text); background: var(--color-hover); }

	.badge {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-width: 16px;
		height: 16px;
		padding: 0 4px;
		border-radius: 8px;
		background: var(--color-warning);
		color: var(--color-bg);
		font-size: 0.55rem;
		font-weight: 800;
	}

	.work-type-group { padding: 0 0 2px; }

	.genre-btn {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 5px 14px;
		background: none;
		border: none;
		cursor: pointer;
		gap: 4px;
		transition: background 0.15s;
	}

	.genre-btn:hover { background: var(--color-hover); }

	.genre-label {
		flex: 1;
		font-size: 0.6rem;
		font-weight: 700;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.08em;
		text-align: left;
	}

	.genre-chevron {
		font-size: 0.55rem;
		color: var(--color-text-muted);
	}

	.work-item { border-radius: 0; }

	.work-btn {
		display: flex;
		align-items: center;
		width: 100%;
		padding: 5px 14px 5px 20px;
		background: none;
		border: none;
		color: var(--color-text);
		font-family: inherit;
		font-size: 0.82rem;
		cursor: pointer;
		text-align: left;
		gap: 6px;
		transition: background 0.15s;
	}

	.work-btn:hover { background: var(--color-hover); }
	.work-btn.expanded { background: var(--color-active); color: var(--color-accent); }

	.work-meta {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 1px;
	}

	.work-name {
		min-width: 0;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.pos-badge {
		font-size: 0.62rem;
		white-space: nowrap;
	}

	.pos-reading {
		color: var(--color-accent);
	}

	.pos-completed {
		color: var(--color-text-muted);
		font-style: italic;
	}

	.chevron {
		font-size: 0.65rem;
		color: var(--color-text-muted);
		flex-shrink: 0;
	}

	.work-toc { padding: 2px 0 4px; }

	.toc-loading {
		padding: 6px 20px;
		font-size: 0.75rem;
		color: var(--color-text-muted);
	}

	.continue-btn {
		display: block;
		width: calc(100% - 28px);
		margin: 4px 14px 6px;
		padding: 5px 10px;
		border: 1px solid var(--color-accent);
		background: none;
		color: var(--color-accent);
		font-family: inherit;
		font-size: 0.7rem;
		font-weight: 600;
		cursor: pointer;
		border-radius: 6px;
		text-align: center;
		transition: background 0.15s;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.continue-btn:hover { background: var(--color-hover); }

	.toc-act { margin-bottom: 1px; }

	.act-label {
		display: block;
		padding: 3px 20px 1px;
		font-size: 0.58rem;
		font-weight: 700;
		color: var(--color-accent);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.scene-list { list-style: none; padding: 0; margin: 0; }

	.scene-btn {
		display: flex;
		justify-content: space-between;
		align-items: center;
		width: 100%;
		padding: 3px 14px 3px 26px;
		background: none;
		border: none;
		color: var(--color-text-secondary);
		font-family: inherit;
		font-size: 0.78rem;
		cursor: pointer;
		text-align: left;
		transition: background 0.15s, color 0.15s;
		gap: 8px;
	}

	.scene-btn:hover { background: var(--color-hover); color: var(--color-text); }

	.line-count {
		font-size: 0.58rem;
		color: var(--color-text-muted);
		flex-shrink: 0;
	}
</style>
