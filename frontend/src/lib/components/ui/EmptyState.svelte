<script lang="ts">
    import { Webhook } from '@lucide/svelte';

    interface Props {
        title?: string;
        description?: string;
        icon?: boolean;
    }

    let {
        title = 'No events yet',
        description = 'Webhook events will appear here in real time as they arrive.',
        icon = true,
    }: Props = $props();
</script>

<div class="flex h-full flex-col items-center justify-center gap-3 px-6 py-12 text-center">
    {#if icon}
        <div class="text-muted-foreground/40">
            <Webhook class="h-12 w-12" />
        </div>
    {/if}
    <h3 class="text-sm font-medium text-muted-foreground">{title}</h3>
    <p class="max-w-xs text-xs text-muted-foreground/60">{description}</p>
</div>
