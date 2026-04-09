<script lang="ts">
	import { apiFetch, getBaseUrl } from '$lib/api';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';

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

	const dirty = $derived(fileContent !== originalContent && currentFile !== '');

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
		{:else if currentFile}
			<textarea
				bind:value={fileContent}
				class="flex-1 w-full resize-none border-0 bg-[var(--bg-primary)] p-4 font-mono text-sm leading-6 text-[var(--text-primary)] focus:outline-none"
				spellcheck="false"
				wrap="off"
			></textarea>
		{:else}
			<div class="flex flex-1 items-center justify-center text-sm text-[var(--text-tertiary)]">
				Select a file from the sidebar to edit
			</div>
		{/if}
	</div>
</div>
