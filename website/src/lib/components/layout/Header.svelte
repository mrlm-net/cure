<script lang="ts">
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { siteConfig } from '$lib/config/site.js';
  import { topNavLinks } from '$lib/config/navigation.js';

  interface Props {
    onMenuToggle?: () => void;
  }

  let { onMenuToggle }: Props = $props();

  function isActive(href: string): boolean {
    const currentPath = page.url.pathname;
    const fullHref = base + href;
    if (href === '/') return currentPath === base || currentPath === base + '/';
    return currentPath === fullHref || currentPath.startsWith(fullHref + '/');
  }
</script>

<header
  class="fixed top-0 left-0 right-0 z-50 h-14 border-b border-[#30363d] bg-[#0d1117]/95 backdrop-blur-sm"
>
  <div class="flex h-full items-center justify-between px-4 md:pl-64 md:pr-8">
    <!-- Logo / site title -->
    <div class="flex items-center gap-3">
      <!-- Mobile menu button -->
      <button
        type="button"
        class="mr-1 rounded-md p-1.5 text-[#9198a1] hover:bg-[#161b22] hover:text-[#e6edf3] md:hidden"
        onclick={onMenuToggle}
        aria-label="Toggle navigation menu"
      >
        <svg
          class="h-5 w-5"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>

      <a
        href="{base}/"
        class="flex items-center gap-2 font-semibold text-[#e6edf3] hover:text-white"
      >
        <!-- Cure logo / icon -->
        <svg
          class="h-6 w-6 text-[#58a6ff]"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M9 3H5a2 2 0 00-2 2v4m6-6h10a2 2 0 012 2v4M9 3v18m0 0h10a2 2 0 002-2V9M9 21H5a2 2 0 01-2-2V9m0 0h18"
          />
        </svg>
        <span>{siteConfig.title}</span>
      </a>
    </div>

    <!-- Top nav links -->
    <nav class="hidden items-center gap-1 md:flex">
      {#each topNavLinks as link}
        <a
          href="{base}{link.href}"
          class="rounded-md px-3 py-1.5 text-sm transition-colors {isActive(link.href)
            ? 'bg-[#161b22] text-[#e6edf3]'
            : 'text-[#9198a1] hover:bg-[#161b22] hover:text-[#e6edf3]'}"
        >
          {link.label}
        </a>
      {/each}
    </nav>

    <!-- GitHub link -->
    <a
      href={siteConfig.repoUrl}
      target="_blank"
      rel="noopener noreferrer"
      class="rounded-md p-1.5 text-[#9198a1] hover:bg-[#161b22] hover:text-[#e6edf3]"
      aria-label="View on GitHub"
    >
      <svg class="h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
        <path
          d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z"
        />
      </svg>
    </a>
  </div>
</header>
