<script lang="ts">
	import { RefreshCw } from 'lucide-svelte';
	import { syncStatus, doTriggerSync } from '$lib/stores';
	import { relativeTime } from '$lib/utils';

	let status = $derived($syncStatus);
</script>

<div class="flex items-center gap-2">
	{#if status.last_run}
		<span class="text-xs opacity-60">synced {relativeTime(status.last_run)}</span>
	{/if}
	<button
		class="btn btn-ghost btn-sm"
		onclick={() => doTriggerSync()}
		disabled={status.running}
		aria-label="Trigger sync"
	>
		<RefreshCw class="w-4 h-4 {status.running ? 'animate-spin' : ''}" />
		Sync
	</button>
</div>
