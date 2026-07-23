<script lang="ts">
    import * as functions from '$lib/utils/functions';
    import type { WebHookEvent } from '$lib/utils/types';
    import { createVirtualizer } from '@tanstack/svelte-virtual';

    interface Props {
        currentSelectedEvent: WebHookEvent | null;
        displayedEvents: WebHookEvent[];
        loadMore: () => Promise<void>;
        loadingMore: boolean;
        hasMore: boolean;
        onscrollaway?: (away: boolean) => void;
    }
    let {
        currentSelectedEvent = $bindable(),
        displayedEvents,
        loadMore,
        loadingMore,
        hasMore,
        onscrollaway,
    }: Props = $props();

    const ROW_HEIGHT = 36;
    const userTimeZone: string = Intl.DateTimeFormat().resolvedOptions().timeZone;

    let scrollEl: HTMLDivElement;

    // Notify parent when the user scrolls away from / back to the top so the
    // live feed can be frozen while they're reading history.
    let wasAway = $state(false);
    const SCROLL_AWAY_THRESHOLD = 50;

    function handleScroll() {
        if (!scrollEl || !onscrollaway) return;
        const away = scrollEl.scrollTop > SCROLL_AWAY_THRESHOLD;
        if (away !== wasAway) {
            wasAway = away;
            onscrollaway(away);
        }
    }

    let prevTopItemId: string | null = null;

    $effect(() => {
        const currentEvents = displayedEvents;
        // Only apply scroll correction if the user has actively scrolled down past the live-feed threshold
        if (prevTopItemId && scrollEl && scrollEl.scrollTop > 5) {
            // Find how many new items were prepended by locating the previous first item
            const shiftAmount = currentEvents.findIndex((e) => e.id === prevTopItemId);
            if (shiftAmount > 0) {
                // Push the scroll position down by the exact height of the newly inserted items
                scrollEl.scrollTop += shiftAmount * ROW_HEIGHT;
            }
        }
        // Update the tracker for the next batch cycle
        prevTopItemId = currentEvents.length > 0 ? currentEvents[0].id : null;
    });

    let virtualizer = $state(
        createVirtualizer<HTMLDivElement, HTMLDivElement>({
            count: 0,
            getScrollElement: () => scrollEl,
            estimateSize: () => ROW_HEIGHT,
            overscan: 10,
        }),
    );

    $effect(() => {
        // Recreate after DOM commit so scrollEl is bound and has dimensions
        virtualizer = createVirtualizer<HTMLDivElement, HTMLDivElement>({
            count: displayedEvents.length,
            getScrollElement: () => scrollEl,
            estimateSize: () => ROW_HEIGHT,
            overscan: 10,
        });
    });

    $effect(() => {
        const items = $virtualizer.getVirtualItems();
        if (!items.length) return;
        const last = items[items.length - 1];
        if (last.index >= displayedEvents.length - 5 && hasMore && !loadingMore) {
            loadMore();
        }
    });
</script>

<!-- HTML template remains completely unchanged -->

<!-- Sticky header -->
<div class="border-border bg-background sticky top-0 z-10 border-b">
    <div
        class="text-muted-foreground grid h-10 grid-cols-[1fr_1fr_2fr] lg:grid-cols-[1fr_1fr_2fr_1fr_1fr_1fr] items-center px-4 text-xs font-medium"
    >
        <span>Provider</span>
        <span>Status</span>
        <span>Event Type</span>
        <span class="hidden lg:block">Received</span>
        <span class="hidden lg:block">Response</span>
        <span class="hidden lg:block">Retries</span>
    </div>
</div>
<!-- Virtual scroll container -->
<div
    bind:this={scrollEl}
    onscroll={handleScroll}
    class="relative h-full overflow-auto"
    style="overflow-anchor: none;"
>
    <div style="height: {$virtualizer.getTotalSize()}px; position: relative;">
        {#each $virtualizer.getVirtualItems() as row (row.key)}
            {@const event = displayedEvents[row.index]}
            <div
                style="position: absolute; top: 0; left: 0; width: 100%; height: {ROW_HEIGHT}px; transform: translateY({row.start}px);"
                class="border-border grid cursor-pointer grid-cols-[1fr_1fr_2fr] lg:grid-cols-[1fr_1fr_2fr_1fr_1fr_1fr] items-center border-b px-4 text-sm transition-colors
           {currentSelectedEvent?.id === event.id ? 'bg-muted' : 'hover:bg-muted/50'}"
                role="button"
                tabindex="0"
                onclick={() => (currentSelectedEvent = event)}
                onkeydown={(e) => e.key === 'Enter' && (currentSelectedEvent = event)}
            >
                <!-- Provider -->
                <span
                    class="w-fit rounded-none border-b px-2 py-0.5 text-xs font-medium"
                    style="background-color: {functions.getProviderBackgroundColor(
                        event.provider,
                    )}; color: {functions.getProviderTextColor(event.provider)};"
                >
                    {event.provider}
                </span>
                <!-- Status -->
                <span
                    class="flex items-center gap-1.5 text-xs font-medium {functions.getDeliveryStatusTextColor(
                        event.delivery_status,
                    )}"
                >
                    <span class="h-1.5 w-1.5 rounded-full bg-current"></span>
                    {functions.capitalize(event.delivery_status)}
                </span>
                <!-- Event Type -->
                <span class="text-xs text-muted-foreground truncate">{event.event_type}</span>
                <!-- Received -->
                <span class="hidden lg:block text-xs text-muted-foreground">
                    {functions.formatReceivedAtTimeForTable(event.received_at, userTimeZone)}
                </span>
                <!-- Response -->
                <span
                    class="hidden lg:block text-xs {functions.getResponseCodeColor(
                        event.response_code,
                    )}"
                >
                    {event.response_code ?? '-'}
                </span>
                <!-- Retries -->
                <span class="hidden lg:block text-xs text-muted-foreground"
                    >{event.retry_count}</span
                >
            </div>
        {/each}
    </div>
    {#if loadingMore}
        <div class="text-muted-foreground py-2 text-center text-xs">Loading...</div>
    {/if}
</div>
