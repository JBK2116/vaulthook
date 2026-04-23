<script lang="ts">
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
    import { Input } from '$lib/components/ui/input';
    import { Label } from '$lib/components/ui/label';
    import type { Provider } from '$lib/utils/types';
    import { Check, Eye, EyeOff, Pencil, X } from '@lucide/svelte';

    interface Props {
        provider: Provider;
    }
    let { provider }: Props = $props();
    // card state management
    let editing = $state(false);
    let showSecret = $state(false);
    let draft = $state({ signing_secret: '', destination_url: '' });
    // editing and view management
    function startEdit() {
        draft = {
            signing_secret: provider.signing_secret,
            destination_url: provider.destination_url,
        };
        editing = true;
    }

    function cancel() {
        editing = false;
    }

    function save() {
        // TODO: Call the backend from here
        provider = { ...provider, ...draft };
        editing = false;
    }
</script>

<Card.Root class="w-full">
    <Card.Header class="flex flex-row items-start justify-between space-y-0 pb-2">
        <div>
            <div class="flex items-center gap-2">
                <Card.Title class="text-base">{provider.name}</Card.Title>
                {#if !provider.is_configured}
                    <span
                        class="bg-yellow-500/10 text-yellow-500 border border-yellow-500/20 ml-5 rounded-md px-2 py-0.5 text-xs font-medium"
                    >
                        Not configured
                    </span>
                {/if}
            </div>
            <Card.Description class="font-mono text-xs">{provider.id}</Card.Description>
        </div>
        {#if !editing}
            <Button variant="ghost" size="icon" onclick={startEdit}>
                <Pencil class="h-4 w-4" />
            </Button>
        {/if}
    </Card.Header>
    <Card.Content class="space-y-4">
        <div class="space-y-1">
            <Label class="text-muted-foreground text-xs">Destination URL</Label>
            {#if editing}
                <Input bind:value={draft.destination_url} placeholder="https://..." />
            {:else}
                <p class="font-mono text-sm truncate">{provider.destination_url}</p>
            {/if}
        </div>
        <div class="space-y-1">
            <Label class="text-muted-foreground text-xs">Signing Secret</Label>
            {#if editing}
                <div class="relative">
                    <Input
                        type={showSecret ? 'text' : 'password'}
                        bind:value={draft.signing_secret}
                        class="pr-10"
                    />
                    <button
                        type="button"
                        class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                        onclick={() => (showSecret = !showSecret)}
                    >
                        {#if showSecret}<EyeOff class="h-4 w-4" />{:else}<Eye
                                class="h-4 w-4"
                            />{/if}
                    </button>
                </div>
            {:else}
                <p class="font-mono text-sm tracking-widest">••••••••••••</p>
            {/if}
        </div>
    </Card.Content>
    {#if editing}
        <Card.Footer class="flex justify-end gap-2">
            <Button variant="ghost" size="sm" onclick={cancel}>
                <X class="mr-1 h-4 w-4" /> Cancel
            </Button>
            <Button size="sm" onclick={save}>
                <Check class="mr-1 h-4 w-4" /> Save
            </Button>
        </Card.Footer>
    {/if}
</Card.Root>
