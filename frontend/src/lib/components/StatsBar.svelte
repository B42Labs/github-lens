<script lang="ts">
	import { stats } from '$lib/stores';
	import { CircleDot, GitPullRequest, FolderGit2 } from 'lucide-svelte';

	let s = $derived($stats);
</script>

<div class="stats shadow w-full">
	<div class="stat">
		<div class="stat-figure text-warning">
			<CircleDot class="w-8 h-8" />
		</div>
		<div class="stat-title">Open Issues</div>
		<div class="stat-value text-warning">{s.open_issues}</div>
	</div>

	<div class="stat">
		<div class="stat-figure text-info">
			<GitPullRequest class="w-8 h-8" />
		</div>
		<div class="stat-title">Open PRs</div>
		<div class="stat-value text-info">{s.open_prs}</div>
	</div>

	<div class="stat">
		<div class="stat-figure text-accent">
			<FolderGit2 class="w-8 h-8" />
		</div>
		<div class="stat-title">Repos Tracked</div>
		<div class="stat-value text-accent">{s.repo_count}</div>
	</div>
</div>
