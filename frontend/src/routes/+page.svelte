<script lang="ts">
    import ConnIndicator from '$lib/components/ui/ConnIndicator.svelte';
    import EventSheet from '$lib/components/ui/EventSheet.svelte';
    import EventTable from '$lib/components/ui/EventTable.svelte';
    import Navbar from '$lib/components/ui/Navbar.svelte';
    import SearchFilter from '$lib/components/ui/SearchFilter.svelte';
    import SelectFilter from '$lib/components/ui/SelectFilter.svelte';
    import Sidebar from '$lib/components/ui/Sidebar.svelte';
    import StatCard from '$lib/components/ui/StatCard.svelte';

    import * as functions from '../utils/functions.ts';
    import { MOCK_DATA } from '../utils/mock-data.ts';
    import { DeliveryStatusColors, SelectTypes, type WebHookEvent } from '../utils/types.ts';

    // Data Manipulation
    let events: WebHookEvent[] = $state(MOCK_DATA);
    let totalEvents = $derived(functions.getTotalEvents(events));
    let totalDeliveredEvents = $derived(functions.getTotalDeliveredEvents(events));
    let totalRetryingEvents = $derived(functions.getTotalRetryingEvents(events));
    let totalQueuedEvents = $derived(functions.getTotalQueuedEvents(events));
    let totalFailedEvents = $derived(functions.getTotalFailedEvents(events));

    // Connection State
    let isConnectedToBackend: boolean = $state(true);

    // Select & Search Handling
    let currentSelectedOption: SelectTypes = $state(SelectTypes.All);
    let currentSearchString: string = $state('');

    // Table State
    let currentSelectedEvent: WebHookEvent | null = $state(null);
    let displayedEvents: WebHookEvent[] = $derived(
        functions.getDisplayedEvents(currentSelectedOption, currentSearchString, events),
    );
    $effect(() => {
        // TODO: Update this effect to change the sidebar display beside the table
    });

    // Sheet State
    let isSheetOpen: boolean = $state(false);
    $effect(() => {
        if (currentSelectedEvent && window.innerWidth < 768) {
            isSheetOpen = true;
        }
    });

    // TODO: Delete these inspect runes once each associated variable has had it's effect rune applied
    $inspect(currentSelectedEvent);
    $inspect(currentSelectedOption);
    $inspect(currentSearchString);
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
            class="border-border flex shrink-0 flex-col sm:flex-row items-start sm:items-center justify-between gap-2 border-b px-4 py-2.5"
        >
            <SelectFilter bind:currentSelectedOption></SelectFilter>
            <SearchFilter bind:currentSearchString></SearchFilter>
        </div>
        <div class="flex flex-1 flex-col md:flex-row overflow-hidden">
            <div class="basis-full md:basis-2/3 h-full overflow-auto border-r border-border">
                <EventTable bind:currentSelectedEvent {displayedEvents}></EventTable>
            </div>
            <div class="hidden md:block md:basis-1/3 h-full overflow-auto">
                <Sidebar {currentSelectedEvent}></Sidebar>
            </div>
        </div>
        <EventSheet bind:open={isSheetOpen} {currentSelectedEvent} />
    </div>
</div>
