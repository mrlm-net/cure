import { redirect } from '@sveltejs/kit';
import { base } from '$app/paths';

export const prerender = true;

export function load() {
  throw redirect(301, `${base}/docs/installation`);
}
