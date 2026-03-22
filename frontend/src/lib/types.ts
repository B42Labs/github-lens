export interface RawItem {
	id: number;
	github_id: number;
	type: 'issue' | 'pr';
	state: 'open' | 'closed' | 'merged';
	title: string;
	body: string;
	url: string;
	number: number;
	org: string;
	repo: string;
	author: string;
	author_avatar: string;
	labels: string;
	assignees: string;
	created_at: string;
	updated_at: string;
	synced_at: string;
}

export interface Item {
	id: number;
	github_id: number;
	type: 'issue' | 'pr';
	state: 'open' | 'closed' | 'merged';
	title: string;
	body: string;
	url: string;
	number: number;
	org: string;
	repo: string;
	author: string;
	author_avatar: string;
	labels: string[];
	assignees: Assignee[];
	created_at: string;
	updated_at: string;
	synced_at: string;
}

export interface Assignee {
	login: string;
	avatar_url: string;
}

export interface ItemsResponse {
	items: RawItem[];
	total: number;
	page: number;
	per_page: number;
	total_pages: number;
}

export interface ItemsQuery {
	q?: string;
	type?: string;
	state?: string;
	org?: string;
	repo?: string;
	author?: string;
	label?: string;
	since?: string;
	sort?: string;
	order?: string;
	page?: number;
	per_page?: number;
}

export interface Stats {
	open_issues: number;
	open_prs: number;
	repo_count: number;
}

export interface SyncStatus {
	running: boolean;
	last_run: string | null;
	progress: string;
}

export interface ApiError {
	error: string;
	code: string;
}

export interface Toast {
	id: number;
	message: string;
	type: 'success' | 'error' | 'info';
}

export interface ActiveFilter {
	key: string;
	label: string;
	value: string;
}

export interface OrgConfig {
	name: string;
	include_repos: string[] | null;
	exclude_repos: string[] | null;
}

export interface RepoInfo {
	org: string;
	repo: string;
}
