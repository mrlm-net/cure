<script lang="ts">
	import { apiFetch } from '$lib/api';
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

	async function fetchProjects(): Promise<void> {
		try {
			projects = await apiFetch<Project[]>('/api/project');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load projects';
		} finally {
			loading = false;
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
	</div>

	{#if error}
		<ErrorBanner message={error} onDismiss={() => (error = null)} />
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<LoadingSpinner />
		</div>

	{:else if projects.length === 0}
		<div class="flex flex-col items-center justify-center py-16 text-center">
			<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="var(--text-tertiary)" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" class="mb-4">
				<path d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
			</svg>
			<p class="text-sm text-[var(--text-secondary)]">No projects configured</p>
			<p class="mt-1 text-xs text-[var(--text-tertiary)]">Create one with <code class="rounded bg-[var(--bg-tertiary)]/50 px-1.5 py-0.5">cure project init</code></p>
		</div>

	{:else}
		<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
			{#each projects as project (project.name)}
				<a
					href="/project/{project.name}"
					class="group rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5 transition-colors hover:border-[#58a6ff]/30 hover:bg-[var(--bg-secondary)]/80"
				>
					<div class="flex items-start justify-between">
						<h2 class="font-medium text-[var(--text-primary)] group-hover:text-[var(--accent)]">
							{project.name}
						</h2>
						<span class="rounded bg-[var(--bg-tertiary)]/50 px-2 py-0.5 text-xs text-[var(--text-secondary)]">
							{project.repos.length} repo{project.repos.length !== 1 ? 's' : ''}
						</span>
					</div>
					{#if project.description}
						<p class="mt-2 text-sm text-[var(--text-secondary)] line-clamp-2">
							{project.description}
						</p>
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
