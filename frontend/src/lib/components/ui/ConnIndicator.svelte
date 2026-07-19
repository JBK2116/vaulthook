<script lang="ts">
    import { ConnState } from '$lib/utils/types';
    import { LoaderCircle, ShieldAlert, Wifi, WifiOff } from '@lucide/svelte';

    interface Props {
        connState: ConnState;
    }
    let { connState }: Props = $props();

    const states: Record<
        ConnState,
        { label: string; bg: string; text: string; icon: typeof Wifi }
    > = {
        [ConnState.Connected]: {
            label: 'Live',
            bg: 'bg-emerald-500',
            text: 'text-emerald-400',
            icon: Wifi,
        },
        [ConnState.Connecting]: {
            label: 'Connecting...',
            bg: 'bg-amber-500',
            text: 'text-amber-400',
            icon: LoaderCircle,
        },
        [ConnState.Disconnected]: {
            label: 'Disconnected',
            bg: 'bg-red-500',
            text: 'text-red-400',
            icon: WifiOff,
        },
        [ConnState.Unauthenticated]: {
            label: 'Auth Error',
            bg: 'bg-orange-500',
            text: 'text-orange-400',
            icon: ShieldAlert,
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
    <span class="text-xs font-medium {current.text}">{current.label}</span>
</div>
