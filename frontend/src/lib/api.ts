import type { ItemsResponse, ItemsQuery, RawItem, Item, Stats, SyncStatus, OrgConfig, RepoInfo, Assignee } from './types';

async function request<T>(path: string, options?: RequestInit): Promise<T> {
	const res = await fetch(path, options);
	if (!res.ok) {
		const body = await res.json().catch(() => ({ error: res.statusText }));
		throw new Error(body.error || `Request failed: ${res.status}`);
	}
	return res.json();
}

export function parseItem(raw: RawItem): Item {
	let labels: string[] = [];
	if (raw.labels) {
		labels = raw.labels.split(',').filter((l) => l.length > 0);
	}

	let assignees: Assignee[] = [];
	if (raw.assignees) {
		try {
			assignees = JSON.parse(raw.assignees);
		} catch {
			assignees = [];
		}
	}

	return {
		...raw,
		labels,
		assignees
	};
}

export async function fetchItems(query: ItemsQuery): Promise<{ items: Item[]; total: number; page: number; per_page: number; total_pages: number }> {
	const params = new URLSearchParams();
	if (query.q) params.set('q', query.q);
	if (query.type) params.set('type', query.type);
	if (query.state) params.set('state', query.state);
	if (query.org) params.set('org', query.org);
	if (query.repo) params.set('repo', query.repo);
	if (query.author) params.set('author', query.author);
	if (query.label) params.set('label', query.label);
	if (query.since) params.set('since', query.since);
	if (query.sort) params.set('sort', query.sort);
	if (query.order) params.set('order', query.order);
	if (query.page) params.set('page', String(query.page));
	if (query.per_page) params.set('per_page', String(query.per_page));

	const qs = params.toString();
	const data = await request<ItemsResponse>(`/api/items${qs ? '?' + qs : ''}`);
	return {
		...data,
		items: data.items.map(parseItem)
	};
}

export async function fetchItem(id: number): Promise<Item> {
	const raw = await request<RawItem>(`/api/items/${id}`);
	return parseItem(raw);
}

export async function triggerSync(): Promise<void> {
	await request('/api/sync', { method: 'POST' });
}

export async function fetchSyncStatus(): Promise<SyncStatus> {
	return request<SyncStatus>('/api/sync/status');
}

export async function fetchOrgs(): Promise<OrgConfig[]> {
	return request<OrgConfig[]>('/api/config/orgs');
}

export async function fetchRepos(): Promise<RepoInfo[]> {
	return request<RepoInfo[]>('/api/repos');
}

export async function fetchLabels(): Promise<string[]> {
	return request<string[]>('/api/labels');
}

export async function fetchAuthors(): Promise<string[]> {
	return request<string[]>('/api/authors');
}

export async function fetchStats(): Promise<Stats> {
	return request<Stats>('/api/stats');
}
