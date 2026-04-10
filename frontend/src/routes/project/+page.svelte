<script lang="ts">
	import { goto } from '$app/navigation';
	import { apiFetch, getBaseUrl } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';

	interface Project {
		name: string;
		description?: string;
		repos: { path: string; remote?: string }[];
		defaults: { provider?: string; model?: string };
		updated_at: string;
	}

	let projects = $state<Project[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let showCreate = $state(false);

	// Create form
	let newName = $state('');
	let newDesc = $state('');
	let newRepoPath = $state('');
	let newRepoRemote = $state('');
	let newProvider = $state('claude-code');
	let newModel = $state('claude-sonnet-4-6');
	let newTrackerType = $state('github');
	let newTrackerOwner = $state('');
	let newTrackerRepo = $state('');
	let creating = $state(false);

	async function fetchProjects(): Promise<void> {
		try {
			projects = await apiFetch<Project[]>('/api/project');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load projects';
		} finally {
			loading = false;
		}
	}

	async function createProject(): Promise<void> {
		if (!newName || creating) return;
		creating = true;
		error = null;
		try {
			const body: any = {
				name: newName,
				description: newDesc,
				repos: [],
				defaults: {
					provider: newProvider,
					model: newModel,
				}
			};

			if (newRepoPath) {
				const repo: any = { path: newRepoPath };
				if (newRepoRemote) repo.remote = newRepoRemote;
				body.repos.push(repo);
			}

			if (newTrackerType && newTrackerOwner) {
				body.defaults.tracker = {
					type: newTrackerType,
					owner: newTrackerOwner,
					repo: newTrackerRepo
				};
			}

			body.workflow = {
				branch_pattern: '^(feat|fix|docs|refactor|test|chore)/\\d+-.*$',
				commit_pattern: '^(feat|fix|docs|test|refactor|chore)(\\(.+\\))?!?: .+',
				protected_branches: ['main']
			};

			const base = getBaseUrl();
			const res = await fetch(`${base}/api/project`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(body)
			});
			if (!res.ok) {
				const data = await res.json().catch(() => ({ error: 'Create failed' }));
				throw new Error(data.error);
			}
			goto(`/project/${newName}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create project';
		} finally {
			creating = false;
		}
	}

	$effect(() => {
		fetchProjects();
	});
</script>

<svelte:head>
	<title>Projects - cure</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-xl font-semibold tracking-tight text-[var(--text-primary)]">Projects</h1>
		<button
			onclick={() => (showCreate = !showCreate)}
			class="rounded-md bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white hover:opacity-90"
		>
			New Project
		</button>
	</div>

	{#if error}
		<ErrorBanner message={error} onDismiss={() => (error = null)} />
	{/if}

	<!-- Create form -->
	{#if showCreate}
		<div class="space-y-4 rounded-lg border border-[var(--accent)]/30 bg-[var(--bg-secondary)] p-5">
			<h2 class="text-sm font-semibold text-[var(--text-primary)]">Create Project</h2>

			<section class="grid gap-4 sm:grid-cols-2">
				<div>
					<label class="block mb-1 text-xs text-[var(--text-secondary)]">Name *</label>
					<input bind:value={newName} placeholder="my-project"
						class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none" />
					<p class="mt-0.5 text-[10px] text-[var(--text-tertiary)]">lowercase, hyphens allowed</p>
				</div>
				<div>
					<label class="block mb-1 text-xs text-[var(--text-secondary)]">Description</label>
					<input bind:value={newDesc} placeholder="Project description"
						class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none" />
				</div>
			</section>

			<section>
				<h3 class="mb-2 text-xs font-medium text-[var(--text-tertiary)]">Repository</h3>
				<div class="grid gap-4 sm:grid-cols-2">
					<div>
						<label class="block mb-1 text-xs text-[var(--text-secondary)]">Local Path</label>
						<input bind:value={newRepoPath} placeholder="/path/to/repo"
							class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none" />
					</div>
					<div>
						<label class="block mb-1 text-xs text-[var(--text-secondary)]">Remote URL</label>
						<input bind:value={newRepoRemote} placeholder="git@github.com:org/repo.git"
							class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none" />
					</div>
				</div>
			</section>

			<section>
				<h3 class="mb-2 text-xs font-medium text-[var(--text-tertiary)]">AI Provider</h3>
				<div class="grid gap-4 sm:grid-cols-2">
					<div>
						<label class="block mb-1 text-xs text-[var(--text-secondary)]">Provider</label>
						<select bind:value={newProvider}
							class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none">
							<option value="claude-code">Claude Code (CLI)</option>
							<option value="claude">Claude (API)</option>
							<option value="openai">OpenAI</option>
							<option value="gemini">Gemini</option>
						</select>
					</div>
					<div>
						<label class="block mb-1 text-xs text-[var(--text-secondary)]">Model</label>
						<input bind:value={newModel}
							class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none" />
					</div>
				</div>
			</section>

			<section>
				<h3 class="mb-2 text-xs font-medium text-[var(--text-tertiary)]">Tracker</h3>
				<div class="grid gap-4 sm:grid-cols-3">
					<div>
						<label class="block mb-1 text-xs text-[var(--text-secondary)]">Type</label>
						<select bind:value={newTrackerType}
							class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none">
							<option value="">None</option>
							<option value="github">GitHub Issues</option>
							<option value="azdo">Azure DevOps</option>
						</select>
					</div>
					<div>
						<label class="block mb-1 text-xs text-[var(--text-secondary)]">Owner</label>
						<input bind:value={newTrackerOwner} placeholder="org"
							class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none" />
					</div>
					<div>
						<label class="block mb-1 text-xs text-[var(--text-secondary)]">Repo</label>
						<input bind:value={newTrackerRepo} placeholder="repo"
							class="w-full rounded-md border border-[var(--border)] bg-[var(--bg-primary)] px-3 py-2 text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none" />
					</div>
				</div>
			</section>

			<div class="flex justify-end gap-2">
				<button onclick={() => (showCreate = false)}
					class="rounded-md bg-[var(--bg-tertiary)] px-4 py-2 text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)]">
					Cancel
				</button>
				<button onclick={createProject} disabled={!newName || creating}
					class="rounded-md bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white disabled:opacity-50">
					{creating ? 'Creating...' : 'Create Project'}
				</button>
			</div>
		</div>
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<LoadingSpinner />
		</div>
	{:else if projects.length === 0 && !showCreate}
		<div class="flex flex-col items-center justify-center py-16 text-center">
			<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--text-tertiary)" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" class="mb-4">
				<path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
			</svg>
			<p class="text-sm text-[var(--text-secondary)]">No projects yet</p>
			<p class="mt-1 text-xs text-[var(--text-tertiary)]">Click "New Project" to get started</p>
		</div>
	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each projects as project (project.name)}
				<a
					href="/project/{project.name}"
					class="group rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5 transition-colors hover:border-[var(--accent)]/30"
				>
					<div class="flex items-start justify-between">
						<h2 class="font-medium text-[var(--text-primary)] group-hover:text-[var(--accent)]">{project.name}</h2>
						<span class="rounded bg-[var(--bg-tertiary)] px-2 py-0.5 text-xs text-[var(--text-secondary)]">
							{project.repos.length} repo{project.repos.length !== 1 ? 's' : ''}
						</span>
					</div>
					{#if project.description}
						<p class="mt-2 text-sm text-[var(--text-secondary)] line-clamp-2">{project.description}</p>
					{/if}
					<div class="mt-3 flex items-center gap-3 text-xs text-[var(--text-tertiary)]">
						{#if project.defaults.provider}
							<span>{project.defaults.provider}</span>
						{/if}
						{#if project.defaults.model}
							<span class="font-mono">{project.defaults.model}</span>
						{/if}
					</div>
				</a>
			{/each}
		</div>
	{/if}
</div>
