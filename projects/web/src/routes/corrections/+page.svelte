<script lang="ts">
	import PageHeader from '$lib/components/ui/PageHeader.svelte';
	import { corrections } from '$lib/stores/corrections.svelte';

	let { data } = $props();

	let filter = $state<'all' | 'open' | 'closed'>('all');

	let filteredIssues = $derived(
		filter === 'all'
			? data.issues
			: data.issues.filter((i) => i.state === filter)
	);

	let openCount = $derived(data.issues.filter((i) => i.state === 'open').length);
	let closedCount = $derived(data.issues.filter((i) => i.state === 'closed').length);

	function newIssueUrl() {
		return 'https://github.com/scottdkey/bardbase/issues/new?labels=correction&template=correction.md';
	}

	function formatDate(iso: string): string {
		return new Date(iso).toLocaleDateString('en-US', {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	// Check for any pending local corrections that haven't been submitted yet
	let pendingLocal = $derived(corrections.pending);
</script>

<svelte:head>
	<title>Corrections &mdash; Bardbase</title>
</svelte:head>

<div class="corrections-page">
	<PageHeader title="Corrections">
		<a href={newIssueUrl()} target="_blank" rel="noopener" class="new-issue-link">
			+ New Issue
		</a>
	</PageHeader>

	{#if pendingLocal.length > 0}
		<div class="local-pending">
			<h3>Pending Submissions ({pendingLocal.length})</h3>
			<p class="hint">These corrections haven't been submitted to GitHub yet.</p>
			{#each pendingLocal as c (c.id)}
				<div class="local-card">
					<div class="card-header">
						<span class="card-entry">{c.entryKey}</span>
						{#if c.citationRef}
							<span class="card-ref">{c.citationRef}</span>
						{/if}
					</div>
					<p class="card-text">{c.correctionText}</p>
					<div class="card-actions">
						<a
							href={corrections.toGitHubIssueUrl(c)}
							target="_blank"
							rel="noopener"
							class="submit-link"
							onclick={() => corrections.markSubmitted(c.id)}
						>
							Submit to GitHub
						</a>
						<button class="remove-btn" onclick={() => corrections.remove(c.id)}>
							Remove
						</button>
					</div>
				</div>
			{/each}
		</div>
	{/if}

	<div class="filter-bar">
		<button class="filter-btn" class:active={filter === 'all'} onclick={() => (filter = 'all')}>
			All ({data.issues.length})
		</button>
		<button
			class="filter-btn"
			class:active={filter === 'open'}
			onclick={() => (filter = 'open')}
		>
			Open ({openCount})
		</button>
		<button
			class="filter-btn"
			class:active={filter === 'closed'}
			onclick={() => (filter = 'closed')}
		>
			Closed ({closedCount})
		</button>
	</div>

	{#if filteredIssues.length === 0}
		<div class="empty-state">
			<p>No {filter === 'all' ? '' : filter} corrections found.</p>
			<p class="hint">
				Flag entries or citations while browsing the lexicon, or
				<a href={newIssueUrl()} target="_blank" rel="noopener">create an issue on GitHub</a>.
			</p>
		</div>
	{:else}
		<ul class="issue-list">
			{#each filteredIssues as issue (issue.number)}
				<li class="issue-card">
					<a href={issue.url} target="_blank" rel="noopener" class="issue-link">
						<div class="issue-header">
							<span class="issue-state" class:open={issue.state === 'open'}>
								{issue.state}
							</span>
							<span class="issue-number">#{issue.number}</span>
						</div>
						<h3 class="issue-title">{issue.title}</h3>
						{#if issue.labels.length > 1}
							<div class="issue-labels">
								{#each issue.labels.filter((l) => l !== 'correction') as label}
									<span class="issue-label">{label}</span>
								{/each}
							</div>
						{/if}
						<div class="issue-meta">
							<span>opened {formatDate(issue.created_at)}</span>
							{#if issue.state === 'closed'}
								<span>closed {formatDate(issue.updated_at)}</span>
							{/if}
						</div>
					</a>
				</li>
			{/each}
		</ul>
	{/if}
</div>

<style>
	.corrections-page {
		max-width: 720px;
		margin: 0 auto;
		padding: 0 16px 60px;
	}

	.new-issue-link {
		font-size: 0.8rem;
		font-weight: 600;
		color: var(--color-accent);
		text-decoration: none;
		padding: 4px 10px;
		border: 1px solid var(--color-accent);
		border-radius: 6px;
	}

	.new-issue-link:hover {
		background: var(--color-hover);
	}

	/* ─── Local pending ─── */
	.local-pending {
		background: var(--color-surface);
		border: 1px solid var(--color-warning);
		border-radius: 10px;
		padding: 14px;
		margin-bottom: 16px;
	}

	.local-pending h3 {
		margin: 0 0 4px;
		font-size: 0.9rem;
		color: var(--color-warning);
	}

	.local-card {
		padding: 8px 0;
		border-bottom: 1px solid var(--color-border);
	}

	.local-card:last-child {
		border-bottom: none;
	}

	.card-header {
		display: flex;
		gap: 8px;
		align-items: baseline;
	}

	.card-entry {
		font-weight: 700;
		color: var(--color-text);
	}

	.card-ref {
		font-size: 0.8rem;
		color: var(--color-accent);
		font-weight: 600;
	}

	.card-text {
		margin: 4px 0;
		font-size: 0.85rem;
		color: var(--color-text-secondary);
	}

	.card-actions {
		display: flex;
		gap: 8px;
		margin-top: 4px;
	}

	.submit-link {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--color-accent);
	}

	.remove-btn {
		font-size: 0.75rem;
		font-weight: 600;
		color: var(--color-danger);
		background: none;
		border: none;
		cursor: pointer;
		font-family: inherit;
		padding: 0;
	}

	/* ─── Filter bar ─── */
	.filter-bar {
		display: flex;
		gap: 4px;
		margin-bottom: 16px;
		border-bottom: 1px solid var(--color-border);
	}

	.filter-btn {
		padding: 8px 14px;
		border: none;
		background: none;
		color: var(--color-text-muted);
		font-family: inherit;
		font-size: 0.8rem;
		font-weight: 600;
		cursor: pointer;
		border-bottom: 2px solid transparent;
		transition: color 0.15s, border-color 0.15s;
	}

	.filter-btn:hover {
		color: var(--color-text);
	}

	.filter-btn.active {
		color: var(--color-accent);
		border-bottom-color: var(--color-accent);
	}

	/* ─── Empty state ─── */
	.empty-state {
		text-align: center;
		padding: 40px 0;
		color: var(--color-text-muted);
	}

	.empty-state p {
		margin: 0 0 8px;
	}

	.hint {
		font-size: 0.85rem;
		color: var(--color-text-muted);
	}

	/* ─── Issue list ─── */
	.issue-list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.issue-card {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 10px;
		transition: border-color 0.15s;
	}

	.issue-card:hover {
		border-color: var(--color-accent);
	}

	.issue-link {
		display: block;
		padding: 14px;
		text-decoration: none;
		color: inherit;
	}

	.issue-header {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 4px;
	}

	.issue-state {
		font-size: 0.65rem;
		font-weight: 700;
		text-transform: uppercase;
		padding: 2px 6px;
		border-radius: 4px;
		background: var(--color-hover);
		color: var(--color-text-muted);
	}

	.issue-state.open {
		background: rgba(109, 218, 208, 0.15);
		color: var(--color-accent);
	}

	.issue-number {
		font-size: 0.75rem;
		color: var(--color-text-muted);
		font-weight: 600;
	}

	.issue-title {
		margin: 0;
		font-size: 0.95rem;
		font-weight: 600;
		color: var(--color-text);
		line-height: 1.4;
	}

	.issue-labels {
		display: flex;
		gap: 4px;
		margin-top: 6px;
		flex-wrap: wrap;
	}

	.issue-label {
		font-size: 0.6rem;
		font-weight: 600;
		padding: 1px 6px;
		border-radius: 10px;
		background: var(--color-hover);
		color: var(--color-text-muted);
	}

	.issue-meta {
		display: flex;
		gap: 12px;
		margin-top: 6px;
		font-size: 0.7rem;
		color: var(--color-text-muted);
	}
</style>
