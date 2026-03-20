<script lang="ts">
	import { Sun, Moon } from 'lucide-svelte';
	import { browser } from '$app/environment';

	let dark = $state(false);

	$effect(() => {
		if (!browser) return;
		const saved = localStorage.getItem('theme');
		if (saved) {
			dark = saved === 'dark';
		} else {
			dark = window.matchMedia('(prefers-color-scheme: dark)').matches;
		}
		applyTheme();
	});

	function toggle() {
		dark = !dark;
		localStorage.setItem('theme', dark ? 'dark' : 'corporate');
		applyTheme();
	}

	function applyTheme() {
		document.documentElement.setAttribute('data-theme', dark ? 'dark' : 'corporate');
	}
</script>

<button class="btn btn-ghost btn-sm" onclick={toggle} aria-label="Toggle theme">
	{#if dark}
		<Sun class="w-5 h-5" />
	{:else}
		<Moon class="w-5 h-5" />
	{/if}
</button>
