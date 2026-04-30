<script lang="ts">
    import { goto } from '$app/navigation';
    import ConnIndicator from '$lib/components/ui/ConnIndicator.svelte';
    import EventSheet from '$lib/components/ui/EventSheet.svelte';
    import EventTable from '$lib/components/ui/EventTable.svelte';
    import Navbar from '$lib/components/ui/Navbar.svelte';
    import SearchFilter from '$lib/components/ui/SearchFilter.svelte';
    import SelectFilter from '$lib/components/ui/SelectFilter.svelte';
    import Sidebar from '$lib/components/ui/Sidebar.svelte';
    import StatCard from '$lib/components/ui/StatCard.svelte';
    import * as functions from '$lib/utils/functions';
    import {
        ConnState,
        DeliveryStatusColors,
        SelectTypes,
        type WebHookEvent,
    } from '$lib/utils/types';
    import { onMount } from 'svelte';
    import { toast } from 'svelte-sonner';

    // Event Manipulation
    let events: WebHookEvent[] = $state([]);
    let totalEvents = $derived(functions.getTotalEvents(events));
    let totalDeliveredEvents = $derived(functions.getTotalDeliveredEvents(events));
    let totalRetryingEvents = $derived(functions.getTotalRetryingEvents(events));
    let totalQueuedEvents = $derived(functions.getTotalQueuedEvents(events));
    let totalFailedEvents = $derived(functions.getTotalFailedEvents(events));

    // Connection State
    let connState: ConnState = $state(ConnState.Connecting);

    // Select & Search Handling
    let currentSelectedOption: SelectTypes = $state(SelectTypes.All);
    let currentSearchString: string = $state('');

    // Table State
    let currentSelectedEvent: WebHookEvent | null = $state(null);
    let displayedEvents: WebHookEvent[] = $derived(
        functions.getDisplayedEvents(currentSelectedOption, currentSearchString, events),
    );

    // Sheet State
    let isSheetOpen: boolean = $state(false);
    $effect(() => {
        if (currentSelectedEvent && window.innerWidth < 768) {
            isSheetOpen = true;
        }
    });

    // SSE Handling
    onMount(() => {
        let es: EventSource;
        let authCheckTimeout: any;
        (async () => {
            try {
                // load all events first
                const res = await fetch('/api/events', { credentials: 'include' });
                if (!res.ok) throw new Error('Failed to load events');
                events = (await res.json()) ?? [];
                // sse logic
                toast.info('Connecting ...', { position: 'top-center' });
                es = new EventSource('/api/events/stream', { withCredentials: true });
                es.onopen = () => {
                    toast.info('Connected', { position: 'top-center' });
                    connState = ConnState.Connected;
                };
                // each message comes in the form of {"data": "<webhook object>"}
                es.onmessage = (e) => {
                    const event: WebHookEvent = JSON.parse(e.data);
                    events = [event, ...events.filter((ev) => ev.id !== event.id)];
                };
                es.onerror = () => {
                    clearTimeout(authCheckTimeout);
                    toast.warning('Reconnecting ...');
                    connState = ConnState.Disconnected;
                    authCheckTimeout = setTimeout(async () => {
                        // check access token status
                        const me = await fetch('/api/me', {
                            credentials: 'include',
                            method: 'GET',
                        });
                        if (me.ok) {
                            connState = ConnState.Connected;
                            return;
                        }
                        // check refresh token status
                        const refresh = await fetch('/api/refresh', {
                            credentials: 'include',
                            method: 'POST',
                        });
                        if (refresh.ok) {
                            connState = ConnState.Connected;
                            return;
                        }
                        // user is unauthenticated
                        if (es) {
                            es.close();
                        }
                        goto('/login');
                    }, 3000);
                };
            } catch (err: any) {
                toast.error(err.message, { position: 'top-center' });
            }
        })();
        return () => es?.close();
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
            <ConnIndicator {connState}></ConnIndicator>
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
