<script lang="ts">
	import { page } from '$app/stores';
	import { apiFetch } from '$lib/api';
	import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';
	import ErrorBanner from '$lib/components/ErrorBanner.svelte';

	interface Repo {
		path: string;
		remote?: string;
		default_branch?: string;
	}

	interface Project {
		name: string;
		description?: string;
		repos: Repo[];
		defaults: {
			provider?: string;
			model?: string;
			system_prompt?: string;
			max_agents?: number;
			max_turns?: number;
			max_budget_usd?: number;
			tracker?: {
				type: string;
				owner?: string;
				repo?: string;
				project_number?: number;
			};
		};
		devcontainer?: {
			image?: string;
			dockerfile?: string;
			features?: string[];
		};
		notifications?: {
			teams?: { webhook_url?: string; bidirectional?: boolean };
			local?: { enabled?: boolean; event_types?: string[] };
		};
		workflow?: {
			branch_pattern?: string;
			commit_pattern?: string;
			require_review?: boolean;
			protected_branches?: string[];
		};
		created_at: string;
		updated_at: string;
	}

	const projectName = $derived($page.params.name ?? '');

	let project = $state<Project | null>(null);
	let loading = $state(true);
	let error = $state<string | null>(null);

	async function fetchProject(): Promise<void> {
		try {
			project = await apiFetch<Project>(`/api/project/${projectName}`);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load project';
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		fetchProject();
	});
</script>

<svelte:head>
	<title>{projectName} - Projects - cure</title>
</svelte:head>

<div class="space-y-6">
	<!-- Header -->
	<div class="flex items-center gap-3">
		<a
			href="/project"
			class="rounded-md p-1 text-[rgba(230,237,243,0.5)] transition-colors hover:bg-white/5 hover:text-[#e6edf3]"
			aria-label="Back to projects"
		>
			<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
				<path d="M15 18l-6-6 6-6" />
			</svg>
		</a>
		<h1 class="text-xl font-semibold tracking-tight text-[#e6edf3]">{projectName}</h1>
	</div>

	{#if error}
		<ErrorBanner message={error} onDismiss={() => (error = null)} />
	{/if}

	{#if loading}
		<div class="flex items-center justify-center py-12">
			<LoadingSpinner />
		</div>
	{:else if project}
		<!-- Description -->
		{#if project.description}
			<p class="text-sm text-[rgba(230,237,243,0.5)]">{project.description}</p>
		{/if}

		<div class="grid gap-6 lg:grid-cols-2">
			<!-- Repositories -->
			<section class="rounded-lg border border-white/10 bg-[#161b22] p-5">
				<h2 class="mb-3 text-sm font-semibold uppercase tracking-wider text-[rgba(230,237,243,0.4)]">Repositories</h2>
				<div class="space-y-2">
					{#each project.repos as repo}
						<div class="rounded-md bg-white/5 px-3 py-2">
							<div class="font-mono text-sm text-[#e6edf3]">{repo.path}</div>
							{#if repo.remote}
								<div class="mt-0.5 text-xs text-[rgba(230,237,243,0.3)]">{repo.remote}</div>
							{/if}
							{#if repo.default_branch}
								<div class="mt-0.5 text-xs text-[rgba(230,237,243,0.3)]">branch: {repo.default_branch}</div>
							{/if}
						</div>
					{/each}
				</div>
			</section>

			<!-- Defaults -->
			<section class="rounded-lg border border-white/10 bg-[#161b22] p-5">
				<h2 class="mb-3 text-sm font-semibold uppercase tracking-wider text-[rgba(230,237,243,0.4)]">Defaults</h2>
				<dl class="space-y-2 text-sm">
					{#if project.defaults.provider}
						<div class="flex justify-between"><dt class="text-[rgba(230,237,243,0.4)]">Provider</dt><dd class="text-[#e6edf3]">{project.defaults.provider}</dd></div>
					{/if}
					{#if project.defaults.model}
						<div class="flex justify-between"><dt class="text-[rgba(230,237,243,0.4)]">Model</dt><dd class="font-mono text-[#e6edf3]">{project.defaults.model}</dd></div>
					{/if}
					{#if project.defaults.max_turns}
						<div class="flex justify-between"><dt class="text-[rgba(230,237,243,0.4)]">Max turns</dt><dd class="text-[#e6edf3]">{project.defaults.max_turns}</dd></div>
					{/if}
					{#if project.defaults.max_budget_usd}
						<div class="flex justify-between"><dt class="text-[rgba(230,237,243,0.4)]">Budget</dt><dd class="text-[#e6edf3]">${project.defaults.max_budget_usd}</dd></div>
					{/if}
					{#if project.defaults.max_agents}
						<div class="flex justify-between"><dt class="text-[rgba(230,237,243,0.4)]">Max agents</dt><dd class="text-[#e6edf3]">{project.defaults.max_agents}</dd></div>
					{/if}
				</dl>
				{#if project.defaults.tracker}
					<div class="mt-3 border-t border-white/5 pt-3">
						<h3 class="mb-1 text-xs font-medium text-[rgba(230,237,243,0.3)]">Tracker</h3>
						<div class="text-sm text-[#e6edf3]">{project.defaults.tracker.type}: {project.defaults.tracker.owner}/{project.defaults.tracker.repo}</div>
					</div>
				{/if}
			</section>

			<!-- Workflow -->
			{#if project.workflow}
				<section class="rounded-lg border border-white/10 bg-[#161b22] p-5">
					<h2 class="mb-3 text-sm font-semibold uppercase tracking-wider text-[rgba(230,237,243,0.4)]">Workflow</h2>
					<dl class="space-y-2 text-sm">
						{#if project.workflow.branch_pattern}
							<div><dt class="text-[rgba(230,237,243,0.4)]">Branch pattern</dt><dd class="mt-0.5 font-mono text-xs text-[#e6edf3]">{project.workflow.branch_pattern}</dd></div>
						{/if}
						{#if project.workflow.commit_pattern}
							<div><dt class="text-[rgba(230,237,243,0.4)]">Commit pattern</dt><dd class="mt-0.5 font-mono text-xs text-[#e6edf3]">{project.workflow.commit_pattern}</dd></div>
						{/if}
						{#if project.workflow.protected_branches?.length}
							<div><dt class="text-[rgba(230,237,243,0.4)]">Protected</dt><dd class="mt-0.5 text-[#e6edf3]">{project.workflow.protected_branches.join(', ')}</dd></div>
						{/if}
						{#if project.workflow.require_review}
							<div class="flex justify-between"><dt class="text-[rgba(230,237,243,0.4)]">Review required</dt><dd class="text-[#e6edf3]">Yes</dd></div>
						{/if}
					</dl>
				</section>
			{/if}

			<!-- Notifications -->
			{#if project.notifications}
				<section class="rounded-lg border border-white/10 bg-[#161b22] p-5">
					<h2 class="mb-3 text-sm font-semibold uppercase tracking-wider text-[rgba(230,237,243,0.4)]">Notifications</h2>
					<dl class="space-y-2 text-sm">
						{#if project.notifications.teams}
							<div class="flex justify-between">
								<dt class="text-[rgba(230,237,243,0.4)]">Teams</dt>
								<dd class="text-[#e6edf3]">{project.notifications.teams.webhook_url ? 'Configured' : 'Not configured'}</dd>
							</div>
						{/if}
						{#if project.notifications.local}
							<div class="flex justify-between">
								<dt class="text-[rgba(230,237,243,0.4)]">Local</dt>
								<dd class="text-[#e6edf3]">{project.notifications.local.enabled ? 'Enabled' : 'Disabled'}</dd>
							</div>
						{/if}
					</dl>
				</section>
			{/if}

			<!-- Devcontainer -->
			{#if project.devcontainer}
				<section class="rounded-lg border border-white/10 bg-[#161b22] p-5">
					<h2 class="mb-3 text-sm font-semibold uppercase tracking-wider text-[rgba(230,237,243,0.4)]">Devcontainer</h2>
					<dl class="space-y-2 text-sm">
						{#if project.devcontainer.image}
							<div><dt class="text-[rgba(230,237,243,0.4)]">Image</dt><dd class="mt-0.5 font-mono text-xs text-[#e6edf3]">{project.devcontainer.image}</dd></div>
						{/if}
						{#if project.devcontainer.dockerfile}
							<div><dt class="text-[rgba(230,237,243,0.4)]">Dockerfile</dt><dd class="mt-0.5 font-mono text-xs text-[#e6edf3]">{project.devcontainer.dockerfile}</dd></div>
						{/if}
					</dl>
				</section>
			{/if}
		</div>

		<!-- Timestamps -->
		<div class="flex gap-6 text-xs text-[rgba(230,237,243,0.2)]">
			<span>Created: {new Date(project.created_at).toLocaleDateString()}</span>
			<span>Updated: {new Date(project.updated_at).toLocaleDateString()}</span>
		</div>
	{/if}
</div>
