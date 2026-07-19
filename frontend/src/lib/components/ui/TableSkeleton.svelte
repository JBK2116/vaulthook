<script lang="ts">
    interface Props {
        rows?: number;
    }
    let { rows = 8 }: Props = $props();
</script>

<div class="flex h-full flex-col overflow-hidden">
    <!-- Sticky header mimic -->
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
    <!-- Skeleton rows -->
    <div class="flex-1 overflow-hidden">
        {#each Array(rows) as _}
            <div
                class="border-border grid h-9 grid-cols-[1fr_1fr_2fr] lg:grid-cols-[1fr_1fr_2fr_1fr_1fr_1fr] items-center border-b px-4"
            >
                <div class="bg-muted h-3 w-14 animate-pulse rounded" />
                <div class="bg-muted h-3 w-16 animate-pulse rounded" />
                <div class="bg-muted h-3 w-24 animate-pulse rounded" />
                <div class="bg-muted hidden h-3 w-16 animate-pulse rounded lg:block" />
                <div class="bg-muted hidden h-3 w-8 animate-pulse rounded lg:block" />
                <div class="bg-muted hidden h-3 w-6 animate-pulse rounded lg:block" />
            </div>
        {/each}
    </div>
</div>
