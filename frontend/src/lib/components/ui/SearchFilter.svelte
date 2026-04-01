<script lang="ts">
    import { Input } from './input/index.ts';

    interface Props {
        currentSearchString: string;
    }
    let { currentSearchString = $bindable() }: Props = $props();
</script>

<Input
    type="search"
    placeholder="Search by event..."
    class="max-w-3xs rounded-sm"
    bind:value={currentSearchString}
></Input>
