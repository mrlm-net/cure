<script lang="ts">
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import type { NavSection } from '$lib/types/index.js';

  interface Props {
    sections: NavSection[];
    mobileOpen?: boolean;
    onClose?: () => void;
  }

  let { sections, mobileOpen = false, onClose }: Props = $props();

  // Track which sections are open — initialised from defaultOpen.
  // Using a plain object in $state avoids the Svelte 5 warning about
  // capturing a prop reference in $state initialiser.
  let openSections = $state<Record<string, boolean>>({});

  $effect(() => {
    for (const s of sections) {
      if (!(s.id in openSections)) {
        openSections[s.id] = s.defaultOpen ?? true;
      }
    }
  });

  function toggleSection(id: string) {
    openSections[id] = !openSections[id];
  }

  function isActive(slug: string): boolean {
    const currentPath = page.url.pathname;
    const docPath = `${base}/docs/${slug}`;
    return currentPath === docPath;
  }

  function handleOverlayClick() {
    onClose?.();
  }
</script>

<!-- Mobile overlay -->
{#if mobileOpen}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-40 bg-black/60 md:hidden"
    onclick={handleOverlayClick}
    aria-hidden="true"
  ></div>
{/if}

<!-- Sidebar panel -->
<aside
  class="fixed top-14 bottom-0 left-0 z-40 w-64 overflow-y-auto border-r border-[#30363d] bg-[#0d1117] transition-transform duration-200 md:translate-x-0 {mobileOpen
    ? 'translate-x-0'
    : '-translate-x-full'}"
>
  <div class="py-4">
    <!-- Close button (mobile) -->
    {#if mobileOpen}
      <div class="flex justify-end px-4 pb-2 md:hidden">
        <button
          type="button"
          class="rounded-md p-1 text-[#9198a1] hover:text-[#e6edf3]"
          onclick={onClose}
          aria-label="Close navigation menu"
        >
          <svg class="h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    {/if}

    {#each sections as section}
      <div class="mb-2">
        <!-- Section header -->
        <button
          type="button"
          class="flex w-full items-center justify-between px-4 py-1.5 text-left"
          onclick={() => toggleSection(section.id)}
        >
          <span class="text-xs font-semibold uppercase tracking-wider text-[#848d97]">
            {section.title}
          </span>
          <svg
            class="h-3.5 w-3.5 text-[#848d97] transition-transform duration-150 {openSections[section.id]
              ? 'rotate-0'
              : '-rotate-90'}"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        <!-- Section docs -->
        {#if openSections[section.id]}
          <ul class="mt-1">
            {#each section.docs as doc}
              <li>
                <a
                  href="{base}/docs/{doc.slug}"
                  class="block px-4 py-1.5 text-sm transition-colors {isActive(doc.slug)
                    ? 'bg-[#161b22] text-[#58a6ff] border-l-2 border-[#58a6ff]'
                    : 'text-[#9198a1] hover:bg-[#161b22] hover:text-[#e6edf3] border-l-2 border-transparent'}"
                  onclick={() => onClose?.()}
                >
                  {doc.title}
                </a>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
    {/each}
  </div>
</aside>
