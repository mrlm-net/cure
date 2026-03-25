import { getAllDocSlugs } from '$lib/content/pipeline.server.js';
import { siteConfig } from '$lib/config/site.js';

export const prerender = true;

export function GET(): Response {
  const slugs = getAllDocSlugs();
  const base = siteConfig.url;

  const staticPages = ['', '/docs', '/changelog', '/getting-started'];

  const docPages = slugs.map((slug) => `/docs/${slug}`);

  const allPages = [...staticPages, ...docPages];

  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
${allPages
  .map(
    (path) => `  <url>
    <loc>${base}${path}</loc>
  </url>`
  )
  .join('\n')}
</urlset>`;

  return new Response(xml, {
    headers: {
      'Content-Type': 'application/xml',
      'Cache-Control': 'max-age=3600'
    }
  });
}
