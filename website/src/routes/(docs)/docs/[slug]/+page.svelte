<script lang="ts">
  import { base } from '$app/paths';
  import { siteConfig } from '$lib/config/site.js';
  import { extractToc } from '$lib/content/toc.js';
  import type { DocPage } from '$lib/types/index.js';
  import { onMount } from 'svelte';
  import Prism from 'prismjs';
  import 'prismjs/components/prism-go.js';
  import 'prismjs/components/prism-bash.js';
  import 'prismjs/components/prism-typescript.js';
  import 'prismjs/components/prism-json.js';
  import 'prismjs/components/prism-yaml.js';

  interface Props {
    data: { doc: DocPage; html: string };
  }

  let { data }: Props = $props();

  const toc = $derived(extractToc(data.doc.content));

  onMount(() => {
    // Run Prism highlighting after mount
    Prism.highlightAll();

    // Inject copy buttons into each code block
    document.querySelectorAll('pre').forEach((pre) => {
      const btn = document.createElement('button');
      btn.textContent = 'Copy';
      btn.className = 'copy-btn';
      btn.addEventListener('click', () => {
        const code = pre.querySelector('code');
        if (code) {
          navigator.clipboard.writeText(code.innerText);
          btn.textContent = 'Copied!';
          setTimeout(() => {
            btn.textContent = 'Copy';
          }, 2000);
        }
      });
      pre.style.position = 'relative';
      pre.appendChild(btn);
    });
  });
</script>

<svelte:head>
  <title>{data.doc.title} &mdash; {siteConfig.title}</title>
  <meta name="description" content={data.doc.description} />
  <meta property="og:title" content="{data.doc.title} — {siteConfig.title}" />
  <meta property="og:description" content={data.doc.description} />
</svelte:head>

<div class="flex gap-12">
  <!-- Main content -->
  <article class="prose max-w-none min-w-0 flex-1">
    <!-- Breadcrumb -->
    <nav class="mb-6 flex items-center gap-2 text-sm text-[#848d97] not-prose">
      <a href="{base}/docs" class="hover:text-[#58a6ff]">Docs</a>
      <span>/</span>
      <span class="text-[#9198a1]">{data.doc.title}</span>
    </nav>

    <!-- eslint-disable-next-line svelte/no-at-html-tags -->
    {@html data.html}
  </article>

  <!-- Table of contents (desktop) -->
  {#if toc.length > 0}
    <aside class="hidden w-36 shrink-0 xl:block">
      <div class="sticky top-20">
        <p class="mb-3 text-xs font-semibold uppercase tracking-wider text-[#848d97]">
          On this page
        </p>
        <ul class="space-y-1.5">
          {#each toc as entry}
            <li
              style="padding-left: {(entry.level - 2) * 12}px"
            >
              <a
                href="#{entry.id}"
                class="block text-sm text-[#848d97] hover:text-[#58a6ff] leading-snug"
              >
                {entry.text}
              </a>
            </li>
          {/each}
        </ul>
      </div>
    </aside>
  {/if}
</div>
