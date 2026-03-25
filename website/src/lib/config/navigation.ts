import type { NavSection, DocMeta } from '$lib/types/index.js';

export interface SectionMeta {
  title: string;
  defaultOpen: boolean;
}

export const sectionMeta: Record<string, SectionMeta> = {
  'getting-started': { title: 'Getting Started', defaultOpen: true },
  commands: { title: 'Commands', defaultOpen: true },
  libraries: { title: 'Libraries', defaultOpen: true },
  development: { title: 'Development', defaultOpen: false }
};

export const sectionOrder = ['getting-started', 'commands', 'libraries', 'development'];

export interface TopNavLink {
  label: string;
  href: string;
}

export const topNavLinks: TopNavLink[] = [
  { label: 'Getting Started', href: '/getting-started' },
  { label: 'Docs', href: '/docs' },
  { label: 'Changelog', href: '/changelog' }
];

/**
 * Build the full navigation structure from a flat list of doc metadata entries.
 */
export function buildNavSections(docs: DocMeta[]): NavSection[] {
  const grouped: Record<string, DocMeta[]> = {};

  for (const doc of docs) {
    const section = doc.section || 'other';
    if (!grouped[section]) {
      grouped[section] = [];
    }
    grouped[section].push(doc);
  }

  // Sort docs within each section by their order field
  for (const section of Object.keys(grouped)) {
    grouped[section].sort((a, b) => a.order - b.order);
  }

  return sectionOrder
    .filter((id) => grouped[id])
    .map((id) => ({
      id,
      title: sectionMeta[id]?.title ?? id,
      defaultOpen: sectionMeta[id]?.defaultOpen ?? false,
      docs: grouped[id]
    }));
}
