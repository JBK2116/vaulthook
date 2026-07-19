<script lang="ts">
    import { Input } from './input/index';

    interface Props {
        currentSearchString: string;
    }
    let { currentSearchString = $bindable() }: Props = $props();
</script>

<Input
    type="search"
    placeholder="Search by event..."
    class="w-full sm:max-w-3xs rounded-sm"
    bind:value={currentSearchString}
></Input>
