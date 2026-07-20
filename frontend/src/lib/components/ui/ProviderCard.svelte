<script lang="ts">
    import { Button } from '$lib/components/ui/button';
    import * as Card from '$lib/components/ui/card';
    import { Input } from '$lib/components/ui/input';
    import { Label } from '$lib/components/ui/label';
    import type { Provider } from '$lib/utils/types';
    import { Check, Eye, EyeOff, Pencil, X } from '@lucide/svelte';
    import { toast } from 'svelte-sonner';

    interface Props {
        data: Provider;
    }
    let { data }: Props = $props();
    // svelte-ignore state_referenced_locally
    let provider: Provider = $state(data);
    // card state management
    let editing = $state(false);
    let showSecret = $state(false);
    let draft = $state({ signing_secret: '', destination_url: '' });
    let savingData = $state(false);
    // card accent when not configured
    let cardClass = $derived(
        `w-full${!provider.is_configured ? ' border-l-2 border-l-yellow-500/50 bg-yellow-500/[0.02]' : ''}`,
    );
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
    async function save() {
        if (savingData) {
            return;
        }
        savingData = true;
        const body = draft;
        // minor body validation
        if (!provider.is_configured) {
            // both fields must change
            if (body.destination_url.length <= 0) {
                toast.error('Destination URL is required');
                savingData = false;
                return;
            }
            if (body.signing_secret.length <= 0) {
                toast.error('Signing Secret is required');
                savingData = false;
                return;
            }
        } else {
            // at least one field must change
            if (body.destination_url.length <= 0 || body.signing_secret.length <= 0) {
                toast.error('At least one field required');
                savingData = false;
                return;
            }
        }
        // prevent empty changes
        if (
            body.signing_secret === provider.signing_secret &&
            body.destination_url === provider.destination_url
        ) {
            toast.warning('No changes detected');
            savingData = false;
            return;
        }
        // update the provider in the database
        const url = `/api/providers/${provider.id}`;
        const res = await fetch(url, {
            method: 'PATCH',
            body: JSON.stringify(body),
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
        });
        if (!res.ok) {
            toast.error('Unable to update provider');
            savingData = false;
            return;
        }
        const updatedProvider: Provider = await res.json();
        provider = updatedProvider;
        editing = false;
        savingData = false;
        toast.success('Provider updated');
    }
</script>

<Card.Root class={cardClass}>
    <Card.Header class="flex flex-row items-start justify-between space-y-0 pb-2">
        <div>
            <div class="flex items-center">
                <Card.Title class="text-base w-18 truncate">{provider.name}</Card.Title>
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
                <Input
                    bind:value={draft.destination_url}
                    placeholder={`https://your-domain.com/webhooks/${provider.name.toLowerCase()}`}
                />
            {:else}
                <p class="font-mono text-sm truncate">
                    {#if !provider.destination_url}
                        {`https://your-domain.com/webhooks/${provider.name.toLowerCase()}`}
                    {:else}
                        {provider.destination_url}
                    {/if}
                </p>
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
            <Button size="sm" onclick={save} disabled={savingData}>
                <Check class="mr-1 h-4 w-4" /> Save
            </Button>
        </Card.Footer>
    {/if}
</Card.Root>
