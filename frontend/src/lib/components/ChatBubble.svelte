<script lang="ts">
	import { marked } from 'marked';

	interface Props {
		role: 'user' | 'assistant';
		content: string;
		streaming?: boolean;
	}

	let { role, content, streaming = false }: Props = $props();

	const isUser = $derived(role === 'user');

	// Parse assistant messages as GitHub-Flavored Markdown.
	// User messages are shown as plain text (preserving formatting intent).
	const html = $derived(
		isUser ? '' : (marked.parse(content, { gfm: true, breaks: false }) as string)
	);
</script>

<div class="flex {isUser ? 'justify-end' : 'justify-start'}">
	<div
		class="max-w-[85%] rounded-lg px-4 py-3 text-sm leading-relaxed md:max-w-[70%]
			{isUser ? 'bg-[#58a6ff]/10 text-[#e6edf3]' : 'bg-[#161b22] text-[#e6edf3]'}"
	>
		{#if isUser}
			<p class="whitespace-pre-wrap break-words">
				{content}{#if streaming}<span
					class="ml-1 inline-block h-4 w-1.5 animate-pulse bg-[#58a6ff] align-middle"
					aria-label="Streaming"
				></span>{/if}
			</p>
		{:else}
			<div class="md-prose">{@html html}</div>
			{#if streaming}
				<span
					class="ml-1 inline-block h-4 w-1.5 animate-pulse bg-[#58a6ff] align-middle"
					aria-label="Streaming"
				></span>
			{/if}
		{/if}
	</div>
</div>

<style>
	/* Markdown prose styles scoped to assistant bubbles via :global on .md-prose children.
	   Svelte's {@html} bypasses scoping, so :global is required for generated HTML elements. */

	:global(.md-prose > *:first-child) { margin-top: 0; }
	:global(.md-prose > *:last-child) { margin-bottom: 0; }

	:global(.md-prose p) { margin-bottom: 0.5rem; }

	:global(.md-prose h1),
	:global(.md-prose h2),
	:global(.md-prose h3),
	:global(.md-prose h4) {
		font-weight: 700;
		margin-top: 0.75rem;
		margin-bottom: 0.25rem;
		color: #e6edf3;
	}
	:global(.md-prose h1) { font-size: 1.1em; }
	:global(.md-prose h2) { font-size: 1em; }

	:global(.md-prose ul) { list-style: disc; padding-left: 1.25rem; margin-bottom: 0.5rem; }
	:global(.md-prose ol) { list-style: decimal; padding-left: 1.25rem; margin-bottom: 0.5rem; }
	:global(.md-prose li) { margin-bottom: 0.2rem; }
	:global(.md-prose li > p) { margin-bottom: 0.2rem; }

	:global(.md-prose strong) { font-weight: 700; }
	:global(.md-prose em) { font-style: italic; }

	/* Inline code */
	:global(.md-prose code) {
		font-family: ui-monospace, 'Cascadia Code', 'Fira Code', monospace;
		font-size: 0.85em;
		background: rgba(255, 255, 255, 0.08);
		padding: 0.1em 0.35em;
		border-radius: 3px;
		color: #79c0ff;
		word-break: break-word;
	}

	/* Fenced code blocks */
	:global(.md-prose pre) {
		background: #0d1117;
		border: 1px solid rgba(255, 255, 255, 0.08);
		border-radius: 6px;
		padding: 0.75rem 1rem;
		margin: 0.5rem 0;
		overflow-x: auto;
	}
	:global(.md-prose pre code) {
		background: none;
		padding: 0;
		color: #e6edf3;
		font-size: 0.8rem;
		line-height: 1.5;
		word-break: normal;
	}

	:global(.md-prose a) {
		color: #58a6ff;
		text-decoration: underline;
		text-underline-offset: 2px;
	}
	:global(.md-prose a:hover) { color: #79b8ff; }

	:global(.md-prose blockquote) {
		border-left: 2px solid rgba(255, 255, 255, 0.2);
		padding-left: 0.75rem;
		margin: 0.5rem 0;
		color: rgba(230, 237, 243, 0.6);
		font-style: italic;
	}

	:global(.md-prose hr) {
		border: none;
		border-top: 1px solid rgba(255, 255, 255, 0.1);
		margin: 0.75rem 0;
	}

	/* GFM tables */
	:global(.md-prose table) {
		width: 100%;
		border-collapse: collapse;
		margin: 0.5rem 0;
		font-size: 0.85em;
	}
	:global(.md-prose th) {
		border: 1px solid rgba(255, 255, 255, 0.12);
		padding: 0.35rem 0.5rem;
		font-weight: 600;
		background: rgba(255, 255, 255, 0.04);
		text-align: left;
	}
	:global(.md-prose td) {
		border: 1px solid rgba(255, 255, 255, 0.12);
		padding: 0.35rem 0.5rem;
	}

	/* GFM task list checkboxes */
	:global(.md-prose input[type='checkbox']) {
		margin-right: 0.4rem;
		accent-color: #58a6ff;
	}
</style>
