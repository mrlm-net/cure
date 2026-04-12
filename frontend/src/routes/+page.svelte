<script lang="ts">
	import { apiFetch } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import { formatRelativeTime } from '$lib/utils';

	interface Project {
		name: string;
		description?: string;
		repos: { path: string }[];
		defaults: { provider?: string; model?: string };
	}

	interface Session {
		id: string;
		name?: string;
		provider: string;
		project_name?: string;
		updated_at: string;
		turns: number;
	}

	interface DoctorCheck {
		name: string;
		status: string;
		message: string;
	}

	let projects = $state<Project[]>([]);
	let sessions = $state<Session[]>([]);
	let doctorChecks = $state<DoctorCheck[]>([]);
	let loading = $state(true);

	$effect(() => {
		Promise.all([
			apiFetch<Project[]>('/api/project').catch(() => []),
			apiFetch<Session[]>('/api/context/sessions').catch(() => []),
			apiFetch<DoctorCheck[]>('/api/doctor').catch(() => []),
		]).then(([p, s, d]) => {
			projects = p;
			sessions = s;
			doctorChecks = d;
			loading = false;
		});
	});

	const activeProject = $derived(projects.length > 0 ? projects[0] : null);
	const recentSessions = $derived(sessions.slice(0, 5));
	const passedChecks = $derived(doctorChecks.filter(c => c.status === 'pass').length);
	const totalChecks = $derived(doctorChecks.length);
	const healthPct = $derived(totalChecks > 0 ? Math.round((passedChecks / totalChecks) * 100) : 0);
</script>

<svelte:head>
	<title>Dashboard - cure</title>
</svelte:head>

{#if loading}
	<div class="flex items-center justify-center py-20">
		<LoadingSpinner />
	</div>
{:else}
	<div class="space-y-6">
		<!-- Project header -->
		{#if activeProject}
			<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5">
				<div class="flex items-start justify-between">
					<div>
						<h1 class="text-xl font-semibold text-[var(--text-primary)]">{activeProject.name}</h1>
						{#if activeProject.description}
							<p class="mt-1 text-sm text-[var(--text-secondary)]">{activeProject.description}</p>
						{/if}
					</div>
					<a href="/project/{activeProject.name}" class="rounded-md bg-[var(--bg-tertiary)] px-3 py-1.5 text-xs text-[var(--text-secondary)] hover:text-[var(--text-primary)]">
						Settings
					</a>
				</div>
				<div class="mt-3 flex items-center gap-4 text-xs text-[var(--text-tertiary)]">
					<span>{activeProject.repos.length} repo{activeProject.repos.length !== 1 ? 's' : ''}</span>
					{#if activeProject.defaults.provider}
						<span>{activeProject.defaults.provider}</span>
					{/if}
					{#if activeProject.defaults.model}
						<span class="font-mono">{activeProject.defaults.model}</span>
					{/if}
				</div>
			</div>
		{:else}
			<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5 text-center">
				<h1 class="text-xl font-semibold text-[var(--text-primary)]">cure</h1>
				<p class="mt-2 text-sm text-[var(--text-secondary)]">No project configured</p>
				<p class="mt-1 text-xs text-[var(--text-tertiary)]">Run <code class="rounded bg-[var(--bg-tertiary)] px-1.5 py-0.5">cure project init</code> to get started</p>
			</div>
		{/if}

		<!-- Stats row -->
		<div class="grid grid-cols-3 gap-4">
			<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-4">
				<div class="text-xs font-medium uppercase tracking-wider text-[var(--text-tertiary)]">Sessions</div>
				<div class="mt-1 text-2xl font-semibold text-[var(--text-primary)]">{sessions.length}</div>
			</div>
			<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-4">
				<div class="text-xs font-medium uppercase tracking-wider text-[var(--text-tertiary)]">Health</div>
				<div class="mt-1 text-2xl font-semibold {healthPct === 100 ? 'text-[var(--success)]' : healthPct > 50 ? 'text-[var(--warning)]' : 'text-[var(--danger)]'}">{healthPct}%</div>
			</div>
			<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-4">
				<div class="text-xs font-medium uppercase tracking-wider text-[var(--text-tertiary)]">Projects</div>
				<div class="mt-1 text-2xl font-semibold text-[var(--text-primary)]">{projects.length}</div>
			</div>
		</div>

		<div class="grid gap-6 lg:grid-cols-2">
			<!-- Recent sessions -->
			<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5">
				<div class="mb-3 flex items-center justify-between">
					<h2 class="text-sm font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Recent Sessions</h2>
					<a href="/context" class="text-xs text-[var(--accent)] hover:underline">View all</a>
				</div>
				{#if recentSessions.length === 0}
					<p class="py-4 text-center text-sm text-[var(--text-tertiary)]">No sessions yet</p>
				{:else}
					<div class="space-y-2">
						{#each recentSessions as s}
							<a href="/context/{s.id}" class="flex items-center justify-between rounded-md bg-[var(--bg-tertiary)]/50 px-3 py-2 text-sm hover:bg-[var(--bg-tertiary)]">
								<div class="flex items-center gap-2">
									<span class="font-mono text-[var(--accent)]">{s.name || s.id.slice(0, 8)}</span>
									<span class="text-xs text-[var(--text-tertiary)]">{s.provider}</span>
								</div>
								<span class="text-xs text-[var(--text-tertiary)]">{formatRelativeTime(s.updated_at)}</span>
							</a>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Quick actions -->
			<div class="rounded-lg border border-[var(--border)] bg-[var(--bg-secondary)] p-5">
				<h2 class="mb-3 text-sm font-semibold uppercase tracking-wider text-[var(--text-tertiary)]">Quick Actions</h2>
				<div class="grid grid-cols-2 gap-2">
					<a href="/context" class="flex items-center gap-2 rounded-md bg-[var(--accent-subtle)] px-3 py-2.5 text-sm text-[var(--accent)] hover:bg-[var(--accent)]/20">
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z"/></svg>
						Sessions
					</a>
					<a href="/editor" class="flex items-center gap-2 rounded-md bg-[var(--accent-subtle)] px-3 py-2.5 text-sm text-[var(--accent)] hover:bg-[var(--accent)]/20">
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg>
						Editor
					</a>
					<a href="/terminal" class="flex items-center gap-2 rounded-md bg-[var(--accent-subtle)] px-3 py-2.5 text-sm text-[var(--accent)] hover:bg-[var(--accent)]/20">
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
						Terminal
					</a>
					<a href="/doctor" class="flex items-center gap-2 rounded-md bg-[var(--accent-subtle)] px-3 py-2.5 text-sm text-[var(--accent)] hover:bg-[var(--accent)]/20">
						<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"/></svg>
						Doctor
					</a>
				</div>
			</div>
		</div>
	</div>
{/if}
