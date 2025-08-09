import type {AppItem} from '../types/domain';

export function filterApps(apps: AppItem[], opts: {
  clusters?: Set<string>, namespaces?: Set<string>, projects?: Set<string>, q?: string
}) {
  let out = apps;
  if (opts.clusters?.size) out = out.filter(a => opts.clusters!.has(a.clusterLabel || ''));
  if (opts.namespaces?.size) out = out.filter(a => opts.namespaces!.has(a.namespace || ''));
  if (opts.projects?.size) out = out.filter(a => opts.projects!.has(a.project || ''));
  const q = (opts.q || '').toLowerCase();
  if (q) out = out.filter(a => [a.name, a.sync, a.health, a.namespace, a.project].some(v => (v || '').toLowerCase().includes(q)));
  return out;
}
