<script lang="ts">
    import { getConnColor } from '$lib/utils/functions';
    import type { ConnState } from '$lib/utils/types';

    interface Props {
        connState: ConnState;
    }
    let { connState }: Props = $props();
</script>

<div class="flex items-center justify-center gap-2">
    <div
        class={`mr-2 sm:mr-5 h-3 w-3 sm:h-5 sm:w-5 animate-pulse rounded-full ${getConnColor(connState)}`}
    ></div>
</div>
