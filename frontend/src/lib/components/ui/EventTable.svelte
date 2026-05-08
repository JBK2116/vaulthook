<script lang="ts">
    import * as Table from '$lib/components/ui/table/index.js';
    import * as functions from '$lib/utils/functions';
    import type { WebHookEvent } from '$lib/utils/types';
    import { onMount } from 'svelte';

    interface Props {
        currentSelectedEvent: WebHookEvent | null;
        displayedEvents: WebHookEvent[];
        loadMore: () => Promise<void>;
        loadingMore: boolean;
        hasMore: boolean;
    }
    let {
        currentSelectedEvent = $bindable(),
        displayedEvents,
        loadMore,
        loadingMore,
        hasMore,
    }: Props = $props();
    const userTimeZone: string = Intl.DateTimeFormat().resolvedOptions().timeZone;
    let container: HTMLDivElement;
    onMount(() => {
        function onScroll() {
            const { scrollTop, scrollHeight, clientHeight } = container;
            if (scrollHeight - scrollTop - clientHeight < 200 && !loadingMore && hasMore) {
                loadMore().then(() => {
                    // re-check in case content still doesn't fill threshold
                    if (
                        container.scrollHeight - container.scrollTop - container.clientHeight <
                            200 &&
                        hasMore
                    ) {
                        loadMore();
                    }
                });
            }
        }
        $effect(() => {
            if (hasMore && container && container.scrollHeight <= container.clientHeight) {
                loadMore();
            }
        });
        // If initial load didn't fill the container, trigger immediately
        function checkFill() {
            if (container.scrollHeight - container.clientHeight < 200 && hasMore) {
                loadMore().then(checkFill);
            }
        }
        container.addEventListener('scroll', onScroll, { passive: true });
        checkFill();
        return () => container.removeEventListener('scroll', onScroll);
    });
</script>

<div bind:this={container} style="overflow-anchor: none;" class="relative h-full overflow-auto">
    <Table.Root class="w-full">
        <Table.Header>
            <Table.Row>
                <Table.Head>Provider</Table.Head>
                <Table.Head>Status</Table.Head>
                <Table.Head>Event Type</Table.Head>
                <Table.Head>Received</Table.Head>
                <Table.Head>Response</Table.Head>
                <Table.Head>Retries</Table.Head>
            </Table.Row>
        </Table.Header>
        <Table.Body>
            {#each displayedEvents as event (event.id)}
                <Table.Row onclick={() => (currentSelectedEvent = event)}>
                    <Table.Cell
                        ><span
                            class="rounded-none border-border border-b p-3 px-2 py-0.5 text-xs font-medium"
                            style="background-color: {functions.getProviderBackgroundColor(
                                event.provider,
                            )}; color: {functions.getProviderTextColor(event.provider)};"
                        >
                            {event.provider}
                        </span></Table.Cell
                    >
                    <Table.Cell>
                        <span
                            class="flex items-center gap-1.5 text-xs font-medium {functions.getDeliveryStatusTextColor(
                                event.delivery_status,
                            )}"
                        >
                            <span
                                class="h-1.5 w-1.5 rounded-full bg-current {functions.getDeliveryStatusTextColor(
                                    event.delivery_status,
                                )}"
                            ></span>
                            {functions.capitalize(event.delivery_status)}
                        </span>
                    </Table.Cell>
                    <Table.Cell class="text-muted-foreground">
                        {event.event_type}
                    </Table.Cell>
                    <Table.Cell class="text-muted-foreground">
                        {functions.formatReceivedAtTimeForTable(event.received_at, userTimeZone)}
                    </Table.Cell>
                    <Table.Cell class={functions.getResponseCodeColor(event.response_code)}>
                        {event.response_code ? event.response_code : '-'}
                    </Table.Cell>
                    <Table.Cell class="text-muted-foreground">
                        {event.retry_count}
                    </Table.Cell>
                </Table.Row>
            {/each}
        </Table.Body>
    </Table.Root>
</div>
{#if loadingMore}
    <div class="text-muted-foreground py-2 text-center text-xs">Loading...</div>
{/if}
