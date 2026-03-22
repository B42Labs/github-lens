<script lang="ts">
	import { Search, X, ChevronsUpDown } from 'lucide-svelte';
	import {
		searchQuery,
		selectedOrg,
		selectedRepo,
		selectedType,
		selectedState,
		selectedLabel,
		selectedAuthor,
		activeFilters,
		orgs,
		repos,
		labels,
		authors,
		removeFilter,
		clearAllFilters,
		currentPage
	} from '$lib/stores';
	import { debounce } from '$lib/utils';

	let searchInput = $state($searchQuery);
	let filters = $derived($activeFilters);
	let orgList = $derived($orgs);
	let repoList = $derived($repos);
	let labelList = $derived($labels);
	let authorList = $derived($authors);

	// Repo combobox state
	let repoSearch = $state('');
	let repoDropdownOpen = $state(false);
	let repoInputEl: HTMLInputElement | undefined = $state();

	let filteredRepos = $derived.by(() => {
		const term = repoSearch.toLowerCase();
		if (!term) return repoList;
		return repoList.filter(
			(r) => r.repo.toLowerCase().includes(term) || r.org.toLowerCase().includes(term)
		);
	});

	const debouncedSearch = debounce((value: string) => {
		searchQuery.set(value);
		currentPage.set(1);
	}, 300);

	function onSearchInput(e: Event) {
		const value = (e.target as HTMLInputElement).value;
		searchInput = value;
		debouncedSearch(value);
	}

	function clearSearch() {
		searchInput = '';
		searchQuery.set('');
		currentPage.set(1);
	}

	function onOrgChange(e: Event) {
		selectedOrg.set((e.target as HTMLSelectElement).value);
		currentPage.set(1);
	}

	function onTypeChange(e: Event) {
		selectedType.set((e.target as HTMLSelectElement).value);
		currentPage.set(1);
	}

	function onStateChange(e: Event) {
		selectedState.set((e.target as HTMLSelectElement).value);
		currentPage.set(1);
	}

	function selectRepo(repo: string) {
		selectedRepo.set(repo);
		repoSearch = '';
		repoDropdownOpen = false;
		currentPage.set(1);
	}

	function clearRepo() {
		selectedRepo.set('');
		repoSearch = '';
		currentPage.set(1);
	}

	function onRepoInputFocus() {
		repoDropdownOpen = true;
	}

	function onRepoInputKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			repoDropdownOpen = false;
			repoInputEl?.blur();
		}
	}

	function handleRepoBlur() {
		// Delay closing so that onmousedown on a dropdown item can fire first
		setTimeout(() => {
			repoDropdownOpen = false;
		}, 150);
	}

	// Label combobox state
	let labelSearch = $state('');
	let labelDropdownOpen = $state(false);
	let labelInputEl: HTMLInputElement | undefined = $state();

	let filteredLabels = $derived.by(() => {
		const term = labelSearch.toLowerCase();
		if (!term) return labelList;
		return labelList.filter((l) => l.toLowerCase().includes(term));
	});

	function selectLabel(label: string) {
		selectedLabel.set(label);
		labelSearch = '';
		labelDropdownOpen = false;
		currentPage.set(1);
	}

	function clearLabel() {
		selectedLabel.set('');
		labelSearch = '';
		currentPage.set(1);
	}

	function onLabelInputFocus() {
		labelDropdownOpen = true;
	}

	function onLabelInputKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			labelDropdownOpen = false;
			labelInputEl?.blur();
		}
	}

	function handleLabelBlur() {
		// Delay closing so that onmousedown on a dropdown item can fire first
		setTimeout(() => {
			labelDropdownOpen = false;
		}, 150);
	}

	// Author combobox state
	let authorSearch = $state('');
	let authorDropdownOpen = $state(false);
	let authorInputEl: HTMLInputElement | undefined = $state();

	let filteredAuthors = $derived.by(() => {
		const term = authorSearch.toLowerCase();
		if (!term) return authorList;
		return authorList.filter((a) => a.toLowerCase().includes(term));
	});

	function selectAuthor(author: string) {
		selectedAuthor.set(author);
		authorSearch = '';
		authorDropdownOpen = false;
		currentPage.set(1);
	}

	function clearAuthor() {
		selectedAuthor.set('');
		authorSearch = '';
		currentPage.set(1);
	}

	function onAuthorInputFocus() {
		authorDropdownOpen = true;
	}

	function onAuthorInputKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			authorDropdownOpen = false;
			authorInputEl?.blur();
		}
	}

	function handleAuthorBlur() {
		// Delay closing so that onmousedown on a dropdown item can fire first
		setTimeout(() => {
			authorDropdownOpen = false;
		}, 150);
	}
</script>

<div class="card bg-base-100 shadow-sm">
	<div class="card-body p-4 flex flex-col gap-3">
		<!-- Row 1: Search -->
		<div class="form-control">
			<label class="input input-bordered input-md flex items-center gap-2">
				<Search class="w-4 h-4 opacity-50" />
				<input
					type="text"
					class="grow"
					placeholder="Search issues and PRs..."
					value={searchInput}
					oninput={onSearchInput}
				/>
				{#if searchInput}
					<button
						class="btn btn-ghost btn-xs btn-circle"
						onclick={clearSearch}
						aria-label="Clear search"
					>
						<X class="w-3.5 h-3.5" />
					</button>
				{/if}
			</label>
		</div>

		<!-- Row 2: Filters -->
		<div class="flex items-center gap-2 flex-wrap">
			<select class="select select-bordered select-sm" onchange={onOrgChange} value={$selectedOrg}>
				<option value="">All Organizations</option>
				{#each orgList as org}
					<option value={org.name}>{org.name}</option>
				{/each}
			</select>

			<!-- Searchable repo dropdown -->
			<div class="relative">
				{#if $selectedRepo}
					<div class="input input-bordered input-sm flex items-center gap-2 pr-1.5">
						<span class="text-sm">{$selectedRepo}</span>
						<button class="btn btn-ghost btn-xs p-0" onclick={clearRepo} aria-label="Clear repo filter">
							<X class="w-3.5 h-3.5" />
						</button>
					</div>
				{:else}
					<label class="input input-bordered input-sm flex items-center gap-2">
						<input
							bind:this={repoInputEl}
							type="text"
							class="grow w-28"
							placeholder="Repository..."
							bind:value={repoSearch}
							onfocus={onRepoInputFocus}
							onblur={handleRepoBlur}
							onkeydown={onRepoInputKeydown}
						/>
						<ChevronsUpDown class="w-3.5 h-3.5 opacity-40" />
					</label>
				{/if}

				{#if repoDropdownOpen && !$selectedRepo}
					<ul class="absolute z-50 mt-1 w-64 max-h-60 overflow-y-auto bg-base-100 border border-base-300 rounded-lg shadow-lg">
						{#if filteredRepos.length === 0}
							<li class="px-3 py-2 text-sm opacity-50">No repos found</li>
						{:else}
							{#each filteredRepos as r}
								<li>
									<button
										class="w-full text-left px-3 py-2 text-sm hover:bg-base-200 transition-colors"
										onmousedown={() => selectRepo(r.repo)}
									>
										<span class="opacity-50">{r.org}/</span>{r.repo}
									</button>
								</li>
							{/each}
						{/if}
					</ul>
				{/if}
			</div>

			<div class="divider divider-horizontal mx-0 h-6 self-center"></div>

			<select class="select select-bordered select-sm" onchange={onTypeChange} value={$selectedType}>
				<option value="">All Types</option>
				<option value="issue">Issues</option>
				<option value="pr">Pull Requests</option>
			</select>

			<select class="select select-bordered select-sm" onchange={onStateChange} value={$selectedState}>
				<option value="">All States</option>
				<option value="open">Open</option>
				<option value="closed">Closed</option>
				<option value="merged">Merged</option>
			</select>

			<div class="divider divider-horizontal mx-0 h-6 self-center"></div>

			<!-- Searchable label dropdown -->
			<div class="relative">
				{#if $selectedLabel}
					<div class="input input-bordered input-sm flex items-center gap-2 pr-1.5">
						<span class="text-sm">{$selectedLabel}</span>
						<button class="btn btn-ghost btn-xs p-0" onclick={clearLabel} aria-label="Clear label filter">
							<X class="w-3.5 h-3.5" />
						</button>
					</div>
				{:else}
					<label class="input input-bordered input-sm flex items-center gap-2">
						<input
							bind:this={labelInputEl}
							type="text"
							class="grow w-24"
							placeholder="Label..."
							bind:value={labelSearch}
							onfocus={onLabelInputFocus}
							onblur={handleLabelBlur}
							onkeydown={onLabelInputKeydown}
						/>
						<ChevronsUpDown class="w-3.5 h-3.5 opacity-40" />
					</label>
				{/if}

				{#if labelDropdownOpen && !$selectedLabel}
					<ul class="absolute z-50 mt-1 w-64 max-h-60 overflow-y-auto bg-base-100 border border-base-300 rounded-lg shadow-lg">
						{#if filteredLabels.length === 0}
							<li class="px-3 py-2 text-sm opacity-50">No labels found</li>
						{:else}
							{#each filteredLabels as l}
								<li>
									<button
										class="w-full text-left px-3 py-2 text-sm hover:bg-base-200 transition-colors"
										onmousedown={() => selectLabel(l)}
									>
										{l}
									</button>
								</li>
							{/each}
						{/if}
					</ul>
				{/if}
			</div>

			<!-- Searchable author dropdown -->
			<div class="relative">
				{#if $selectedAuthor}
					<div class="input input-bordered input-sm flex items-center gap-2 pr-1.5">
						<span class="text-sm">{$selectedAuthor}</span>
						<button class="btn btn-ghost btn-xs p-0" onclick={clearAuthor} aria-label="Clear author filter">
							<X class="w-3.5 h-3.5" />
						</button>
					</div>
				{:else}
					<label class="input input-bordered input-sm flex items-center gap-2">
						<input
							bind:this={authorInputEl}
							type="text"
							class="grow w-24"
							placeholder="Author..."
							bind:value={authorSearch}
							onfocus={onAuthorInputFocus}
							onblur={handleAuthorBlur}
							onkeydown={onAuthorInputKeydown}
						/>
						<ChevronsUpDown class="w-3.5 h-3.5 opacity-40" />
					</label>
				{/if}

				{#if authorDropdownOpen && !$selectedAuthor}
					<ul class="absolute z-50 mt-1 w-64 max-h-60 overflow-y-auto bg-base-100 border border-base-300 rounded-lg shadow-lg">
						{#if filteredAuthors.length === 0}
							<li class="px-3 py-2 text-sm opacity-50">No authors found</li>
						{:else}
							{#each filteredAuthors as a}
								<li>
									<button
										class="w-full text-left px-3 py-2 text-sm hover:bg-base-200 transition-colors"
										onmousedown={() => selectAuthor(a)}
									>
										{a}
									</button>
								</li>
							{/each}
						{/if}
					</ul>
				{/if}
			</div>

		</div>

		<!-- Row 3: Active filter badges + Clear all -->
		{#if filters.length > 0}
			<div class="flex flex-wrap items-center gap-1.5">
				{#each filters as filter}
					<span class="badge badge-sm gap-1">
						{filter.label}: {filter.value}
						<button onclick={() => removeFilter(filter.key)} aria-label="Remove filter">
							<X class="w-3 h-3" />
						</button>
					</span>
				{/each}
				<button class="btn btn-ghost btn-xs opacity-60" onclick={clearAllFilters}>Clear all</button>
			</div>
		{/if}
	</div>
</div>
