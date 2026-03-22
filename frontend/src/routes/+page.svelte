<script lang="ts">
	import { onMount } from 'svelte';
	import Navbar from '$lib/components/Navbar.svelte';
	import FilterBar from '$lib/components/FilterBar.svelte';
	import ItemsTable from '$lib/components/ItemsTable.svelte';
	import DetailDrawer from '$lib/components/DetailDrawer.svelte';
	import {
		loadItems,
		loadStats,
		loadOrgs,
		loadRepos,
		loadLabels,
		loadAuthors,
		loadSyncStatus,
		searchQuery,
		selectedOrg,
		selectedRepo,
		selectedType,
		selectedState,
		selectedLabel,
		selectedAuthor,
		currentPage,
		sortField,
		sortOrder,
		toasts
	} from '$lib/stores';

	let toastList = $derived($toasts);

	// Reload items when filters change
	$effect(() => {
		// Track all filter dependencies
		void $searchQuery;
		void $selectedOrg;
		void $selectedRepo;
		void $selectedType;
		void $selectedState;
		void $selectedLabel;
		void $selectedAuthor;
		void $currentPage;
		void $sortField;
		void $sortOrder;
		loadItems();
	});

	// Initial data load (run once on mount)
	onMount(() => {
		loadStats();
		loadOrgs();
		loadRepos();
		loadLabels();
		loadAuthors();
		loadSyncStatus();

		// Poll sync status every 30s
		const interval = setInterval(loadSyncStatus, 30000);
		return () => clearInterval(interval);
	});
</script>

<div class="min-h-screen bg-base-200">
	<Navbar />

	<main class="container mx-auto px-4 py-6 max-w-7xl flex flex-col gap-6">
		<FilterBar />
		<ItemsTable />
	</main>

	<DetailDrawer />

	<!-- Toast container -->
	<div class="toast toast-end toast-bottom z-[60]">
		{#each toastList as toast (toast.id)}
			<div
				class="alert"
				class:alert-success={toast.type === 'success'}
				class:alert-error={toast.type === 'error'}
				class:alert-info={toast.type === 'info'}
			>
				<span>{toast.message}</span>
			</div>
		{/each}
	</div>
</div>
