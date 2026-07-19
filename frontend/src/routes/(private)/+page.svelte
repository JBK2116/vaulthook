<script lang="ts">
    import { goto } from '$app/navigation';
    import ConnIndicator from '$lib/components/ui/ConnIndicator.svelte';
    import EmptyState from '$lib/components/ui/EmptyState.svelte';
    import EventSheet from '$lib/components/ui/EventSheet.svelte';
    import EventTable from '$lib/components/ui/EventTable.svelte';
    import Navbar from '$lib/components/ui/Navbar.svelte';
    import SearchFilter from '$lib/components/ui/SearchFilter.svelte';
    import SelectFilter from '$lib/components/ui/SelectFilter.svelte';
    import Sidebar from '$lib/components/ui/Sidebar.svelte';
    import StatCard from '$lib/components/ui/StatCard.svelte';
    import TableSkeleton from '$lib/components/ui/TableSkeleton.svelte';
    import { reAuthenticate } from '$lib/utils/auth';
    import * as functions from '$lib/utils/functions';
    import {
        ConnState,
        DeliveryStatusColors,
        DeliveryStatusTypes,
        SelectTypes,
        type Stats,
        type WebHookEvent,
    } from '$lib/utils/types';
    import { Pause, Play } from '@lucide/svelte';
    import { onMount } from 'svelte';
    import { toast } from 'svelte-sonner';

    // Extend WebHookEvent locally to cache parsed timestamp avoids repeated Date.parse in hot sort path
    type CachedEvent = WebHookEvent & { _ts: number };

    // Primary source of truth keyed by event id for O(1) dedup + update
    const eventMap = new Map<string, CachedEvent>();
    const eventCap = 500;

    // Incremental stat counters
    const counts = { delivered: 0, retrying: 0, queued: 0, failed: 0 };

    // Reactive display state
    let events: WebHookEvent[] = $state([]);
    let totalDeliveredEvents = $state(0);
    let totalRetryingEvents = $state(0);
    let totalQueuedEvents = $state(0);
    let totalFailedEvents = $state(0);
    let totalEvents = $derived(
        Math.max(0, totalDeliveredEvents + totalRetryingEvents + totalQueuedEvents + totalFailedEvents),
    );

    // Pause functionality
    let isPaused: boolean = $state(false);
    let firstFlush: boolean = $state(false);

    // Overflow tracking — when the backend drops SSE events due to throughput,
    // we reload from the REST API to keep the displayed events consistent.
    let overflowCount: number = $state(0);
    let overflowTimeout: ReturnType<typeof setTimeout> | null = null;

    // Pagination
    let cursor: string | null = $state(null);
    let hasMore: boolean = $state(false);
    let loadingMore: boolean = $state(false);
    let initialLoading: boolean = $state(true);

    // Backend Connection
    let connState: ConnState = $state(ConnState.Connecting);

    // Filter / Search
    let currentSelectedOption: SelectTypes = $state(SelectTypes.All);
    let currentSearchString: string = $state('');

    // Throttled derived view recomputes at most every 100ms
    let displayedEvents: WebHookEvent[] = $state([]);
    let rebuildPending = false;

    // Scroll-away tracking: freeze table updates while user reads history
    let scrolledAway = $state(false);
    let pendingRebuild = false;

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

    // Called by EventTable when the user scrolls away from / back to the top.
    function handleScrollAway(away: boolean) {
        scrolledAway = away;
        if (!away && pendingRebuild && !isPaused) {
            rebuildEventsArray();
        }
    }
    // Keyboard shortcuts
    function handleKeydown(e: KeyboardEvent) {
        if (e.key === 'Escape' && currentSelectedEvent) {
            currentSelectedEvent = null;
            isSheetOpen = false;
        }
    }

    // Sheet reactivity
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

    // All writes to eventMap go through here. Only new events increment
    // the live counters; status changes are corrected by the 1s poll.
    function upsertEvent(event: WebHookEvent) {
        const existing = eventMap.get(event.id);
        if (existing) {
            // if the display is currently paused patch the events array only for this specific event since it already exists
            if (isPaused) {
                const idx = events.findIndex((e) => e.id === event.id);
                if (idx !== -1) {
                    events[idx] = event;
                }
            }
            // Always update the map so the event is up-to-date when unpausing.
            eventMap.set(event.id, { ...event, _ts: Date.parse(event.created_at) });
            return;
        }
        incrementStat(event.delivery_status);
        // Cache parsed timestamp once so the sort comparator is a pure subtraction
        eventMap.set(event.id, { ...event, _ts: Date.parse(event.created_at) });
    }

    // Rebuild the reactive events array from the map.
    // Called after bulk loads and after each SSE flush.
    // When the user has scrolled away from the top the display is frozen —
    // events still accumulate in the map and stats flush normally.
    function rebuildEventsArray() {
        const sorted = Array.from(eventMap.values()).sort((a, b) => b._ts - a._ts);

        // User is scrolled down or paused — don't disrupt their view.
        // Counts still flush so the stat cards stay live.
        if (scrolledAway || isPaused) {
            pendingRebuild = true;
            if (!firstFlush) {
                counts.delivered = 0;
                counts.failed = 0;
                counts.retrying = 0;
                counts.queued = 0;
                firstFlush = true;
            } else {
                totalDeliveredEvents += counts.delivered;
                totalFailedEvents += counts.failed;
                totalRetryingEvents += counts.retrying;
                totalQueuedEvents += counts.queued;
                counts.delivered = 0;
                counts.failed = 0;
                counts.retrying = 0;
                counts.queued = 0;
            }
            return;
        }

        if (sorted.length > eventCap) {
            for (let i = eventCap; i < sorted.length; i++) {
                eventMap.delete(sorted[i].id);
            }
        }
        // This assignment triggers the $effect above automatically
        events = sorted.slice(0, eventCap);
        pendingRebuild = false;
        if (!firstFlush) {
            // prevent overwriting call to loadStats
            counts.delivered = 0;
            counts.failed = 0;
            counts.retrying = 0;
            counts.queued = 0;
            firstFlush = true;
        } else {
            // append existing counts to last call to loadStats
            totalDeliveredEvents += counts.delivered;
            totalFailedEvents += counts.failed;
            totalRetryingEvents += counts.retrying;
            totalQueuedEvents += counts.queued;
            counts.delivered = 0;
            counts.failed = 0;
            counts.retrying = 0;
            counts.queued = 0;
        }

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

    // Loads the aggregate webhook processing stats from the database
    async function loadStats(): Promise<Stats> {
        const url = `/api/events/stats`;
        const res = await fetch(url, { method: 'GET', credentials: 'include' });
        if (!res.ok) {
            if (res.status === 401) {
                const ok = await reAuthenticate();
                if (!ok) {
                    goto('/login');
                    return { delivered: 0, failed: 0, queued: 0, retrying: 0 } as Stats;
                }
                // Retry once again after re-auth
                return loadStats();
            }
            throw new Error('Failed to load stats');
        }
        const data = (await res.json()) as Stats;
        totalDeliveredEvents = data.delivered;
        totalFailedEvents = data.failed;
        totalRetryingEvents = data.retrying;
        totalQueuedEvents = data.queued;
        return data;
    }

    // Public load entry points rebuild array only once per call, not per-event
    async function loadEvents(): Promise<void> {
        try {
            await loadPage(null);
            rebuildEventsArray();
        } catch (err: any) {
            toast.error('Failed to load events');
        } finally {
            initialLoading = false;
        }
    }

    async function loadMore(): Promise<void> {
        if (!hasMore || loadingMore) return;
        loadingMore = true;
        try {
            await loadPage(cursor);
            rebuildEventsArray();
        } catch (err: any) {
            toast.error('Failed to load more webhooks');
        } finally {
            loadingMore = false;
        }
    }

    // Auth — uses shared utility from $lib/utils/auth

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
            };
            es.onmessage = (e) => {
                // Ignore heartbeats and connection confirmations
                if (!e.data || e.data === '{}' || e.data.startsWith(':')) return;
                try {
                    const batch: WebHookEvent[] = JSON.parse(e.data);
                    if (!Array.isArray(batch) || batch.length === 0) return;
                    for (const event of batch) {
                        upsertEvent(event);
                    }
                    if (!isPaused) {
                        scheduleRebuild();
                    }
                } catch {
                    // Malformed JSON — ignore
                }
            };
            // Handle overflow events: backend dropped events from the SSE feed.
            // Reload from the REST API to fill the gap.
            es.addEventListener('overflow', (e: MessageEvent) => {
                try {
                    const data = JSON.parse(e.data);
                    overflowCount += data.count ?? 0;
                    // Debounce: reload at most once per 2 seconds
                    if (!overflowTimeout) {
                        overflowTimeout = setTimeout(async () => {
                            overflowTimeout = null;
                            const n = overflowCount;
                            overflowCount = 0;
                            toast.warning(
                                `Live feed skipped ${n} event${n !== 1 ? 's' : ''}. Reloading…`,
                            );
                            // Reload from the REST API to get a consistent view
                            await loadPage(null);
                            rebuildEventsArray();
                            toast.success('Feed resynced');
                        }, 2000);
                    }
                } catch {
                    // Malformed overflow JSON — ignore
                }
            });
            es.onerror = () => {
                es?.close();
                es = null;
                connState = ConnState.Disconnected;

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
                await loadStats();
                await loadEvents();
                connect();
            } catch (err: any) {
                toast.error(err.message);
            }
        })();

        // Poll every second to keep stats and rows accurate regardless of
        // SSE pipeline saturation. The DB is the source of truth for both.
        const POLL_MS = 1_000;
        let polling = false;
        const pollInterval = setInterval(async () => {
            if (polling) return; // skip if previous poll still in flight
            polling = true;
            try {
                // Stats: replace totals from DB, reset incremental counters.
                await loadStats();
                counts.delivered = 0;
                counts.failed = 0;
                counts.retrying = 0;
                counts.queued = 0;

                // Rows: refresh the most recent page so status changes from
                // worker retries are visible even when SSE drops updates.
                await loadPage(null);
                rebuildEventsArray();
            } catch {
                // Silently retry next interval.
            } finally {
                polling = false;
            }
        }, POLL_MS);

        return () => {
            destroyed = true;
            clearInterval(pollInterval);
            if (authCheckTimeout) clearTimeout(authCheckTimeout);
            rebuildPending = false;
            es?.close();
        };
    });
</script>

<svelte:head>
    <title>Dashboard</title>
    <meta name="description" content="Self-hostable webhook gateway" />
</svelte:head>

<svelte:window onkeydown={handleKeydown} />

<Navbar></Navbar>

<div
    class="fixed bottom-4 left-4 z-50 rounded-md border border-border bg-background/90 px-3 py-1.5 backdrop-blur-sm"
>
    <ConnIndicator {connState} />
</div>

<div>
    <div class="flex h-[calc(100vh-4rem)] flex-col">
        <div class="border-border flex flex-col border-b">
            <div class="flex flex-row items-center justify-between">
                <div
                    class="border-border flex shrink-0 overflow-x-auto snap-x snap-mandatory scroll-smooth [-webkit-overflow-scrolling:touch] relative"
                >
                    <StatCard label="Total" valueNumber={totalEvents} valueNumberColor={''} />
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
                        class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-md border border-border bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                    >
                        {#if isPaused}
                            <Play class="h-3.5 w-3.5" />
                        {:else}
                            <Pause class="h-3.5 w-3.5" />
                        {/if}
                    </button>
                </div>
            </div>
        </div>
        <div
            class="border-border flex shrink-0 flex-col sm:flex-row items-start sm:items-center justify-between gap-2 border-b px-4 py-2.5"
        >
            <SelectFilter bind:currentSelectedOption></SelectFilter>
            <SearchFilter bind:currentSearchString></SearchFilter>
        </div>
        {#if initialLoading}
            <div class="flex flex-1 flex-col md:flex-row overflow-hidden min-h-0">
                <div
                    class="basis-full lg:basis-2/3 h-full overflow-hidden border-r border-border min-h-0"
                >
                    <TableSkeleton />
                </div>
                <div class="hidden lg:flex lg:basis-1/3 h-full items-center justify-center">
                    <div class="bg-muted h-3 w-24 animate-pulse rounded"></div>
                </div>
            </div>
        {:else if totalEvents === 0}
            <div class="flex flex-1 items-center justify-center min-h-0">
                <EmptyState />
            </div>
        {:else}
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
                        onscrollaway={handleScrollAway}
                    />
                </div>
                <div class="hidden lg:block lg:basis-1/3 h-full overflow-auto">
                    <Sidebar {currentSelectedEvent}></Sidebar>
                </div>
            </div>
        {/if}
        <EventSheet bind:open={isSheetOpen} {currentSelectedEvent} />
    </div>
</div>
