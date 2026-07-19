<script lang="ts">
    import { ConnState } from '$lib/utils/types';

    interface Props {
        connState: ConnState;
    }
    let { connState }: Props = $props();

    const states: Record<ConnState, { label: string; bg: string; text: string }> = {
        [ConnState.Connected]: {
            label: 'Live',
            bg: 'bg-conn-live',
            text: 'text-conn-live-foreground',
        },
        [ConnState.Connecting]: {
            label: 'Connecting...',
            bg: 'bg-conn-connecting',
            text: 'text-conn-connecting-foreground',
        },
        [ConnState.Disconnected]: {
            label: 'Disconnected',
            bg: 'bg-conn-disconnected',
            text: 'text-conn-disconnected-foreground',
        },
        [ConnState.Unauthenticated]: {
            label: 'Auth Error',
            bg: 'bg-conn-error',
            text: 'text-conn-error-foreground',
        },
    };

    const current = $derived(states[connState]);
</script>

<div class="flex items-center gap-2">
    <div class="relative flex items-center justify-center">
        <div
            class="h-2 w-2 rounded-full {current.bg} {connState === ConnState.Connecting
                ? 'animate-ping'
                : ''}"
        ></div>
        <div class="absolute h-2 w-2 rounded-full {current.bg}"></div>
    </div>
    <span class="font-medium {current.text}">{current.label}</span>
</div>
