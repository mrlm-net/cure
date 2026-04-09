<script lang="ts">
	import { apiFetch, getBaseUrl } from '$lib/api';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import hljs from 'highlight.js/lib/core';
	import go from 'highlight.js/lib/languages/go';
	import typescript from 'highlight.js/lib/languages/typescript';
	import javascript from 'highlight.js/lib/languages/javascript';
	import python from 'highlight.js/lib/languages/python';
	import bash from 'highlight.js/lib/languages/bash';
	import json_lang from 'highlight.js/lib/languages/json';
	import yaml from 'highlight.js/lib/languages/yaml';
	import rust from 'highlight.js/lib/languages/rust';
	import sql from 'highlight.js/lib/languages/sql';
	import xml from 'highlight.js/lib/languages/xml';
	import css_lang from 'highlight.js/lib/languages/css';
	import markdown from 'highlight.js/lib/languages/markdown';

	hljs.registerLanguage('go', go);
	hljs.registerLanguage('typescript', typescript);
	hljs.registerLanguage('ts', typescript);
	hljs.registerLanguage('javascript', javascript);
	hljs.registerLanguage('js', javascript);
	hljs.registerLanguage('python', python);
	hljs.registerLanguage('py', python);
	hljs.registerLanguage('bash', bash);
	hljs.registerLanguage('sh', bash);
	hljs.registerLanguage('json', json_lang);
	hljs.registerLanguage('yaml', yaml);
	hljs.registerLanguage('yml', yaml);
	hljs.registerLanguage('rust', rust);
	hljs.registerLanguage('rs', rust);
	hljs.registerLanguage('sql', sql);
	hljs.registerLanguage('html', xml);
	hljs.registerLanguage('xml', xml);
	hljs.registerLanguage('svelte', xml);
	hljs.registerLanguage('css', css_lang);
	hljs.registerLanguage('md', markdown);
	hljs.registerLanguage('markdown', markdown);

	interface FileEntry {
		name: string;
		is_dir: boolean;
		size: number;
	}

	let currentPath = $state('');
	let currentFile = $state('');
	let fileContent = $state('');
	let originalContent = $state('');
	let files = $state<FileEntry[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let saving = $state(false);
	let editing = $state(false);

	const dirty = $derived(fileContent !== originalContent && currentFile !== '');

	function getLang(path: string): string | undefined {
		const ext = path.split('.').pop()?.toLowerCase() || '';
		return hljs.getLanguage(ext) ? ext : undefined;
	}

	const highlightedContent = $derived.by(() => {
		if (!currentFile || !fileContent) return '';
		const lang = getLang(currentFile);
		if (lang) {
			return hljs.highlight(fileContent, { language: lang }).value;
		}
		try { return hljs.highlightAuto(fileContent).value; } catch { return fileContent; }
	});

	async function fetchFiles(path: string): Promise<void> {
		try {
			const url = path && path !== '.' ? `/api/files?path=${encodeURIComponent(path)}` : '/api/files';
			files = await apiFetch<FileEntry[]>(url);
			currentPath = path || '.';
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load files';
		} finally {
			loading = false;
		}
	}

	async function openFile(name: string): Promise<void> {
		const path = currentPath && currentPath !== '.' ? `${currentPath}/${name}` : name;
		try {
			const base = getBaseUrl();
			const res = await fetch(`${base}/api/files/${path}`);
			if (!res.ok) throw new Error(`Failed to load: ${res.status}`);
			const content = await res.text();
			currentFile = path;
			fileContent = content;
			originalContent = content;
			editing = false;
			error = null;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to open file';
		}
	}

	async function openDir(name: string): Promise<void> {
		const path = currentPath && currentPath !== '.' ? `${currentPath}/${name}` : name;
		await fetchFiles(path);
	}

	async function goUp(): Promise<void> {
		if (!currentPath || currentPath === '.') return;
		const parts = currentPath.split('/');
		parts.pop();
		await fetchFiles(parts.join('/') || '.');
	}

	async function saveFile(): Promise<void> {
		if (!currentFile || !dirty) return;
		saving = true;
		try {
			const base = getBaseUrl();
			const res = await fetch(`${base}/api/files/${currentFile}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'text/plain' },
				body: fileContent
			});
			if (!res.ok) throw new Error(`Save failed: ${res.status}`);
			originalContent = fileContent;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if ((e.metaKey || e.ctrlKey) && e.key === 's') {
			e.preventDefault();
			saveFile();
		}
	}

	$effect(() => {
		fetchFiles('.');
	});
</script>

<svelte:head>
	<title>Editor - cure</title>
</svelte:head>

<svelte:window onkeydown={handleKeydown} />

<div class="-m-6 flex h-[calc(100vh-3.5rem)] md:h-screen">
	<!-- File browser -->
	<div class="w-56 shrink-0 overflow-y-auto border-r border-[var(--border)] bg-[var(--bg-secondary)] p-2">
		<div class="mb-2 px-2 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Files</div>
		{#if currentPath && currentPath !== '.'}
			<button onclick={goUp} class="mb-1 flex w-full items-center gap-1 rounded px-2 py-1 text-xs text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]">
				..
			</button>
		{/if}
		{#each files as f}
			{#if f.is_dir}
				<button
					onclick={() => openDir(f.name)}
					class="flex w-full items-center gap-1 rounded px-2 py-1 text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]"
				>
					<span class="text-[var(--accent)]">+</span> {f.name}
				</button>
			{:else}
				<button
					onclick={() => openFile(f.name)}
					class="flex w-full items-center rounded px-2 py-1 text-sm hover:bg-[var(--bg-tertiary)]
						{currentFile.endsWith('/' + f.name) || currentFile === f.name ? 'bg-[var(--accent-subtle)] text-[var(--accent)]' : 'text-[var(--text-primary)]'}"
				>
					{f.name}
				</button>
			{/if}
		{/each}
	</div>

	<!-- Editor area -->
	<div class="flex flex-1 flex-col">
		<!-- Tab bar -->
		<div class="flex h-9 items-center justify-between border-b border-[var(--border)] bg-[var(--bg-secondary)] px-3">
			<div class="flex items-center gap-2">
				{#if currentFile}
					<span class="text-xs text-[var(--text-secondary)]">{currentFile}</span>
					{#if dirty}
						<span class="text-xs text-[var(--warning)]">(modified)</span>
					{/if}
					{#if saving}
						<span class="text-xs text-[var(--accent)]">Saving...</span>
					{/if}
				{:else}
					<span class="text-xs text-[var(--text-tertiary)]">Select a file</span>
				{/if}
			</div>
			{#if dirty}
				<button onclick={saveFile} class="rounded bg-[var(--accent)] px-2 py-0.5 text-xs text-white">Save</button>
			{/if}
		</div>

		{#if error}
			<div class="p-4"><ErrorBanner message={error} onDismiss={() => (error = null)} /></div>
		{/if}

		{#if loading}
			<div class="flex flex-1 items-center justify-center"><LoadingSpinner /></div>
		{:else if currentFile && editing}
			<textarea
				bind:value={fileContent}
				class="flex-1 w-full resize-none border-0 bg-[var(--bg-primary)] p-4 font-mono text-sm leading-6 text-[var(--text-primary)] outline-none ring-0"
				spellcheck="false"
				wrap="off"
				style="tab-size: 4;"
			></textarea>
		{:else if currentFile}
			<!-- svelte-ignore a11y_click_events_have_key_events -->
			<!-- svelte-ignore a11y_no_static_element_interactions -->
			<div
				onclick={() => (editing = true)}
				class="flex-1 overflow-auto cursor-text"
			>
				<pre class="m-0 p-4 font-mono text-sm leading-6 whitespace-pre overflow-x-auto"><code class="hljs">{@html highlightedContent}</code></pre>
			</div>
		{:else}
			<div class="flex flex-1 items-center justify-center text-sm text-[var(--text-tertiary)]">
				Select a file from the sidebar to edit
			</div>
		{/if}
	</div>
</div>

<style>
	:global(.hljs-keyword) { color: #ff7b72; }
	:global(.hljs-string) { color: #a5d6ff; }
	:global(.hljs-number) { color: #79c0ff; }
	:global(.hljs-function) { color: #d2a8ff; }
	:global(.hljs-title) { color: #d2a8ff; }
	:global(.hljs-params) { color: #e6edf3; }
	:global(.hljs-comment) { color: #8b949e; font-style: italic; }
	:global(.hljs-type) { color: #79c0ff; }
	:global(.hljs-built_in) { color: #ffa657; }
	:global(.hljs-literal) { color: #79c0ff; }
	:global(.hljs-attr) { color: #79c0ff; }
	:global(.hljs-meta) { color: #8b949e; }
	:global(.hljs-variable) { color: #ffa657; }
	:global(.hljs-property) { color: #79c0ff; }
	:global(.hljs-selector-class) { color: #7ee787; }
	:global(.hljs-selector-tag) { color: #7ee787; }
	:global(.hljs-section) { color: #79c0ff; font-weight: 600; }
	:global(.hljs-bullet) { color: #ffa657; }
	:global(.hljs-emphasis) { font-style: italic; }
	:global(.hljs-strong) { font-weight: 700; }
</style>
