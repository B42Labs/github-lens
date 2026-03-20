<script lang="ts">
	import { Eye } from 'lucide-svelte';
	import ThemeToggle from './ThemeToggle.svelte';
	import SyncButton from './SyncButton.svelte';
</script>

<div class="navbar bg-base-100 shadow-sm sticky top-0 z-50">
	<div class="flex-1">
		<a href="/" class="btn btn-ghost text-xl gap-2">
			<Eye class="w-6 h-6" />
			GitHub Lens
		</a>
	</div>
	<div class="flex-none gap-2">
		<SyncButton />
		<ThemeToggle />
	</div>
</div>
