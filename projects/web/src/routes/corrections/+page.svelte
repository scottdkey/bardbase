<script lang="ts">
	import { dev } from '$app/environment';
	import { corrections } from '$lib/stores/corrections.svelte';
	import PageHeader from '$lib/components/ui/PageHeader.svelte';
	import Button from '$lib/components/ui/Button.svelte';

	function submitToGitHub(id: string) {
		const c = corrections.items.find((i) => i.id === id);
		if (!c) return;
		const url = corrections.toGitHubIssueUrl(c);
		window.open(url, '_blank');
		corrections.markSubmitted(id);
	}

	async function submitLocal(id: string) {
		const c = corrections.items.find((i) => i.id === id);
		if (!c) return;
		try {
			await fetch('/api/corrections', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(c)
			});
			corrections.markSubmitted(id);
		} catch (e) {
			console.error('Failed to save correction:', e);
		}
	}
</script>

<svelte:head>
	<title>Corrections &mdash; Bardbase</title>
</svelte:head>

<div class="corrections-page">
	<PageHeader title="Corrections"><a href="/" class="back-link">Back to Lexicon</a></PageHeader>

	{#if corrections.count === 0}
		<div class="empty-state">
			<p>No corrections flagged yet.</p>
			<p class="hint">While viewing a scene, hover over any line and click the flag icon to report an issue.</p>
		</div>
	{:else}
		<div class="stats">
			{corrections.pendingCount} pending / {corrections.count} total
		</div>

		<ul class="correction-list">
			{#each corrections.items as c (c.id)}
				<li class="correction-card" class:submitted={c.status === 'submitted'}>
					<div class="card-header">
						<div class="card-location">
							<span class="card-entry">{c.entryKey}</span>
							<span class="card-ref">{c.workTitle} {c.act}.{c.scene}.{c.lineNumber}</span>
						</div>
						<span class="card-status" class:pending={c.status === 'pending'}>
							{c.status}
						</span>
					</div>

					{#if c.characterName}
						<div class="card-speaker">{c.characterName}</div>
					{/if}

					<div class="card-current">
						<span class="card-label">Current:</span>
						<span class="card-text">{c.currentText}</span>
					</div>

					<div class="card-correction">
						<span class="card-label">Correction:</span>
						<span class="card-text">{c.correctionText}</span>
					</div>

					{#if c.notes}
						<div class="card-notes">
							<span class="card-label">Notes:</span>
							<span class="card-text">{c.notes}</span>
						</div>
					{/if}

					<div class="card-meta">
						<span class="card-time">{new Date(c.timestamp).toLocaleDateString()}</span>
						<span class="card-edition">{c.editionName}</span>
					</div>

					<div class="card-actions">
						{#if c.status === 'pending'}
							{#if dev}
								<Button variant="primary" onclick={() => submitLocal(c.id)}>Save to File</Button>
							{:else}
								<Button variant="primary" onclick={() => submitToGitHub(c.id)}>Submit to GitHub</Button>
							{/if}
						{/if}
						<Button variant="danger" onclick={() => corrections.remove(c.id)}>Remove</Button>
					</div>
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
	}

	.stats {
		font-size: 0.8rem;
		color: var(--color-text-muted);
		margin-bottom: 12px;
	}

	.correction-list {
		list-style: none;
		padding: 0;
		margin: 0;
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.correction-card {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: 10px;
		padding: 14px;
	}

	.correction-card.submitted {
		opacity: 0.6;
	}

	.card-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 8px;
	}

	.card-entry {
		font-weight: 700;
		color: var(--color-text);
		margin-right: 8px;
	}

	.card-ref {
		font-size: 0.8rem;
		color: var(--color-accent);
		font-weight: 600;
	}

	.card-status {
		font-size: 0.65rem;
		font-weight: 600;
		text-transform: uppercase;
		padding: 2px 6px;
		border-radius: 4px;
		background: var(--color-hover);
		color: var(--color-text-muted);
	}

	.card-status.pending {
		background: rgba(232, 167, 53, 0.15);
		color: var(--color-warning);
	}

	.card-speaker {
		font-size: 0.7rem;
		font-weight: 600;
		color: var(--color-text-muted);
		text-transform: uppercase;
		letter-spacing: 0.03em;
		margin-bottom: 6px;
	}

	.card-current,
	.card-correction,
	.card-notes {
		margin-bottom: 6px;
		font-size: 0.85rem;
		line-height: 1.5;
	}

	.card-label {
		font-weight: 600;
		color: var(--color-text-muted);
		font-size: 0.75rem;
		margin-right: 4px;
	}

	.card-current .card-text {
		font-style: italic;
		color: var(--color-text-secondary);
	}

	.card-correction .card-text {
		color: var(--color-text);
	}

	.card-meta {
		display: flex;
		gap: 12px;
		font-size: 0.7rem;
		color: var(--color-text-muted);
		margin-bottom: 8px;
	}

	.card-actions {
		display: flex;
		gap: 8px;
	}

</style>
