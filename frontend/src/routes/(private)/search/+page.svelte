<script lang="ts">
    import { goto } from '$app/navigation';
    import EmptyState from '$lib/components/ui/EmptyState.svelte';
    import EventTable from '$lib/components/ui/EventTable.svelte';
    import MultiSelect from '$lib/components/ui/MultiSelect.svelte';
    import Navbar from '$lib/components/ui/Navbar.svelte';
    import Sidebar from '$lib/components/ui/Sidebar.svelte';
    import Button from '$lib/components/ui/button/button.svelte';
    import Calendar from '$lib/components/ui/calendar/calendar.svelte';
    import { Input } from '$lib/components/ui/input/index.js';
    import * as Popover from '$lib/components/ui/popover/index.js';
    import * as Switch from '$lib/components/ui/switch/index.js';
    import { reAuthenticate } from '$lib/utils/auth';
    import { firstToUpper, formatDateTime } from '$lib/utils/functions';
    import {
        DeliveryStatusTypes,
        ProviderTypes,
        type SearchPayload,
        SearchTypes,
        type WebHookEvent,
    } from '$lib/utils/types';
    import { type CalendarDate, getLocalTimeZone } from '@internationalized/date';
    import { ChevronDown, LoaderCircle } from '@lucide/svelte';

    // Quick Lookup state
    let searchWebhookId = $state('');
    let searchEventId = $state('');

    // Filter state
    const PROVIDER_OPTIONS = [ProviderTypes.Stripe, ProviderTypes.Github];
    const STATUS_OPTIONS = [
        DeliveryStatusTypes.Delivered,
        DeliveryStatusTypes.Failed,
        DeliveryStatusTypes.Queued,
        DeliveryStatusTypes.Retrying,
    ];

    // Filter Options
    let selectedProviders = $state<string[]>([]);
    let eventType = $state('');
    let selectedStatuses = $state<string[]>([]);
    let responseCode = $state('');
    let fromDate: CalendarDate | undefined = $state(undefined);
    let fromTime = $state('00:00:00');
    let fromOpen = $state(false);
    let toDate: CalendarDate | undefined = $state(undefined);
    let toTime = $state('23:59:00');
    let toOpen = $state(false);
    let payloadSearch = $state('');
    let hasRetries = $state(false);
    let hasError = $state(false);

    // Mutual exclusion
    let quickLookupActive = $derived(searchWebhookId.trim() !== '' || searchEventId.trim() !== '');
    let filtersActive = $derived(
        selectedProviders.length > 0 ||
            eventType.trim() !== '' ||
            selectedStatuses.length > 0 ||
            responseCode.trim() !== '' ||
            fromDate !== undefined ||
            toDate !== undefined ||
            payloadSearch.trim() !== '' ||
            hasRetries ||
            hasError,
    );

    let canSearch = $derived(quickLookupActive || filtersActive);

    let filterError = $state('');
    let lookupError = $state('');
    let lookupLoading = $state(false);
    let filterLoading = $state(false);

    // Results placeholder
    let displayedEvents: WebHookEvent[] = $state([]);
    let currentSelectedEvent: WebHookEvent | null = $state(null);

    /** Builds the SearchPayload using the provided options */
    function buildFormState(type: SearchTypes) {
        return {
            // type of search to execute
            type: type,
            // quick lookup options
            webhook_id: searchWebhookId.trim() || null,
            event_id: searchEventId.trim() || null,
            // filter options
            providers: selectedProviders,
            event_type: eventType.trim() || null,
            delivery_statuses: selectedStatuses.map((s) => s.toLowerCase()),
            response_code: responseCode.trim() !== '' ? Number(responseCode.trim()) : null,
            from_time: formatDateTime(fromDate, fromTime),
            to_time: formatDateTime(toDate, toTime),
            payload_search: payloadSearch.trim() || null,
            has_retries: hasRetries,
            has_error: hasError,
        } as SearchPayload;
    }

    const searchURL = `/api/events`;

    /** Queries the database for events using the provided lookup options */
    async function handleLookup(): Promise<void> {
        lookupLoading = true;
        try {
            if (!validateLookup()) {
                return;
            }
            clearLookupError();
            clearFilterError();
            const form = buildFormState(SearchTypes.Lookup);
            const payload = JSON.stringify(form);
            let response = await fetch(searchURL, {
                method: 'POST',
                credentials: 'include',
                headers: { 'Content-Type': 'application/json' },
                body: payload,
            });
            if (response.status === 401) {
                const ok = await reAuthenticate();
                if (!ok) {
                    await goto('/login');
                    return;
                }
                response = await fetch(searchURL, {
                    method: 'POST',
                    credentials: 'include',
                    headers: { 'Content-Type': 'application/json' },
                    body: payload,
                });
            }
            if (!response.ok) {
                if (response.status === 400) {
                    const text = await response.text();
                    lookupError = firstToUpper(text);
                } else if (response.status === 500) {
                    lookupError = 'Failed to execute search. Please try again.';
                } else {
                    lookupError = 'Failed to execute search. Please try again soon.';
                }
                return;
            }
            const body = (await response.json()) as WebHookEvent[] | null;
            displayedEvents = [];
            if (Array.isArray(body) && body.length > 0) {
                displayedEvents.push(...body);
            } else {
                lookupError = 'No results found.';
            }
        } catch (err: any) {
            lookupError = 'Failed to execute search. Please try again soon.';
            console.log(err);
        } finally {
            lookupLoading = false;
        }
    }

    /** Queries the database for events using the provided filter options */
    async function handleFilter(): Promise<void> {
        filterLoading = true;
        try {
            if (!validateFilter()) {
                return;
            }
            clearLookupError();
            clearFilterError();
            const form = buildFormState(SearchTypes.Filter);
            const payload = JSON.stringify(form);
            let response = await fetch(searchURL, {
                method: 'POST',
                credentials: 'include',
                headers: { 'Content-Type': 'application/json' },
                body: payload,
            });
            if (response.status === 401) {
                const ok = await reAuthenticate();
                if (!ok) {
                    await goto('/login');
                    return;
                }
                response = await fetch(searchURL, {
                    method: 'POST',
                    credentials: 'include',
                    headers: { 'Content-Type': 'application/json' },
                    body: payload,
                });
            }
            if (!response.ok) {
                if (response.status === 400) {
                    const text = await response.text();
                    filterError = firstToUpper(text);
                } else if (response.status === 500) {
                    filterError = 'Failed to execute search. Please try again.';
                } else {
                    filterError = 'Failed to execute search. Please try again soon.';
                }
                return;
            }
            const body = (await response.json()) as WebHookEvent[] | null;
            displayedEvents = [];
            if (Array.isArray(body) && body.length > 0) {
                displayedEvents.push(...body);
            } else {
                filterError = 'No results found.';
            }
        } catch (error: any) {
            return;
        } finally {
            filterLoading = false;
        }
    }

    /** Validates the lookup options section */
    function validateLookup(): boolean {
        const hookId = searchWebhookId.trim();
        const eventId = searchEventId.trim();
        if (!hookId && !eventId) {
            lookupError = '';
            return false;
        }
        if (hookId.length > 255) {
            lookupError = 'Webhook ID must be 255 characters or fewer.';
            return false;
        }
        if (eventId.length > 255) {
            lookupError = 'Event ID must be 255 characters or fewer.';
            return false;
        }
        return true;
    }

    /** Validates the filter options section */
    function validateFilter(): boolean {
        if (selectedProviders.length === 0) {
            filterError = 'Please select at least one provider.';
            return false;
        }
        if (selectedStatuses.length === 0) {
            filterError = 'Please select at least one delivery status.';
            return false;
        }
        if (responseCode) {
            if (responseCode.length !== 3 || !/^[0-9]+$/.test(responseCode)) {
                filterError = 'Response code must be a 3-digit number (e.g. 200, 404).';
                return false;
            }
            const responseNum = Number(responseCode);
            if (responseNum < 100 || responseNum > 511) {
                filterError = 'Response code must be between 100 and 511.';
                return false;
            }
        }
        if ((fromDate && !toDate) || (toDate && !fromDate)) {
            filterError = 'Both "From" and "To" dates must be set together.';
            return false;
        }
        if (fromDate && toDate) {
            const fromIso = formatDateTime(fromDate, fromTime);
            if (!fromIso) {
                filterError = 'Invalid "From" date or time.';
                return false;
            }
            const toIso = formatDateTime(toDate, toTime);
            if (!toIso) {
                filterError = 'Invalid "To" date or time.';
                return false;
            }
            const fromDateTime = new Date(fromIso);
            const toDateTime = new Date(toIso);
            if (toDateTime < fromDateTime) {
                filterError = '"To" date must be after "From" date.';
                return false;
            }
        }
        if (payloadSearch.trim().length > 255) {
            filterError = 'Payload search must be 255 characters or fewer.';
            return false;
        }
        if (eventType.trim().length > 255) {
            filterError = 'Event type must be 255 characters or fewer.';
            return false;
        }
        return true;
    }

    /** Clears all lookup and filter selections */
    function clearSelections(): void {
        clearLookupError();
        clearFilterError();
        selectedProviders = [];
        eventType = '';
        selectedStatuses = [];
        responseCode = '';
        fromDate = undefined;
        fromTime = '00:00:00';
        toDate = undefined;
        toTime = '23:59:00';
        payloadSearch = '';
        hasRetries = false;
        hasError = false;
    }

    /** Helper function to reset lookup errors */
    function clearLookupError() {
        lookupError = '';
    }

    /** Helper function to reset filter errors */
    function clearFilterError() {
        filterError = '';
    }
</script>

<svelte:head>
    <title>Search</title>
</svelte:head>

<Navbar />

<div class="flex h-[calc(100vh-4rem)] flex-col">
    <!-- Search area -->
    <div class="border-border shrink-0 overflow-auto border-b">
        <div class="mx-auto max-w-4xl px-6 py-5">
            <!-- Quick Lookup -->
            <section
                class="border-border rounded-sm border p-5 transition-opacity"
                class:pointer-events-none={filtersActive}
                class:opacity-40={filtersActive}
            >
                <h2
                    class="text-muted-foreground mb-4 flex items-center justify-between text-[10px] font-medium tracking-widest uppercase"
                >
                    <span>Quick Lookup</span>
                    {#if lookupError}
                        <span
                            class="text-[12px] font-normal normal-case tracking-normal text-red-400"
                            >{lookupError}</span
                        >
                    {/if}
                </h2>
                <div class="flex flex-col gap-3 sm:flex-row sm:items-end">
                    <label class="flex flex-1 flex-col">
                        <span class="text-muted-foreground mb-1 block text-[10px] uppercase">
                            Webhook ID
                        </span>
                        <Input
                            type="text"
                            placeholder="e.g. a1b2c3d4-..."
                            bind:value={searchWebhookId}
                        />
                    </label>
                    <label class="flex flex-1 flex-col">
                        <span class="text-muted-foreground mb-1 block text-[10px] uppercase">
                            Event ID
                        </span>
                        <Input type="text" placeholder="e.g. evt_..." bind:value={searchEventId} />
                    </label>
                    <Button
                        variant="outline"
                        size="default"
                        onclick={handleLookup}
                        disabled={!canSearch || filtersActive || lookupLoading}
                    >
                        {#if lookupLoading}
                            <LoaderCircle class="animate-spin" />
                        {:else}
                            Find
                        {/if}
                    </Button>
                </div>
            </section>

            <!-- Divider -->
            <div class="my-5 flex items-center gap-3">
                <div class="border-border flex-1 border-t"></div>
                <span class="text-muted-foreground text-[10px] uppercase tracking-widest"
                    >or filter by</span
                >
                <div class="border-border flex-1 border-t"></div>
            </div>

            <!-- Filters -->
            <section
                class="border-border rounded-sm border p-5 transition-opacity"
                class:pointer-events-none={quickLookupActive}
                class:opacity-40={quickLookupActive}
            >
                <h2
                    class="text-muted-foreground mb-4 text-[10px] font-medium tracking-widest uppercase"
                >
                    Filters
                </h2>

                <!-- Row 1 -->
                <div class="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
                    <div>
                        <span class="text-muted-foreground mb-1 block text-[10px] uppercase">
                            Provider
                        </span>
                        <MultiSelect
                            options={PROVIDER_OPTIONS}
                            bind:selected={selectedProviders}
                            placeholder="All providers"
                            class="w-full"
                        />
                    </div>
                    <label class="flex flex-col">
                        <span class="text-muted-foreground mb-1 block text-[10px] uppercase">
                            Event Type
                        </span>
                        <Input
                            type="text"
                            placeholder="e.g. payment_intent.succeeded"
                            bind:value={eventType}
                        />
                    </label>
                    <div>
                        <span class="text-muted-foreground mb-1 block text-[10px] uppercase">
                            Delivery Status
                        </span>
                        <MultiSelect
                            options={STATUS_OPTIONS.map((f) => firstToUpper(f))}
                            bind:selected={selectedStatuses}
                            placeholder="All statuses"
                            class="w-full"
                        />
                    </div>
                </div>

                <!-- Row 2 -->
                <div class="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2">
                    <label class="flex flex-col">
                        <span class="text-muted-foreground mb-1 block text-[10px] uppercase">
                            Response Code
                        </span>
                        <Input
                            type="text"
                            placeholder="e.g. 200, 404, 501"
                            bind:value={responseCode}
                        />
                    </label>
                    <div>
                        <span class="text-muted-foreground mb-1 block text-[10px] uppercase">
                            Date Range
                        </span>
                        <div class="flex flex-col gap-2">
                            <!-- From -->
                            <div class="flex items-center gap-1.5">
                                <span class="text-muted-foreground w-10 shrink-0 text-[10px]"
                                    >From</span
                                >
                                <Popover.Root bind:open={fromOpen}>
                                    <Popover.Trigger class="flex-1">
                                        {#snippet child({ props })}
                                            <button
                                                {...props}
                                                class="border-border dark:bg-input/30 dark:hover:bg-input/50 flex h-8 w-full items-center justify-between gap-1 rounded-none border bg-transparent px-2 text-xs transition-colors outline-none select-none"
                                            >
                                                {#if fromDate}
                                                    {fromDate
                                                        .toDate(getLocalTimeZone())
                                                        .toLocaleDateString()}
                                                {:else}
                                                    <span class="text-muted-foreground"
                                                        >Pick date</span
                                                    >
                                                {/if}
                                                <ChevronDown
                                                    class="text-muted-foreground size-3.5 shrink-0"
                                                />
                                            </button>
                                        {/snippet}
                                    </Popover.Trigger>
                                    <Popover.Content
                                        class="w-auto overflow-hidden p-0"
                                        align="start"
                                    >
                                        <Calendar
                                            type="single"
                                            bind:value={fromDate}
                                            onValueChange={() => {
                                                fromOpen = false;
                                            }}
                                            captionLayout="dropdown"
                                        />
                                    </Popover.Content>
                                </Popover.Root>
                                <Input
                                    type="time"
                                    step="1"
                                    bind:value={fromTime}
                                    class="border-border dark:bg-input/30 h-8 w-28 shrink-0 rounded-none border px-2 text-xs [&::-webkit-calendar-picker-indicator]:hidden [&::-webkit-calendar-picker-indicator]:appearance-none"
                                />
                            </div>
                            <!-- To -->
                            <div class="flex items-center gap-1.5">
                                <span class="text-muted-foreground w-10 shrink-0 text-[10px]"
                                    >To</span
                                >
                                <Popover.Root bind:open={toOpen}>
                                    <Popover.Trigger class="flex-1">
                                        {#snippet child({ props })}
                                            <button
                                                {...props}
                                                class="border-border dark:bg-input/30 dark:hover:bg-input/50 flex h-8 w-full items-center justify-between gap-1 rounded-none border bg-transparent px-2 text-xs transition-colors outline-none select-none"
                                            >
                                                {#if toDate}
                                                    {toDate
                                                        .toDate(getLocalTimeZone())
                                                        .toLocaleDateString()}
                                                {:else}
                                                    <span class="text-muted-foreground"
                                                        >Pick date</span
                                                    >
                                                {/if}
                                                <ChevronDown
                                                    class="text-muted-foreground size-3.5 shrink-0"
                                                />
                                            </button>
                                        {/snippet}
                                    </Popover.Trigger>
                                    <Popover.Content
                                        class="w-auto overflow-hidden p-0"
                                        align="start"
                                    >
                                        <Calendar
                                            type="single"
                                            bind:value={toDate}
                                            onValueChange={() => {
                                                toOpen = false;
                                            }}
                                            captionLayout="dropdown"
                                        />
                                    </Popover.Content>
                                </Popover.Root>
                                <Input
                                    type="time"
                                    step="1"
                                    bind:value={toTime}
                                    class="border-border dark:bg-input/30 h-8 w-28 shrink-0 rounded-none border px-2 text-xs [&::-webkit-calendar-picker-indicator]:hidden [&::-webkit-calendar-picker-indicator]:appearance-none"
                                />
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Row 3 -->
                <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                    <label class="flex flex-1 flex-col">
                        <span class="text-muted-foreground mb-1 block text-[10px] uppercase">
                            Payload Search
                        </span>
                        <Input
                            type="text"
                            placeholder="search payload contents..."
                            bind:value={payloadSearch}
                        />
                    </label>
                    <div class="flex items-center gap-6">
                        <label class="flex cursor-pointer items-center gap-2 text-xs">
                            <Switch.Root bind:checked={hasRetries} size="sm" />
                            Has retries
                        </label>
                        <label class="flex cursor-pointer items-center gap-2 text-xs">
                            <Switch.Root bind:checked={hasError} size="sm" />
                            Has error
                        </label>
                    </div>
                </div>

                <!-- Action buttons -->
                <div class="border-border mt-5 flex items-center gap-2 border-t pt-4">
                    <Button
                        variant="default"
                        size="default"
                        onclick={handleFilter}
                        disabled={!canSearch || filterLoading}
                    >
                        {#if filterLoading}
                            <LoaderCircle class="animate-spin" />
                        {:else}
                            Search
                        {/if}
                    </Button>
                    <Button
                        variant="ghost"
                        size="default"
                        onclick={clearSelections}
                        disabled={!filtersActive}
                    >
                        Clear filters
                    </Button>
                    {#if filterError}
                        <span class="ml-auto text-[12px] text-red-400">{filterError}</span>
                    {/if}
                </div>
            </section>
        </div>
    </div>

    <!-- Results area -->
    <div class="flex flex-1 flex-col md:flex-row overflow-hidden min-h-0">
        <div class="basis-full lg:basis-2/3 h-full overflow-hidden border-r border-border min-h-0">
            {#if displayedEvents.length === 0}
                <EmptyState
                    title="Enter search criteria above"
                    description="Fill in the Quick Lookup or Filters section and click Search to find events."
                    icon={true}
                />
            {:else}
                <EventTable
                    bind:currentSelectedEvent
                    {displayedEvents}
                    loadMore={async () => {}}
                    loadingMore={false}
                    hasMore={false}
                />
            {/if}
        </div>
        <div class="hidden lg:block lg:basis-1/3 h-full overflow-auto">
            <Sidebar {currentSelectedEvent} />
        </div>
    </div>
</div>
