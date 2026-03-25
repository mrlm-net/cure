<script lang="ts">
  import { page } from '$app/state';
  import { base } from '$app/paths';
  import { siteConfig } from '$lib/config/site.js';
</script>

<svelte:head>
  <title>Error &mdash; {siteConfig.title}</title>
</svelte:head>

<div class="flex min-h-screen flex-col items-center justify-center px-4 text-center">
  <div class="mb-4 text-6xl font-bold text-[#30363d]">{page.status}</div>
  <h1 class="mb-2 text-2xl font-semibold text-[#e6edf3]">
    {page.status === 404 ? 'Page not found' : 'Something went wrong'}
  </h1>
  <p class="mb-8 text-[#9198a1]">
    {page.status === 404
      ? "The page you're looking for doesn't exist."
      : page.error?.message ?? 'An unexpected error occurred.'}
  </p>
  <div class="flex gap-4">
    <a
      href="{base}/"
      class="rounded-lg bg-[#58a6ff] px-4 py-2 text-sm font-medium text-[#0d1117] hover:bg-[#79b8ff]"
    >
      Go home
    </a>
    <a
      href="{base}/docs"
      class="rounded-lg border border-[#30363d] px-4 py-2 text-sm font-medium text-[#9198a1] hover:border-[#58a6ff] hover:text-[#e6edf3]"
    >
      View docs
    </a>
  </div>
</div>
