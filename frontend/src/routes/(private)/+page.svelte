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
        DeliveryStatusTypes,
        SelectTypes,
        type WebHookEvent,
    } from '$lib/utils/types';
    import { onMount } from 'svelte';
    import { toast } from 'svelte-sonner';

    // Extend WebHookEvent locally to cache parsed timestamp avoids repeated Date.parse in hot sort path
    type CachedEvent = WebHookEvent & { _ts: number };

    // Primary source of truth keyed by event id for O(1) dedup + update
    const eventMap = new Map<string, CachedEvent>();
    const eventCap = 100;

    // Incremental stat counters
    const counts = { delivered: 0, retrying: 0, queued: 0, failed: 0 };

    // Reactive display state
    let events: WebHookEvent[] = $state([]);
    let totalEvents = $state(0);
    let totalDeliveredEvents = $state(0);
    let totalRetryingEvents = $state(0);
    let totalQueuedEvents = $state(0);
    let totalFailedEvents = $state(0);

    // Pause functionality
    let isPaused: boolean = $state(false);

    // Pagination
    let cursor: string | null = $state(null);
    let hasMore: boolean = $state(false);
    let loadingMore: boolean = $state(false);

    // Backend Connection
    let connState: ConnState = $state(ConnState.Connecting);

    // Filter / Search
    let currentSelectedOption: SelectTypes = $state(SelectTypes.All);
    let currentSearchString: string = $state('');

    // Throttled derived view recomputes at most every 100ms
    let displayedEvents: WebHookEvent[] = $state([]);
    let rebuildPending = false;

    // helper function to handle pause functionality
    function togglePause() {
        isPaused = !isPaused;
        if (!isPaused) {
            rebuildEventsArray();
        }
    }

    function scheduleRebuild() {
        if (rebuildPending) return;
        rebuildPending = true;
        setTimeout(() => {
            rebuildEventsArray();
            rebuildPending = false;
        }, 100);
    }

    // Re-run filter when user changes search/select
    $effect(() => {
        // Explicitly reference events so Svelte tracks it
        const currentEvents = events;
        displayedEvents = functions.getDisplayedEvents(
            currentSelectedOption,
            currentSearchString,
            currentEvents,
        );
    });

    // Sheet / Table
    let currentSelectedEvent: WebHookEvent | null = $state(null);
    let isSheetOpen: boolean = $state(false);
    $effect(() => {
        if (currentSelectedEvent && window.innerWidth < 1024) {
            isSheetOpen = true;
        }
    });

    // Stat Helpers
    function statKey(status: DeliveryStatusTypes): keyof typeof counts | null {
        if (status === DeliveryStatusTypes.Delivered) return 'delivered';
        if (status === DeliveryStatusTypes.Retrying) return 'retrying';
        if (status === DeliveryStatusTypes.Queued) return 'queued';
        if (status === DeliveryStatusTypes.Failed) return 'failed';
        return null;
    }

    function incrementStat(status: DeliveryStatusTypes) {
        const key = statKey(status);
        if (key) counts[key]++;
    }

    function decrementStat(status: DeliveryStatusTypes) {
        const key = statKey(status);
        if (key) counts[key]--;
    }

    // All writes to eventMap go through here keeps stat counters in sync
    function upsertEvent(event: WebHookEvent) {
        const existing = eventMap.get(event.id);
        if (existing) {
            // If status changed, correct the counter before overwriting
            if (existing.delivery_status !== event.delivery_status) {
                decrementStat(existing.delivery_status);
                incrementStat(event.delivery_status);
            }
            // if the display is currently paused patch the events array only for this specific event since it already exists
            if (isPaused) {
                const idx = events.findIndex((e) => e.id === event.id);
                if (idx !== -1) {
                    events[idx] = event;
                }
            }
        } else {
            incrementStat(event.delivery_status);
        }
        // Cache parsed timestamp once so the sort comparator is a pure subtraction
        eventMap.set(event.id, { ...event, _ts: Date.parse(event.created_at) });
    }

    // Rebuild the reactive events array from the map.
    // Called after bulk loads and after each SSE flush
    function rebuildEventsArray() {
        const sorted = Array.from(eventMap.values()).sort((a, b) => b._ts - a._ts);
        if (sorted.length > eventCap) {
            for (let i = eventCap; i < sorted.length; i++) {
                decrementStat(sorted[i].delivery_status);
                eventMap.delete(sorted[i].id);
            }
        }
        // This assignment triggers the $effect above automatically
        events = sorted.slice(0, eventCap);
        totalDeliveredEvents = counts.delivered;
        totalRetryingEvents = counts.retrying;
        totalQueuedEvents = counts.queued;
        totalFailedEvents = counts.failed;
        totalEvents = events.length;

        // Remove scheduleFilterUpdate() - the $effect is now the single source of truth
    }

    // Fetches a page of events. Upserts into map but does NOT rebuild array mid-load
    // caller is responsible for calling rebuildEventsArray() once when appropriate.
    async function loadPage(cur: string | null = null): Promise<WebHookEvent[]> {
        const url = cur ? `/api/events?cursor=${cur}` : '/api/events';
        const res = await fetch(url, { method: 'GET', credentials: 'include' });
        if (!res.ok) {
            if (res.status === 401) {
                const ok = await reAuthenticate();
                if (!ok) {
                    goto('/login');
                    return [];
                }
                // Retry once after re-auth
                return loadPage(cur);
            }
            throw new Error('Failed to load events');
        }
        const data: WebHookEvent[] = (await res.json()) ?? [];
        for (const event of data) {
            upsertEvent(event);
        }
        hasMore = cur ? data.length === 5 : data.length === 25;
        if (data.length > 0) {
            cursor = data[data.length - 1].created_at;
        }
        return data;
    }

    // Public load entry points rebuild array only once per call, not per-event
    async function loadEvents(): Promise<void> {
        try {
            await loadPage(null);
            rebuildEventsArray();
        } catch (err: any) {
            toast.error('Failed to load events', { position: 'top-center' });
        }
    }

    async function loadMore(): Promise<void> {
        if (!hasMore || loadingMore) return;
        loadingMore = true;
        try {
            await loadPage(cursor);
            rebuildEventsArray();
        } catch (err: any) {
            toast.error('Failed to load more webhooks', { position: 'top-center' });
        } finally {
            loadingMore = false;
        }
    }

    // Auth
    async function reAuthenticate(): Promise<boolean> {
        try {
            const res = await fetch('/api/refresh', { method: 'POST', credentials: 'include' });
            return res.ok;
        } catch {
            return false;
        }
    }

    // SSE
    onMount(() => {
        let es: EventSource | null = null;
        let destroyed = false;
        let authCheckTimeout: ReturnType<typeof setTimeout> | null = null;

        function connect() {
            if (destroyed) return;
            es = new EventSource('/api/events/stream', { withCredentials: true });
            es.onopen = () => {
                connState = ConnState.Connected;
                toast.info('Connected', { position: 'top-center' });
            };
            es.onmessage = (e) => {
                try {
                    const batch: WebHookEvent[] = JSON.parse(e.data);
                    if (!Array.isArray(batch) || batch.length === 0) return;
                    for (const event of batch) {
                        upsertEvent(event);
                    }
                    if (!isPaused) {
                        scheduleRebuild();
                    }
                } catch (err) {
                    // Ignore heartbeats
                }
            };
            es.onerror = () => {
                es?.close();
                es = null;
                connState = ConnState.Disconnected;
                toast.warning('Reconnecting ...');

                if (authCheckTimeout) clearTimeout(authCheckTimeout);
                authCheckTimeout = setTimeout(async () => {
                    if (destroyed) return;
                    const me = await fetch('/api/me', { credentials: 'include' });
                    if (!me.ok) {
                        const ok = await reAuthenticate();
                        if (!ok) {
                            goto('/login');
                            return;
                        }
                    }
                    connect();
                }, 3000);
            };
        }
        (async () => {
            try {
                await loadEvents();
                toast.info('Connecting ...', { position: 'top-center' });
                connect();
            } catch (err: any) {
                toast.error(err.message, { position: 'top-center' });
            }
        })();

        return () => {
            destroyed = true;
            if (authCheckTimeout) clearTimeout(authCheckTimeout);
            rebuildPending = false;
            es?.close();
        };
    });
</script>

<Navbar></Navbar>
<div>
    <div class="flex h-[calc(100vh-4rem)] flex-col">
        <div class="border-border flex flex-col border-b">
            <div class="flex flex-row items-center justify-between">
                <div class="border-border flex shrink-0 overflow-x-auto">
                    <StatCard
                        label="Total (7days)"
                        valueNumber={totalEvents}
                        valueNumberColor={''}
                    />
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
                <div class="flex shrink-0 items-center gap-2 px-3">
                    <button
                        onclick={togglePause}
                        class="px-3 py-1 text-sm rounded-md border border-zinc-700 bg-zinc-800 text-zinc-300 hover:bg-zinc-700 transition-colors"
                    >
                        {isPaused ? 'Resume' : 'Pause'}
                    </button>
                    <ConnIndicator {connState} />
                </div>
            </div>
        </div>
        <div
            class="border-border flex shrink-0 flex-col sm:flex-row items-start sm:items-center justify-between gap-2 border-b px-4 py-2.5"
        >
            <SelectFilter bind:currentSelectedOption></SelectFilter>
            <SearchFilter bind:currentSearchString></SearchFilter>
        </div>
        <div class="flex flex-1 flex-col md:flex-row overflow-hidden min-h-0">
            <div
                class="basis-full lg:basis-2/3 h-full overflow-hidden border-r border-border min-h-0"
            >
                <EventTable
                    bind:currentSelectedEvent
                    {displayedEvents}
                    {loadMore}
                    {loadingMore}
                    {hasMore}
                />
            </div>
            <div class="hidden lg:block lg:basis-1/3 h-full overflow-auto">
                <Sidebar {currentSelectedEvent}></Sidebar>
            </div>
        </div>
        <EventSheet bind:open={isSheetOpen} {currentSelectedEvent} />
    </div>
</div>
