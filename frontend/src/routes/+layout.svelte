<script lang="ts">
	import '../app.css';
	import type { Snippet } from 'svelte';
	import { page } from '$app/stores';
	import { afterNavigate } from '$app/navigation';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();
	let mobileNavOpen = $state(false);
	let isDesktop = $state(false);

	$effect(() => {
		if (typeof window === 'undefined') return;
		const mql = window.matchMedia('(min-width: 768px)');
		isDesktop = mql.matches;
		const handler = (e: MediaQueryListEvent) => {
			isDesktop = e.matches;
		};
		mql.addEventListener('change', handler);
		return () => mql.removeEventListener('change', handler);
	});

	afterNavigate(() => {
		mobileNavOpen = false;
	});

	interface NavItem {
		href: string;
		label: string;
		icon: string;
	}

	const mainNav: NavItem[] = [
		{
			href: '/',
			label: 'Dashboard',
			icon: 'M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-4 0a1 1 0 01-1-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 01-1 1'
		},
		{
			href: '/project',
			label: 'Projects',
			icon: 'M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z'
		},
		{
			href: '/context',
			label: 'Sessions',
			icon: 'M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z'
		}
	];

	const toolsNav: NavItem[] = [
		{
			href: '/generate',
			label: 'Generate',
			icon: 'M12 4v16m8-8H4'
		},
		{
			href: '/doctor',
			label: 'Doctor',
			icon: 'M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z'
		}
	];

	const bottomNav: NavItem[] = [
		{
			href: '/config',
			label: 'Config',
			icon: 'M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z M15 12a3 3 0 11-6 0 3 3 0 016 0z'
		}
	];

	function isActive(pathname: string, href: string): boolean {
		if (href === '/') return pathname === '/';
		return pathname.startsWith(href);
	}
</script>

{#snippet navLink(item: NavItem)}
	{@const active = isActive($page.url.pathname, item.href)}
	<li>
		<a
			href={item.href}
			class="flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors
				{active
					? 'bg-[rgba(88,166,255,0.15)] text-[#58a6ff]'
					: 'text-[rgba(230,237,243,0.5)] hover:bg-white/5 hover:text-[#e6edf3]'}"
			aria-current={active ? 'page' : undefined}
		>
			<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
				<path d={item.icon} />
			</svg>
			{item.label}
		</a>
	</li>
{/snippet}

<a
	href="#main-content"
	class="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded focus:bg-[#161b22] focus:px-4 focus:py-2 focus:text-[#58a6ff]"
>
	Skip to content
</a>

<!-- Mobile top bar -->
<div class="fixed top-0 left-0 right-0 z-30 flex items-center border-b border-white/10 bg-[#161b22] px-4 py-3 md:hidden">
	<button
		onclick={() => (mobileNavOpen = !mobileNavOpen)}
		class="rounded p-1 text-[#e6edf3] hover:bg-white/10"
		aria-label={mobileNavOpen ? 'Close navigation' : 'Open navigation'}
		aria-expanded={mobileNavOpen}
	>
		<svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
			{#if mobileNavOpen}
				<path d="M6 6l12 12M18 6L6 18" />
			{:else}
				<path d="M4 6h16M4 12h16M4 18h16" />
			{/if}
		</svg>
	</button>
	<span class="ml-3 text-sm font-semibold tracking-wide text-white/80">cure</span>
</div>

<!-- Mobile backdrop -->
{#if mobileNavOpen}
	<button
		class="fixed inset-0 z-30 bg-black/50 md:hidden"
		onclick={() => (mobileNavOpen = false)}
		aria-label="Close navigation"
		tabindex="-1"
	></button>
{/if}

<!-- Sidebar -->
<aside
	class="fixed top-0 left-0 z-40 flex h-full w-56 flex-col bg-[#161b22] transition-transform duration-200 ease-in-out
		{mobileNavOpen ? 'translate-x-0' : '-translate-x-full'} md:translate-x-0"
	aria-hidden={!isDesktop && !mobileNavOpen ? 'true' : undefined}
>
	<!-- Branding -->
	<div class="flex h-14 items-center border-b border-white/10 px-5">
		<span class="text-sm font-semibold tracking-wide text-white/80">cure</span>
	</div>

	<!-- Main navigation -->
	<nav aria-label="Main navigation" class="flex-1 overflow-y-auto px-3 py-4">
		<ul class="space-y-1">
			{#each mainNav as item}
				{@render navLink(item)}
			{/each}
		</ul>

		<!-- Tools section -->
		<div class="mt-6 mb-2 px-3">
			<span class="text-[10px] font-semibold uppercase tracking-widest text-[rgba(230,237,243,0.25)]">Tools</span>
		</div>
		<ul class="space-y-1">
			{#each toolsNav as item}
				{@render navLink(item)}
			{/each}
		</ul>
	</nav>

	<!-- Bottom: Config (separated) -->
	<div class="border-t border-white/10 px-3 py-3">
		<ul class="space-y-1">
			{#each bottomNav as item}
				{@render navLink(item)}
			{/each}
		</ul>
	</div>

	<!-- Version footer -->
	<div class="border-t border-white/10 px-5 py-3">
		<span class="text-xs text-[rgba(230,237,243,0.3)]">cure GUI</span>
	</div>
</aside>

<!-- Main content -->
<div class="min-h-screen md:pl-56">
	<!-- Spacer for mobile top bar -->
	<div class="h-14 md:hidden"></div>

	<main id="main-content" class="p-6">
		{@render children()}
	</main>
</div>
