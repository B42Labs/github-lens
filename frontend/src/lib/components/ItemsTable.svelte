<script lang="ts">
	import { ArrowUpDown, ArrowUp, ArrowDown } from 'lucide-svelte';
	import {
		items,
		loading,
		totalItems,
		totalPages,
		currentPage,
		sortField,
		sortOrder,
		toggleSort
	} from '$lib/stores';
	import TableRow from './TableRow.svelte';

	let itemList = $derived($items);
	let isLoading = $derived($loading);
	let total = $derived($totalItems);
	let pages = $derived($totalPages);
	let page = $derived($currentPage);
	let sort = $derived($sortField);
	let order = $derived($sortOrder);

	function sortIcon(field: string) {
		if (sort !== field) return ArrowUpDown;
		return order === 'asc' ? ArrowUp : ArrowDown;
	}

	let TitleSortIcon = $derived(sortIcon('title'));
	let UpdatedSortIcon = $derived(sortIcon('updated_at'));

	function goToPage(p: number) {
		if (p >= 1 && p <= pages) currentPage.set(p);
	}

	function pageNumbers(current: number, total: number): (number | '...')[] {
		if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
		const pages: (number | '...')[] = [1];
		if (current > 3) pages.push('...');
		for (let i = Math.max(2, current - 1); i <= Math.min(total - 1, current + 1); i++) {
			pages.push(i);
		}
		if (current < total - 2) pages.push('...');
		pages.push(total);
		return pages;
	}
</script>

<div class="rounded-xl border border-base-300 bg-base-100 overflow-hidden">
	<!-- Sort controls header with pagination -->
	<div class="flex items-center gap-4 px-4 py-2.5 border-b border-base-300 bg-base-200/50 text-sm">
		<button class="flex items-center gap-1 opacity-70 hover:opacity-100 transition-opacity" onclick={() => toggleSort('title')}>
			Title
			<TitleSortIcon class="w-3 h-3" />
		</button>
		<div class="flex items-center gap-4 ml-auto">
			{#if pages > 1}
				<span class="hidden sm:inline opacity-60">{total} items total</span>
				<div class="join">
					<button
						class="join-item btn btn-xs"
						aria-label="Previous page"
						disabled={page <= 1}
						onclick={() => goToPage(page - 1)}
					>
						Prev
					</button>
					{#each pageNumbers(page, pages) as p}
						{#if p === '...'}
							<span class="join-item btn btn-xs btn-disabled" aria-hidden="true">...</span>
						{:else}
							<button
								class="join-item btn btn-xs"
								class:btn-active={p === page}
								aria-label="Go to page {p}"
								aria-current={p === page ? 'page' : undefined}
								onclick={() => goToPage(p as number)}
							>
								{p}
							</button>
						{/if}
					{/each}
					<button
						class="join-item btn btn-xs"
						aria-label="Next page"
						disabled={page >= pages}
						onclick={() => goToPage(page + 1)}
					>
						Next
					</button>
				</div>
			{/if}
			<button class="flex items-center gap-1 opacity-70 hover:opacity-100 transition-opacity" onclick={() => toggleSort('updated_at')}>
				Updated
				<UpdatedSortIcon class="w-3 h-3" />
			</button>
		</div>
	</div>

	<!-- Rows -->
	{#if isLoading}
		<div class="flex justify-center py-16">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if itemList.length === 0}
		<div class="text-center py-16 opacity-50">
			No items found. Try adjusting your filters or trigger a sync.
		</div>
	{:else}
		{#each itemList as item (item.id)}
			<TableRow {item} />
		{/each}
	{/if}
</div>
