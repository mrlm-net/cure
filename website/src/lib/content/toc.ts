export interface TocEntry {
  id: string;
  text: string;
  level: number;
}

/**
 * Extract table of contents entries from raw markdown content.
 * Returns headings h2-h4 with their slugified IDs.
 */
export function extractToc(content: string): TocEntry[] {
  const entries: TocEntry[] = [];
  const lines = content.split('\n');
  const counts: Record<string, number> = {};

  for (const line of lines) {
    const match = line.match(/^(#{2,4})\s+(.+)$/);
    if (!match) continue;

    const level = match[1].length;
    const text = match[2].trim();
    const slug = slugify(text);

    counts[slug] = (counts[slug] || 0) + 1;
    const id = counts[slug] === 1 ? slug : `${slug}-${counts[slug]}`;

    entries.push({ id, text, level });
  }

  return entries;
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, '')
    .replace(/[\s_]+/g, '-')
    .replace(/^-+|-+$/g, '');
}
