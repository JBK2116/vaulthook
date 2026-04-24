<script lang="ts">
    import Navbar from '$lib/components/ui/Navbar.svelte';
    import ProviderCard from '$lib/components/ui/ProviderCard.svelte';
    import type { Provider } from '$lib/utils/types';

    let { data } = $props(); // stores the initial provider data received from the backend
    let providers: Provider[] = $derived(data.providers);
</script>

<Navbar></Navbar>
<div class="m-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
    {#each providers as provider (provider)}
        <ProviderCard data={provider}></ProviderCard>
    {/each}
</div>
