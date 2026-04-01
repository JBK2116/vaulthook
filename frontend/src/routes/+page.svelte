<script lang="ts">
    import { MOCK_DATA } from '../utils/mock-data.ts';
    import * as functions from '../utils/functions.ts';
    import Navbar from '$lib/components/ui/Navbar.svelte';
    import StatCard from '$lib/components/ui/StatCard.svelte';
    import ConnIndicator from '$lib/components/ui/ConnIndicator.svelte';
    import { DeliveryStatusColors, SelectTypes } from '../utils/types.ts';
    import SelectFilter from '$lib/components/ui/SelectFilter.svelte';

    // Data Manipulation
    let data = $state(MOCK_DATA);
    let totalEvents = $derived(functions.getTotalEvents(data));
    let totalDeliveredEvents = $derived(functions.getTotalDeliveredEvents(data));
    let totalRetryingEvents = $derived(functions.getTotalRetryingEvents(data));
    let totalQueuedEvents = $derived(functions.getTotalQueuedEvents(data));
    let totalFailedEvents = $derived(functions.getTotalFailedEvents(data));

    // Connection State
    let isConnectedToBackend: boolean = $state(true);

    // Select Handling
    let currentSelectedOption: SelectTypes = $state(SelectTypes.All);
    $effect(() => {
        // TODO: Update this effect to change the webhook events displayed in the table
        console.log('Effect is running for currentSelectedOption: ' + currentSelectedOption);
    });
</script>

<Navbar></Navbar>
<div>
    <div class="flex h-[calc(100vh-4rem)] flex-col">
        <div class="border-border flex flex-row items-center justify-between border-b">
            <div class="border-border flex shrink-0">
                <StatCard label="Total (7days)" valueNumber={totalEvents} valueNumberColor={''} />
                <StatCard
                    label="Delivered"
                    valueNumber={totalDeliveredEvents}
                    valueNumberColor={DeliveryStatusColors.delivered}
                />
                <StatCard
                    label="Failed"
                    valueNumber={totalFailedEvents}
                    valueNumberColor={DeliveryStatusColors.failed}
                />
                <StatCard
                    label="Retrying"
                    valueNumber={totalRetryingEvents}
                    valueNumberColor={DeliveryStatusColors.retrying}
                />
                <StatCard
                    label="Queued"
                    valueNumber={totalQueuedEvents}
                    valueNumberColor={DeliveryStatusColors.queued}
                />
            </div>
            <ConnIndicator {isConnectedToBackend}></ConnIndicator>
        </div>
        <div
            class="border-border flex shrink-0 flex-row items-center justify-between gap-2 border-b px-4 py-2.5"
        >
            <SelectFilter bind:currentSelectedOption></SelectFilter>
        </div>
    </div>
</div>
