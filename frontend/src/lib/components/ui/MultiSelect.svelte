<script lang="ts">
    import * as Checkbox from '$lib/components/ui/checkbox/index.js';
    import * as Popover from '$lib/components/ui/popover/index.js';
    import { cn } from '$lib/utils.js';
    import { ChevronDown } from '@lucide/svelte';

    interface Props {
        options: string[];
        selected: string[];
        placeholder?: string;
        class?: string;
    }

    let {
        options,
        selected = $bindable(),
        placeholder = 'Select...',
        class: className,
    }: Props = $props();

    let open = $state(false);

    function toggle(option: string) {
        if (selected.includes(option)) {
            selected = selected.filter((s) => s !== option);
        } else {
            selected = [...selected, option];
        }
    }
</script>

<Popover.Root bind:open>
    <Popover.Trigger
        class={cn(
            'border-input dark:bg-input/30 dark:hover:bg-input/50 flex w-full items-center justify-between gap-1.5 rounded-none border bg-transparent px-2.5 py-2 text-xs whitespace-nowrap transition-colors outline-none select-none h-8',
            open && 'ring-ring ring-1',
            className,
        )}
    >
        <span class={cn(selected.length === 0 && 'text-muted-foreground')}>
            {selected.length > 0 ? `${selected.length} selected` : placeholder}
        </span>
        <ChevronDown
            class={cn(
                'text-muted-foreground size-4 shrink-0 transition-transform',
                open && 'rotate-180',
            )}
        />
    </Popover.Trigger>
    <Popover.Content class="w-48 p-0" align="start" sideOffset={4}>
        <div class="max-h-48 overflow-auto p-1">
            {#each options as option}
                <label
                    class="hover:bg-muted flex cursor-pointer items-center gap-2 rounded-none px-2 py-1.5 text-xs"
                >
                    <Checkbox.Root
                        checked={selected.includes(option)}
                        onCheckedChange={() => toggle(option)}
                    />
                    <span>{option}</span>
                </label>
            {/each}
            {#if options.length === 0}
                <div class="text-muted-foreground px-2 py-2 text-xs">No options</div>
            {/if}
        </div>
    </Popover.Content>
</Popover.Root>
