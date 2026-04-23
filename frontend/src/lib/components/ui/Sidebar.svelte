<script lang="ts">
    import { ScrollArea } from '$lib/components/ui/scroll-area/index.js';
    import * as functions from '$lib/utils/functions';
    import type { WebHookEvent } from '$lib/utils/types';

    import Button from './button/button.svelte';

    interface Props {
        currentSelectedEvent: WebHookEvent | null;
    }
    let { currentSelectedEvent }: Props = $props();
    let activeTabIsPayload: boolean = $state(true);
    let userTimeZone: string = $derived(Intl.DateTimeFormat().resolvedOptions().timeZone);

    const copyEventJSON: () => void = () => {
        if (!currentSelectedEvent) {
            return;
        }
        activeTabIsPayload
            ? navigator.clipboard.writeText(JSON.stringify(currentSelectedEvent.payload, null, 2))
            : navigator.clipboard.writeText(JSON.stringify(currentSelectedEvent.headers, null, 2));
    };

    const copyEventId: () => void = () => {
        if (!currentSelectedEvent) {
            return;
        }
        navigator.clipboard.writeText(currentSelectedEvent.id);
    };
</script>

{#if currentSelectedEvent}
    <div class="flex h-full flex-col justify-between">
        <ScrollArea>
            <div class="border-border mb-4 border-b px-5 pt-4 pb-4">
                <h2 class="text-sm font-semibold">{currentSelectedEvent.event_type}</h2>
                <p class="text-muted-foreground text-xs">
                    {currentSelectedEvent.provider} · {currentSelectedEvent.id}
                </p>
            </div>
            <div class="mb-4 px-5">
                <p class="text-muted-foreground mb-2 text-[10px] tracking-widest uppercase">
                    Delivery
                </p>
                <div class="flex flex-col gap-2 text-xs">
                    <div class="flex items-center justify-between">
                        <span class="text-muted-foreground">Status</span>
                        <span
                            class="flex items-center gap-1.5 text-xs font-medium {functions.getDeliveryStatusTextColor(
                                currentSelectedEvent.delivery_status,
                            )}"
                        >
                            <span
                                class="h-1.5 w-1.5 rounded-full bg-current {functions.getDeliveryStatusTextColor(
                                    currentSelectedEvent.delivery_status,
                                )}"
                            ></span>
                            {functions.capitalize(currentSelectedEvent.delivery_status)}
                        </span>
                    </div>
                    <div class="flex items-center justify-between">
                        <span class="text-muted-foreground">Response</span>
                        <p
                            class={functions.getResponseCodeColor(
                                currentSelectedEvent.response_code,
                            )}
                        >
                            {currentSelectedEvent.response_code
                                ? currentSelectedEvent.response_code
                                : '-'}
                        </p>
                    </div>
                    <div class="flex items-center justify-between">
                        <span class="text-muted-foreground">Retries</span>
                        {currentSelectedEvent.retry_count}
                    </div>
                    <div class="flex items-center justify-between">
                        <span class="text-muted-foreground">Received</span>
                        <p class="text-muted-foreground">
                            {functions.formatReceivedAtTimeForSidebar(
                                currentSelectedEvent.received_at,
                                userTimeZone,
                            )}
                        </p>
                    </div>
                    <div class="flex flex-row justify-between gap-1">
                        <span class="text-muted-foreground">Forwarded to</span>
                        <p class="text-muted-foreground">{currentSelectedEvent.forwarded_to}</p>
                    </div>
                </div>
            </div>

            <div class="border-border mb-3 flex gap-2 border-b px-5 pb-2">
                <Button
                    variant={activeTabIsPayload ? 'outline' : 'ghost'}
                    size="sm"
                    onclick={() => (activeTabIsPayload = true)}
                >
                    Payload
                </Button>
                <Button
                    variant={activeTabIsPayload ? 'ghost' : 'outline'}
                    size="sm"
                    onclick={() => (activeTabIsPayload = false)}
                >
                    Headers
                </Button>
            </div>
            <div
                class="border-border overflow-auto rounded-sm border mx-5 p-3 max-h-32 md:max-h-64"
            >
                {#if activeTabIsPayload}
                    <pre class="text-xs whitespace-pre">{JSON.stringify(
                            currentSelectedEvent.payload,
                            null,
                            2,
                        )}</pre>
                {:else}
                    <pre class="text-xs whitespace-pre">{JSON.stringify(
                            currentSelectedEvent.headers,
                            null,
                            2,
                        )}</pre>
                {/if}
            </div>
        </ScrollArea>
        <div class="border-border flex gap-2 border-t px-5 pt-3 pb-4">
            <Button variant="outline" size="sm">Replay</Button>
            <Button variant="outline" size="sm" onclick={copyEventId}>Copy ID</Button>
            <Button variant="outline" size="sm" onclick={copyEventJSON}
                >Copy {#if activeTabIsPayload}
                    Payload
                {:else}
                    Headers
                {/if}</Button
            >
        </div>
    </div>
{/if}
