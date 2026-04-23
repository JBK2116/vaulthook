<script lang="ts">
    import Navbar from '$lib/components/ui/Navbar.svelte';
    import ProviderCard from '$lib/components/ui/ProviderCard.svelte';
    import { MOCK_PROVIDERS } from '$lib/utils/mock-data';
    import type { Provider } from '$lib/utils/types';

    // Data manipulation
    // TODO: Replace this with data from the backend
    let providers: Provider[] = $state(MOCK_PROVIDERS);
</script>

<Navbar></Navbar>
<div class="m-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
    {#each providers as provider (provider)}
        <ProviderCard {provider}></ProviderCard>
    {/each}
</div>
