<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { apiFetch, getBaseUrl } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';
	import { formatRelativeTime } from '$lib/utils';

	interface Session {
		id: string;
		name?: string;
		provider: string;
		project_name?: string;
		branch_name?: string;
		updated_at: string;
		turns: number;
	}

	const projectName = $derived($page.params.name ?? '');

	let projectJson = $state('');
	let originalJson = $state('');
	let sessions = $state<Session[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let saving = $state(false);
	let tab = $state<'dashboard' | 'sessions' | 'tools' | 'config'>('dashboard');
	let dirty = $derived(projectJson !== originalJson);

	const projectData = $derived.by(() => {
		try { return JSON.parse(projectJson); } catch { return null; }
	});

	async function fetchData(): Promise<void> {
		try {
			const [proj, allSessions] = await Promise.all([
				apiFetch<any>(`/api/project/${projectName}`),
				apiFetch<Session[]>('/api/context/sessions').catch(() => [])
			]);
			const json = JSON.stringify(proj, null, 2);
			projectJson = json;
			originalJson = json;
			sessions = allSessions.filter(s => s.project_name === projectName);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load project';
		} finally {
			loading = false;
		}
	}

	async function saveProject(): Promise<void> {
		saving = true;
		error = null;
		try {
			const parsed = JSON.parse(projectJson);
			const base = getBaseUrl();
			const res = await fetch(`${base}/api/project/${projectName}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: projectJson
			});
			if (!res.ok) throw new Error(`Save failed: ${res.status}`);
			originalJson = projectJson;
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}

	async function createSession(): Promise<void> {
		try {
			const data = await apiFetch<{ id: string }>('/api/context/sessions', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ project_name: projectName })
			});
			if (data?.id) goto(`/context/${data.id}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create session';
		}
	}

	$effect(() => {
		fetchData();
	});
</script>

<svelte:head>
	<title>{projectName} - Projects - cure</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center py-20"><LoadingSpinner /></div>
{:else}
	<div class="space-y-4">
		<!-- Header -->
		<div class="flex items-center gap-3">
			<a href="/project" class="rounded-md p-1 text-[var(--text-secondary)] hover:bg-[var(--bg-tertiary)] hover:text-[var(--text-primary)]">
				<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M15 18l-6-6 6-6"/></svg>
			</a>
			<h1 class="text-xl font-semibold text-[var(--text-primary)]">{projectName}</h1>
		</div>

		{#if error}
			<ErrorBanner message={error} onDismiss={() => (error = null)} />
		{/if}

		<!-- Tabs -->
		<div class="flex gap-1 border-b border-[var(--border)]">
			<button
				onclick={() => (tab = 'dashboard')}
				class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
					{tab === 'dashboard' ? 'border-[var(--accent)] text-[var(--accent)]' : 'border-transparent text-[var(--text-secondary)] hover:text-[var(--text-primary)]'}"
			>
				Dashboard
			</button>
			<button
				onclick={() => (tab = 'sessions')}
				class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
					{tab === 'sessions' ? 'border-[var(--accent)] text-[var(--accent)]' : 'border-transparent text-[var(--text-secondary)] hover:text-[var(--text-primary)]'}"
			>
				Sessions ({sessions.length})
			</button>
			<button
				onclick={() => (tab = 'tools')}
				class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
					{tab === 'tools' ? 'border-[var(--accent)] text-[var(--accent)]' : 'border-transparent text-[var(--text-secondary)] hover:text-[var(--text-primary)]'}"
			>
				Tools
			</button>
			<button
				onclick={() => (tab = 'config')}
				class="px-4 py-2 text-sm font-medium border-b-2 transition-colors
					{tab === 'config' ? 'border-[var(--accent)] text-[var(--accent)]' : 'border-transparent text-[var(--text-secondary)] hover:text-[var(--text-primary)]'}"
			>
				Configuration {dirty ? '*' : ''}
			</button>
		</div>

		<!-- Dashboard tab -->
		{#if tab === 'dashboard' && projectData}
			<div class="space-y-4">
				{#if projectData.description}
					<p class="text-sm text-[var(--text-secondary)]">{projectData.description}</p>
				{/if}

				<!-- Stats -->
				<div class="grid grid-cols-3 gap-4">
					<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-4">
						<div class="text-xs font-medium uppercase tracking-wider text-[var(--text-tertiary)]">Repos</div>
						<div class="mt-1 text-2xl font-semibold text-[var(--text-primary)]">{projectData.repos?.length ?? 0}</div>
					</div>
					<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-4">
						<div class="text-xs font-medium uppercase tracking-wider text-[var(--text-tertiary)]">Sessions</div>
						<div class="mt-1 text-2xl font-semibold text-[var(--text-primary)]">{sessions.length}</div>
					</div>
					<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-4">
						<div class="text-xs font-medium uppercase tracking-wider text-[var(--text-tertiary)]">Provider</div>
						<div class="mt-1 text-lg font-semibold text-[var(--text-primary)]">{projectData.defaults?.provider ?? '—'}</div>
					</div>
				</div>

				<div class="grid gap-4 lg:grid-cols-2">
					<!-- Repos -->
					<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5">
						<h3 class="mb-3 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Repositories</h3>
						{#each projectData.repos ?? [] as repo}
							<div class="rounded-md bg-[var(--bg-tertiary)]/50 px-3 py-2 mb-2 font-mono text-sm text-[var(--text-primary)]">
								{repo.path}
								{#if repo.remote}
									<div class="text-xs text-[var(--text-tertiary)] mt-0.5">{repo.remote}</div>
								{/if}
							</div>
						{/each}
					</div>

					<!-- Defaults -->
					<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5">
						<h3 class="mb-3 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Defaults</h3>
						<dl class="space-y-2 text-sm">
							{#if projectData.defaults?.model}
								<div class="flex justify-between"><dt class="text-[var(--text-secondary)]">Model</dt><dd class="font-mono text-[var(--text-primary)]">{projectData.defaults.model}</dd></div>
							{/if}
							{#if projectData.defaults?.max_turns}
								<div class="flex justify-between"><dt class="text-[var(--text-secondary)]">Max turns</dt><dd class="text-[var(--text-primary)]">{projectData.defaults.max_turns}</dd></div>
							{/if}
							{#if projectData.defaults?.tracker}
								<div class="flex justify-between"><dt class="text-[var(--text-secondary)]">Tracker</dt><dd class="text-[var(--text-primary)]">{projectData.defaults.tracker.type}: {projectData.defaults.tracker.owner}/{projectData.defaults.tracker.repo}</dd></div>
							{/if}
						</dl>
					</div>

					<!-- Workflow -->
					{#if projectData.workflow}
						<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5">
							<h3 class="mb-3 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Workflow</h3>
							<dl class="space-y-2 text-sm">
								{#if projectData.workflow.branch_pattern}
									<div><dt class="text-[var(--text-secondary)]">Branch</dt><dd class="mt-0.5 font-mono text-xs text-[var(--text-primary)]">{projectData.workflow.branch_pattern}</dd></div>
								{/if}
								{#if projectData.workflow.commit_pattern}
									<div><dt class="text-[var(--text-secondary)]">Commit</dt><dd class="mt-0.5 font-mono text-xs text-[var(--text-primary)]">{projectData.workflow.commit_pattern}</dd></div>
								{/if}
								{#if projectData.workflow.protected_branches?.length}
									<div class="flex justify-between"><dt class="text-[var(--text-secondary)]">Protected</dt><dd class="text-[var(--text-primary)]">{projectData.workflow.protected_branches.join(', ')}</dd></div>
								{/if}
							</dl>
						</div>
					{/if}

					<!-- Notifications -->
					{#if projectData.notifications}
						<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5">
							<h3 class="mb-3 text-xs font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Notifications</h3>
							<dl class="space-y-2 text-sm">
								{#if projectData.notifications.local}
									<div class="flex justify-between"><dt class="text-[var(--text-secondary)]">Local</dt><dd class="text-[var(--text-primary)]">{projectData.notifications.local.enabled ? 'Enabled' : 'Disabled'}</dd></div>
								{/if}
								{#if projectData.notifications.teams}
									<div class="flex justify-between"><dt class="text-[var(--text-secondary)]">Teams</dt><dd class="text-[var(--text-primary)]">{projectData.notifications.teams.webhook_url ? 'Configured' : 'Not set'}</dd></div>
								{/if}
							</dl>
						</div>
					{/if}
				</div>

				<div class="text-xs text-[var(--text-tertiary)]">
					Created: {new Date(projectData.created_at).toLocaleDateString()} · Updated: {new Date(projectData.updated_at).toLocaleDateString()}
				</div>
			</div>
		{/if}

		<!-- Sessions tab -->
		{#if tab === 'sessions'}
			<div class="space-y-3">
				<div class="flex justify-end">
					<button
						onclick={createSession}
						class="rounded-md bg-[var(--accent)] px-4 py-2 text-sm font-medium text-white hover:opacity-90"
					>
						New Session
					</button>
				</div>

				{#if sessions.length === 0}
					<div class="py-12 text-center">
						<p class="text-sm text-[var(--text-secondary)]">No sessions for this project</p>
					</div>
				{:else}
					{#each sessions as s (s.id)}
						<a
							href="/context/{s.id}"
							class="flex items-center justify-between rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] px-4 py-3 hover:border-[var(--accent)]/30"
						>
							<div>
								<div class="flex items-center gap-2">
									<span class="font-mono text-sm text-[var(--accent)]">{s.name || s.id.slice(0, 8)}</span>
									<span class="rounded bg-[var(--bg-tertiary)] px-2 py-0.5 text-xs text-[var(--text-secondary)]">{s.provider}</span>
								</div>
								<div class="mt-1 flex gap-3 text-xs text-[var(--text-tertiary)]">
									<span>{formatRelativeTime(s.updated_at)}</span>
									<span>{s.turns} turns</span>
									{#if s.branch_name}
										<span class="font-mono">{s.branch_name}</span>
									{/if}
								</div>
							</div>
						</a>
					{/each}
				{/if}
			</div>
		{/if}

		<!-- Config tab (editable JSON) -->
		<!-- Tools tab -->
		{#if tab === 'tools'}
			<div class="grid gap-4 sm:grid-cols-2">
				<a href="/editor" class="group rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5 transition-colors hover:border-[var(--accent)]/30">
					<div class="flex items-center gap-3 mb-2">
						<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
						<h3 class="text-sm font-medium text-[var(--text-primary)] group-hover:text-[var(--accent)]">Editor</h3>
					</div>
					<p class="text-xs text-[var(--text-tertiary)]">Browse and edit files in this project's repositories</p>
				</a>
				<a href="/terminal" class="group rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5 transition-colors hover:border-[var(--accent)]/30">
					<div class="flex items-center gap-3 mb-2">
						<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="var(--accent)" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
						<h3 class="text-sm font-medium text-[var(--text-primary)] group-hover:text-[var(--accent)]">Terminal</h3>
					</div>
					<p class="text-xs text-[var(--text-tertiary)]">Shell session in this project's working directory</p>
				</a>
			</div>
		{/if}

		<!-- Config tab -->
		{#if tab === 'config'}
			<div class="space-y-3">
				<div class="flex items-center justify-between">
					<span class="text-xs text-[var(--text-tertiary)]">~/.cure/projects/{projectName}/project.json</span>
					<div class="flex gap-2">
						{#if dirty}
							<button
								onclick={() => (projectJson = originalJson)}
								class="rounded-md bg-[var(--bg-tertiary)] px-3 py-1.5 text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)]"
							>
								Discard
							</button>
						{/if}
						<button
							onclick={saveProject}
							disabled={!dirty || saving}
							class="rounded-md bg-[var(--accent)] px-3 py-1.5 text-xs text-white disabled:opacity-50"
						>
							{saving ? 'Saving...' : 'Save'}
						</button>
					</div>
				</div>

				<textarea
					bind:value={projectJson}
					class="w-full rounded-lg border border-[var(--border)] bg-[var(--bg-primary)] p-4 font-mono text-sm text-[var(--text-primary)] focus:border-[var(--accent)]/50 focus:outline-none"
					rows={25}
					spellcheck="false"
				></textarea>
			</div>
		{/if}
	</div>
{/if}
