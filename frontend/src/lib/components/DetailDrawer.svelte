<script lang="ts">
	import { X, ExternalLink, CircleDot, GitPullRequest, GitMerge } from 'lucide-svelte';
	import { selectedItem, drawerOpen, closeDrawer } from '$lib/stores';
	import { relativeTime } from '$lib/utils';
	import { marked } from 'marked';
	import DOMPurify from 'dompurify';

	let item = $derived($selectedItem);
	let open = $derived($drawerOpen);

	function stateColor(state: string): string {
		if (state === 'open') return 'badge-success';
		if (state === 'merged') return 'badge-secondary';
		return 'badge-error';
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && open) {
			closeDrawer();
		}
	}

	function renderMarkdown(body: string): string {
		const html = marked.parse(body, { async: false }) as string;
		return DOMPurify.sanitize(html);
	}
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open && item}
	<!-- Backdrop -->
	<button
		class="fixed inset-0 bg-black/30 z-40"
		onclick={closeDrawer}
		aria-label="Close drawer"
	></button>

	<!-- Drawer panel -->
	<div class="fixed right-0 top-0 h-full w-full max-w-2xl bg-base-100 shadow-2xl z-50 flex flex-col overflow-hidden">
		<!-- Header -->
		<div class="flex items-start justify-between p-4 border-b border-base-300">
			<div class="flex-1 min-w-0">
				<div class="flex items-center gap-2 mb-1">
					{#if item.type === 'pr' && item.state === 'merged'}
						<GitMerge class="w-5 h-5 text-purple-500 shrink-0" />
					{:else if item.type === 'pr'}
						<GitPullRequest class="w-5 h-5 {item.state === 'open' ? 'text-green-500' : 'text-red-500'} shrink-0" />
					{:else}
						<CircleDot class="w-5 h-5 {item.state === 'open' ? 'text-green-500' : 'text-red-500'} shrink-0" />
					{/if}
					<span class="badge {stateColor(item.state)}">{item.state}</span>
					<span class="text-sm opacity-50">#{item.number}</span>
				</div>
				<h2 class="text-lg font-bold truncate">{item.title}</h2>
				<p class="text-sm opacity-60">{item.org}/{item.repo}</p>
			</div>
			<button class="btn btn-ghost btn-sm" onclick={closeDrawer} aria-label="Close">
				<X class="w-5 h-5" />
			</button>
		</div>

		<!-- Meta -->
		<div class="px-4 py-3 border-b border-base-300 flex flex-wrap gap-4 text-sm">
			<div class="flex items-center gap-2">
				{#if item.author_avatar}
					<div class="avatar">
						<div class="w-5 rounded-full">
							<img src={item.author_avatar} alt={item.author} />
						</div>
					</div>
				{/if}
				<span>{item.author}</span>
			</div>
			<span class="opacity-50">Created {relativeTime(item.created_at)}</span>
			<span class="opacity-50">Updated {relativeTime(item.updated_at)}</span>
			{#if item.assignees.length > 0}
				<div class="flex items-center gap-1">
					<span class="opacity-50">Assignees:</span>
					{#each item.assignees as assignee}
						<div class="avatar" title={assignee.login}>
							<div class="w-5 rounded-full">
								<img src={assignee.avatar_url} alt={assignee.login} />
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Body -->
		<div class="flex-1 overflow-y-auto p-4">
			{#if item.body}
				<div class="prose prose-sm max-w-none">
					<!-- eslint-disable-next-line svelte/no-at-html-tags -- sanitized via DOMPurify -->
					{@html renderMarkdown(item.body)}
				</div>
			{:else}
				<p class="text-center opacity-50 mt-8">No description provided.</p>
			{/if}
		</div>

		<!-- Footer -->
		<div class="p-4 border-t border-base-300">
			<a href={item.url} target="_blank" rel="noopener noreferrer" class="btn btn-primary w-full gap-2">
				<ExternalLink class="w-4 h-4" />
				Open on GitHub
			</a>
		</div>
	</div>
{/if}
