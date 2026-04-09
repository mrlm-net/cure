<script lang="ts">
	import { onMount } from 'svelte';
	import { apiFetch, getBaseUrl } from '$lib/api';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import { getTheme } from '$lib/theme';

	interface FileEntry {
		name: string;
		is_dir: boolean;
		size: number;
	}

	let editorContainer: HTMLDivElement | undefined = $state();
	let editor: any = $state(null);
	let monacoModule: any = $state(null);
	let currentPath = $state('');
	let currentFile = $state('');
	let files = $state<FileEntry[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let saving = $state(false);

	async function fetchFiles(path: string): Promise<void> {
		try {
			const data = await apiFetch<FileEntry[]>(`/api/files?path=${encodeURIComponent(path)}`);
			files = data;
			currentPath = path;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load files';
		}
	}

	async function openFile(path: string): Promise<void> {
		try {
			const base = getBaseUrl();
			const res = await fetch(`${base}/api/files/${encodeURIComponent(path)}`);
			if (!res.ok) throw new Error(`Failed to load file: ${res.status}`);
			const content = await res.text();
			currentFile = path;

			if (editor && monacoModule) {
				const ext = path.split('.').pop() || '';
				const lang = extToLanguage(ext);
				const model = monacoModule.editor.createModel(content, lang);
				editor.setModel(model);
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to open file';
		}
	}

	async function saveFile(): Promise<void> {
		if (!editor || !currentFile) return;
		saving = true;
		try {
			const content = editor.getValue();
			const base = getBaseUrl();
			const res = await fetch(`${base}/api/files/${encodeURIComponent(currentFile)}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'text/plain' },
				body: content
			});
			if (!res.ok) throw new Error(`Save failed: ${res.status}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}

	function extToLanguage(ext: string): string {
		const map: Record<string, string> = {
			go: 'go', ts: 'typescript', tsx: 'typescript', js: 'javascript', jsx: 'javascript',
			py: 'python', rs: 'rust', java: 'java', json: 'json', yaml: 'yaml', yml: 'yaml',
			md: 'markdown', html: 'html', css: 'css', svelte: 'html', sh: 'shell',
			toml: 'ini', sql: 'sql', xml: 'xml', dockerfile: 'dockerfile'
		};
		return map[ext.toLowerCase()] || 'plaintext';
	}

	function handleKeydown(e: KeyboardEvent) {
		if ((e.metaKey || e.ctrlKey) && e.key === 's') {
			e.preventDefault();
			saveFile();
		}
	}

	onMount(async () => {
		loading = true;
		try {
			const monaco = await import('monaco-editor');
			monacoModule = monaco;

			if (editorContainer) {
				const theme = getTheme();
				editor = monaco.editor.create(editorContainer, {
					value: '// Open a file from the sidebar',
					language: 'plaintext',
					theme: theme === 'dark' ? 'vs-dark' : 'vs',
					minimap: { enabled: false },
					fontSize: 13,
					lineNumbers: 'on',
					renderWhitespace: 'selection',
					tabSize: 4,
					automaticLayout: true,
				});
			}
			await fetchFiles('.');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load editor';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Editor - cure</title>
</svelte:head>

<svelte:window onkeydown={handleKeydown} />

<div class="-m-6 flex h-[calc(100vh-3.5rem)] md:h-screen">
	<!-- File browser sidebar -->
	<div class="w-56 shrink-0 overflow-y-auto border-r border-[var(--border)] bg-[var(--bg-secondary)] p-2">
		<div class="mb-2 px-2 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">
			Files
		</div>
		{#if currentPath && currentPath !== '.'}
			<button
				onclick={() => fetchFiles(currentPath.split('/').slice(0, -1).join('/') || '.')}
				class="mb-1 flex w-full items-center gap-2 rounded px-2 py-1 text-xs text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]"
			>
				..
			</button>
		{/if}
		{#each files as f}
			{#if f.is_dir}
				<button
					onclick={() => fetchFiles(currentPath === '.' ? f.name : `${currentPath}/${f.name}`)}
					class="flex w-full items-center gap-2 rounded px-2 py-1 text-sm text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)]"
				>
					<span class="text-[var(--accent)]">+</span> {f.name}
				</button>
			{:else}
				<button
					onclick={() => openFile(currentPath === '.' ? f.name : `${currentPath}/${f.name}`)}
					class="flex w-full items-center gap-2 rounded px-2 py-1 text-sm text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]
						{currentFile.endsWith(f.name) ? 'bg-[var(--accent-subtle)]' : ''}"
				>
					{f.name}
				</button>
			{/if}
		{/each}
	</div>

	<!-- Editor -->
	<div class="flex flex-1 flex-col">
		<!-- Tab bar -->
		<div class="flex h-9 items-center gap-2 border-b border-[var(--border)] bg-[var(--bg-secondary)] px-3">
			{#if currentFile}
				<span class="text-xs text-[var(--text-secondary)]">{currentFile}</span>
				{#if saving}
					<span class="text-xs text-[var(--warning)]">Saving...</span>
				{/if}
			{:else}
				<span class="text-xs text-[var(--text-tertiary)]">No file open</span>
			{/if}
		</div>

		{#if error}
			<div class="p-4">
				<ErrorBanner message={error} onDismiss={() => (error = null)} />
			</div>
		{/if}

		{#if loading}
			<div class="flex flex-1 items-center justify-center">
				<LoadingSpinner />
			</div>
		{:else}
			<div bind:this={editorContainer} class="flex-1"></div>
		{/if}
	</div>
</div>
