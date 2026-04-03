<script lang="ts">
    import * as Sheet from '$lib/components/ui/sheet/index.js';

    import type { WebHookEvent } from '../../../utils/types.ts';
    import Sidebar from './Sidebar.svelte';

    interface Props {
        currentSelectedEvent: WebHookEvent | null;
        open: boolean;
    }
    let { currentSelectedEvent, open = $bindable() }: Props = $props();
</script>

<Sheet.Root bind:open>
    <Sheet.Content side="bottom" class="h-[85vh] p-0">
        <Sidebar {currentSelectedEvent} />
    </Sheet.Content>
</Sheet.Root>
