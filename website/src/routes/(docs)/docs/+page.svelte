<script lang="ts">
  import { base } from '$app/paths';
  import { siteConfig } from '$lib/config/site.js';
  import type { NavSection, DocMeta } from '$lib/types/index.js';

  interface Props {
    data: { sections: NavSection[]; firstDoc: DocMeta | null };
  }

  let { data }: Props = $props();
</script>

<svelte:head>
  <title>Documentation &mdash; {siteConfig.title}</title>
  <meta name="description" content="Documentation for the Cure CLI tool." />
</svelte:head>

<div class="prose max-w-none">
  <h1>Documentation</h1>
  <p>
    Welcome to the Cure CLI documentation. Cure automates repetitive development tasks through AI
    context management, code generation, and network diagnostics.
  </p>

  {#if data.firstDoc}
    <p>
      <a href="{base}/docs/{data.firstDoc.slug}">Get started with installation</a> or browse the
      sections below.
    </p>
  {/if}
</div>

<div class="mt-10 space-y-8">
  {#each data.sections as section}
    <div>
      <h2 class="mb-3 text-sm font-semibold uppercase tracking-wider text-[#848d97]">
        {section.title}
      </h2>
      <div class="grid gap-3 sm:grid-cols-2">
        {#each section.docs as doc}
          <a
            href="{base}/docs/{doc.slug}"
            class="group rounded-lg border border-[#30363d] bg-[#161b22] p-4 hover:border-[#58a6ff]/50"
          >
            <div class="mb-1 font-medium text-[#e6edf3] group-hover:text-[#58a6ff]">
              {doc.title}
            </div>
            {#if doc.description}
              <div class="text-sm text-[#9198a1]">{doc.description}</div>
            {/if}
          </a>
        {/each}
      </div>
    </div>
  {/each}
</div>
