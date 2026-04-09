<script lang="ts">
	import { marked } from 'marked';

	interface Props {
		role: 'user' | 'assistant';
		content: string;
		streaming?: boolean;
	}

	let { role, content, streaming = false }: Props = $props();

	const isUser = $derived(role === 'user');

	// Configure marked for GFM
	marked.setOptions({ breaks: true, gfm: true });

	const renderedContent = $derived(
		isUser ? '' : (marked.parse(content) as string)
	);
</script>

<div class="flex {isUser ? 'justify-end' : 'justify-start'}">
	<div
		class="rounded-lg px-4 py-3 text-sm leading-relaxed max-w-[85%] md:max-w-[70%]
			{isUser ? 'bg-[var(--accent)]/10 text-[var(--text-primary)]' : 'bg-[var(--bg-secondary)] text-[var(--text-primary)]'}"
	>
		{#if isUser}
			<p class="whitespace-pre-wrap break-words">{content}</p>
		{:else}
			<div class="prose">
				{@html renderedContent}
				{#if streaming}
					<span class="ml-1 inline-block w-1.5 h-4 bg-[var(--accent)] animate-pulse align-middle"></span>
				{/if}
			</div>
		{/if}
	</div>
</div>

<style>
	.prose :global(p) { margin: 0.5em 0; }
	.prose :global(p:first-child) { margin-top: 0; }
	.prose :global(p:last-child) { margin-bottom: 0; }
	.prose :global(code) {
		background: var(--bg-tertiary);
		padding: 0.15em 0.4em;
		border-radius: 4px;
		font-size: 0.85em;
	}
	.prose :global(pre) {
		background: var(--bg-primary);
		border: 1px solid var(--border);
		border-radius: 6px;
		padding: 0.75em 1em;
		overflow-x: auto;
		margin: 0.75em 0;
	}
	.prose :global(pre code) {
		background: none;
		padding: 0;
		font-size: 0.85em;
	}
	.prose :global(ul), .prose :global(ol) {
		padding-left: 1.5em;
		margin: 0.5em 0;
	}
	.prose :global(li) { margin: 0.25em 0; }
	.prose :global(blockquote) {
		border-left: 3px solid var(--accent);
		padding-left: 1em;
		margin: 0.5em 0;
		color: var(--text-secondary);
	}
	.prose :global(a) {
		color: var(--accent);
		text-decoration: underline;
	}
	.prose :global(strong) { font-weight: 600; }
	.prose :global(h1), .prose :global(h2), .prose :global(h3) {
		font-weight: 600;
		margin: 0.75em 0 0.25em;
	}
	.prose :global(table) {
		border-collapse: collapse;
		margin: 0.5em 0;
		font-size: 0.85em;
	}
	.prose :global(th), .prose :global(td) {
		border: 1px solid var(--border);
		padding: 0.4em 0.75em;
	}
	.prose :global(th) {
		background: var(--bg-tertiary);
		font-weight: 600;
	}
</style>
