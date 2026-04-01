<script lang="ts">
    import * as functions from '../utils/functions.ts';
    import Navbar from '$lib/components/ui/Navbar.svelte';
    import StatCard from '$lib/components/ui/StatCard.svelte';
    import { MOCK_DATA } from '../utils/mock-data.ts';
    import { DeliveryStatusColors } from '../utils/types.ts';
    import ConnIndicator from '$lib/components/ui/ConnIndicator.svelte';

    // TODO: Replace this with code from the backend later
    let data = $state(MOCK_DATA);
    let totalEvents = $state(functions.getTotalEvents(data));
    let totalDeliveredEvents = $derived(functions.getTotalDeliveredEvents(data));
    let totalRetryingEvents = $derived(functions.getTotalRetryingEvents(data));
    let totalQueuedEvents = $derived(functions.getTotalQueuedEvents(data));
    let totalFailedEvents = $derived(functions.getTotalFailedEvents(data));

    let isConnectedToBackend: boolean = $state(true);
</script>

<Navbar></Navbar>
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
</div>
