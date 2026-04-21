<script>
	export const prerender = true;
</script>

<svelte:head>
	<title>Help &mdash; Bardbase</title>
</svelte:head>

<div class="help-page">
	<header class="page-header">
		<h1>How to use Bardbase</h1>
	</header>

	<section class="section">
		<h2>Navigating the texts</h2>
		<p>Open the <span class="key">☰</span> menu to browse plays and poetry. Works are grouped by genre — click a group header (Tragedies, Comedies, etc.) to expand it. Click any play to load its table of contents, then select a scene to begin reading.</p>
		<p>Your reading position is saved automatically for each work. Use <em>Continue reading</em> in the table of contents to pick up where you left off.</p>
	</section>

	<section class="section">
		<h2>Editions</h2>
		<p>Multiple editions are displayed side by side. Use the <strong>Editions</strong> dropdown in the reading page header to show or hide specific editions. The First Folio and Open Source Shakespeare (Globe) editions are included for most plays.</p>
	</section>

	<section class="section">
		<h2>Inline references</h2>
		<p>Words with a <span class="dotted">dotted underline</span> have scholarly references attached. Hover or tap a word to see a summary tooltip. Click the word to open the full entry in the reference panel on the right.</p>
		<p>The reference panel shows the full entry text, any sub-entries, and citations back to the play and line where the word appears.</p>
	</section>

	<section class="section">
		<h2>Browsing references</h2>
		<p>Go to <a href="/references">References</a> to browse or search all five reference works. On the all-references page, use the source toggle pills to limit results to specific works. Click a source name in the <span class="key">☰</span> menu to go directly to that reference's search page.</p>
	</section>

	<section class="section">
		<h2>Search</h2>
		<p>Type in the search bar in the <span class="key">☰</span> menu and press <strong>Enter</strong> to run a full-text search across all reference works. Results rank exact headword matches above partial matches and body text matches.</p>
	</section>

	<section class="section">
		<h2>Reporting errors</h2>
		<p>If you spot an error in the text or a reference, use the <a href="/corrections">Corrections</a> page to submit a report. The flag icon on individual lines in the reading view opens a correction form pre-filled with the work, act, scene, and line.</p>
	</section>
</div>

<style>
	.help-page {
		max-width: 660px;
		margin: 0 auto;
		padding: 32px 24px 80px;
	}

	.page-header {
		margin-bottom: 36px;
		padding-bottom: 20px;
		border-bottom: 1px solid var(--color-border);
	}

	.page-header h1 {
		margin: 0;
		font-size: 2rem;
		font-weight: 700;
		color: var(--color-text);
		letter-spacing: -0.01em;
	}

	.section {
		margin-bottom: 32px;
	}

	.section h2 {
		margin: 0 0 10px;
		font-size: 0.75rem;
		font-weight: 700;
		color: var(--color-accent);
		text-transform: uppercase;
		letter-spacing: 0.06em;
	}

	.section p {
		margin: 0 0 10px;
		font-size: 0.88rem;
		color: var(--color-text-secondary);
		line-height: 1.7;
	}

	.section p:last-child { margin-bottom: 0; }

	.key {
		display: inline-block;
		padding: 0 5px;
		border: 1px solid var(--color-border);
		border-radius: 4px;
		font-size: 0.78rem;
		color: var(--color-text-muted);
		background: var(--color-surface);
		vertical-align: middle;
	}

	.dotted {
		border-bottom: 1.5px dotted var(--color-accent);
	}
</style>
