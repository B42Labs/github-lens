import { writable, derived, get } from 'svelte/store';
import type { Item, Stats, SyncStatus, Toast, ActiveFilter, OrgConfig, RepoInfo } from './types';
import * as api from './api';

// Filter stores
export const searchQuery = writable('');
export const selectedOrg = writable('');
export const selectedRepo = writable('');
export const selectedType = writable('');
export const selectedState = writable('open');
export const selectedLabel = writable('');
export const selectedAuthor = writable('');
export const currentPage = writable(1);
export const sortField = writable('updated_at');
export const sortOrder = writable<'asc' | 'desc'>('desc');
export const perPage = writable(25);

// Data stores
export const items = writable<Item[]>([]);
export const totalItems = writable(0);
export const totalPages = writable(0);
export const stats = writable<Stats>({ open_issues: 0, open_prs: 0, repo_count: 0 });
export const orgs = writable<OrgConfig[]>([]);
export const repos = writable<RepoInfo[]>([]);
export const labels = writable<string[]>([]);
export const authors = writable<string[]>([]);

// "New since last visit" stores
export const lastVisitTimestamp = writable<string | null>(null);
export const showNewOnly = writable(false);
export const newItemsCount = writable(0);

// UI stores
export const syncStatus = writable<SyncStatus>({ running: false, last_run: null, progress: '' });
export const selectedItem = writable<Item | null>(null);
export const drawerOpen = writable(false);
export const loading = writable(false);
export const toasts = writable<Toast[]>([]);

let toastId = 0;

// Derived stores
export const activeFilters = derived(
	[searchQuery, selectedOrg, selectedRepo, selectedType, selectedState, selectedLabel, selectedAuthor, showNewOnly, lastVisitTimestamp],
	([$q, $org, $repo, $type, $state, $label, $author, $showNew, $lastVisit]) => {
		const filters: ActiveFilter[] = [];
		if ($q) filters.push({ key: 'q', label: 'Search', value: $q });
		if ($org) filters.push({ key: 'org', label: 'Org', value: $org });
		if ($repo) filters.push({ key: 'repo', label: 'Repo', value: $repo });
		if ($type) filters.push({ key: 'type', label: 'Type', value: $type });
		if ($state) filters.push({ key: 'state', label: 'Status', value: $state });
		if ($label) filters.push({ key: 'label', label: 'Label', value: $label });
		if ($author) filters.push({ key: 'author', label: 'Author', value: $author });
		if ($showNew && $lastVisit) filters.push({ key: 'new', label: 'New', value: 'since last visit' });
		return filters;
	}
);

export const availableRepos = derived(repos, ($repos) => {
	return $repos.map((r) => r.repo).sort();
});

// Actions
let loadRequestId = 0;

export async function loadItems() {
	const thisRequest = ++loadRequestId;
	loading.set(true);
	try {
		const since = get(showNewOnly) && get(lastVisitTimestamp) ? get(lastVisitTimestamp)! : undefined;
		const result = await api.fetchItems({
			q: get(searchQuery) || undefined,
			type: get(selectedType) || undefined,
			state: get(selectedState) || undefined,
			org: get(selectedOrg) || undefined,
			repo: get(selectedRepo) || undefined,
			label: get(selectedLabel) || undefined,
			author: get(selectedAuthor) || undefined,
			since,
			sort: get(sortField),
			order: get(sortOrder),
			page: get(currentPage),
			per_page: get(perPage)
		});
		if (thisRequest !== loadRequestId) return;
		items.set(result.items);
		totalItems.set(result.total);
		totalPages.set(result.total_pages);
	} catch (e) {
		if (thisRequest !== loadRequestId) return;
		addToast((e as Error).message, 'error');
	} finally {
		if (thisRequest === loadRequestId) {
			loading.set(false);
		}
	}
}

export async function loadStats() {
	try {
		const s = await api.fetchStats();
		stats.set(s);
	} catch {
		// silently ignore stats errors
	}
}

export async function loadOrgs() {
	try {
		const o = await api.fetchOrgs();
		orgs.set(o);
	} catch {
		// silently ignore
	}
}

export async function loadRepos() {
	try {
		const r = await api.fetchRepos();
		repos.set(r);
	} catch {
		// silently ignore
	}
}

export async function loadAuthors() {
	try {
		const a = await api.fetchAuthors();
		authors.set(a);
	} catch {
		// silently ignore
	}
}

export async function loadLabels() {
	try {
		const l = await api.fetchLabels();
		labels.set(l);
	} catch {
		// silently ignore
	}
}

export async function loadSyncStatus() {
	try {
		const s = await api.fetchSyncStatus();
		syncStatus.set(s);
	} catch {
		// silently ignore
	}
}

export async function doTriggerSync() {
	try {
		await api.triggerSync();
		addToast('Sync started', 'info');
		// Poll for completion with timeout (max 5 minutes)
		let polls = 0;
		const maxPolls = 150;
		const interval = setInterval(async () => {
			polls++;
			if (polls >= maxPolls) {
				clearInterval(interval);
				addToast('Sync polling timed out — check status manually', 'error');
				return;
			}
			try {
				const s = await api.fetchSyncStatus();
				syncStatus.set(s);
				if (!s.running) {
					clearInterval(interval);
					addToast('Sync completed', 'success');
					loadItems();
					loadStats();
					loadNewItemsCount();
					loadRepos();
					loadLabels();
					loadAuthors();
				}
			} catch {
				clearInterval(interval);
				addToast('Lost connection while polling sync status', 'error');
			}
		}, 2000);
	} catch (e) {
		addToast((e as Error).message, 'error');
	}
}

export function toggleSort(field: string) {
	if (get(sortField) === field) {
		sortOrder.update((o) => (o === 'asc' ? 'desc' : 'asc'));
	} else {
		sortField.set(field);
		sortOrder.set('desc');
	}
	currentPage.set(1);
}

export function openDrawer(item: Item) {
	selectedItem.set(item);
	drawerOpen.set(true);
}

export function closeDrawer() {
	drawerOpen.set(false);
	selectedItem.set(null);
}

export function removeFilter(key: string) {
	if (key === 'q') searchQuery.set('');
	if (key === 'org') selectedOrg.set('');
	if (key === 'repo') selectedRepo.set('');
	if (key === 'type') selectedType.set('');
	if (key === 'state') selectedState.set('');
	if (key === 'label') selectedLabel.set('');
	if (key === 'author') selectedAuthor.set('');
	if (key === 'new') showNewOnly.set(false);
	currentPage.set(1);
}

export function clearAllFilters() {
	searchQuery.set('');
	selectedOrg.set('');
	selectedRepo.set('');
	selectedType.set('');
	selectedState.set('');
	selectedLabel.set('');
	selectedAuthor.set('');
	showNewOnly.set(false);
	currentPage.set(1);
}

export function initLastVisit() {
	if (typeof window === 'undefined') return;
	const saved = localStorage.getItem('github-lens-last-visit');
	lastVisitTimestamp.set(saved);
	// Defer updating the timestamp until the user leaves, so refreshing the page
	// doesn't immediately clear the "new" indicators before the user has seen them.
	window.addEventListener('beforeunload', () => {
		localStorage.setItem('github-lens-last-visit', new Date().toISOString());
	}, { once: true });
}

export async function loadNewItemsCount() {
	const since = get(lastVisitTimestamp);
	if (!since) {
		newItemsCount.set(0);
		return;
	}
	try {
		const result = await api.fetchItems({ since, per_page: 1, page: 1 });
		newItemsCount.set(result.total);
	} catch {
		// silently ignore
	}
}

export function addToast(message: string, type: Toast['type'] = 'info') {
	const id = ++toastId;
	toasts.update((t) => [...t, { id, message, type }]);
	setTimeout(() => {
		toasts.update((t) => t.filter((toast) => toast.id !== id));
	}, 4000);
}
