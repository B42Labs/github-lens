<script lang="ts">
	import { CircleDot, GitPullRequest, GitMerge, ChevronRight } from 'lucide-svelte';
	import type { Item } from '$lib/types';
	import { openDrawer } from '$lib/stores';
	import { relativeTime, labelBadgeColor } from '$lib/utils';

	let { item }: { item: Item } = $props();

	function typeIcon(type: string, state: string) {
		if (type === 'pr' && state === 'merged') return { icon: GitMerge, color: 'text-purple-500' };
		if (type === 'pr') return { icon: GitPullRequest, color: state === 'open' ? 'text-green-500' : 'text-red-500' };
		return { icon: CircleDot, color: state === 'open' ? 'text-green-500' : 'text-red-500' };
	}

	let iconInfo = $derived(typeIcon(item.type, item.state));
	let IconComponent = $derived(iconInfo.icon);
</script>

<div class="flex items-center gap-3 px-4 py-3 border-b border-base-300 last:border-b-0 hover:bg-base-200/50 transition-colors">
	<!-- Type icon -->
	<div class="shrink-0">
		<IconComponent class="w-5 h-5 {iconInfo.color}" />
	</div>

	<!-- Main content: two lines -->
	<div class="flex-1 min-w-0">
		<div class="text-sm opacity-60">
			{item.org}/{item.repo}
			<span class="ml-1">#{item.number}</span>
		</div>
		<div class="font-medium truncate">
			<a href={item.url} target="_blank" rel="noopener noreferrer" class="hover:underline">
				{item.title}
			</a>
		</div>
		{#if item.labels.length > 0}
			<div class="flex flex-wrap gap-1 mt-1">
				{#each item.labels as label}
					<span class="badge badge-sm {labelBadgeColor(label)}">{label}</span>
				{/each}
			</div>
		{/if}
	</div>

	<!-- Right-aligned meta -->
	<div class="flex items-center gap-4 shrink-0">
		{#if item.author_avatar}
			<div class="avatar" title={item.author}>
				<div class="w-7 rounded-full">
					<img src={item.author_avatar} alt={item.author} />
				</div>
			</div>
		{/if}
		<span class="text-sm opacity-50 w-24 text-right">{relativeTime(item.updated_at)}</span>
		<button class="btn btn-ghost btn-xs" onclick={() => openDrawer(item)} aria-label="View details">
			<ChevronRight class="w-4 h-4" />
		</button>
	</div>
</div>
