<script lang="ts">
	import { stats, newItemsCount, lastVisitTimestamp } from '$lib/stores';
	import { CircleDot, GitPullRequest, FolderGit2, Sparkles } from 'lucide-svelte';

	let s = $derived($stats);
	let newCount = $derived($newItemsCount);
	let hasVisitHistory = $derived($lastVisitTimestamp !== null);
</script>

<div class="flex items-center gap-4 text-sm">
	<div class="flex items-center gap-1.5 text-warning">
		<CircleDot class="w-4 h-4" />
		<span class="font-medium">{s.open_issues}</span>
		<span class="opacity-60">Issues</span>
	</div>
	<div class="flex items-center gap-1.5 text-info">
		<GitPullRequest class="w-4 h-4" />
		<span class="font-medium">{s.open_prs}</span>
		<span class="opacity-60">PRs</span>
	</div>
	<div class="flex items-center gap-1.5 text-accent">
		<FolderGit2 class="w-4 h-4" />
		<span class="font-medium">{s.repo_count}</span>
		<span class="opacity-60">Repos</span>
	</div>
	{#if hasVisitHistory && newCount > 0}
		<div class="flex items-center gap-1.5 text-success">
			<Sparkles class="w-4 h-4" />
			<span class="font-medium">{newCount}</span>
			<span class="opacity-60">New</span>
		</div>
	{/if}
</div>
