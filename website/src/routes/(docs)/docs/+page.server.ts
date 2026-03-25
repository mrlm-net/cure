import { getAllDocMeta } from '$lib/content/pipeline.server.js';
import { buildNavSections } from '$lib/config/navigation.js';
import type { NavSection, DocMeta } from '$lib/types/index.js';

export const prerender = true;

export function load(): { sections: NavSection[]; firstDoc: DocMeta | null } {
  const docs = getAllDocMeta();
  const sections = buildNavSections(docs);

  // Find the first doc (installation) to link to
  const firstDoc = sections.length > 0 && sections[0].docs.length > 0 ? sections[0].docs[0] : null;

  return { sections, firstDoc };
}
