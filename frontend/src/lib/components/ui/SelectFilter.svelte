<script lang="ts">
    import * as Select from '$lib/components/ui/select/index.js';

    import * as functions from '../../../utils/functions.ts';
    import { SelectTypes } from '../../../utils/types.ts';

    interface Props {
        currentSelectedOption: string;
    }
    let { currentSelectedOption = $bindable() }: Props = $props();
    const updateSelectedOptionDisplay: () => void = () => {
        return functions.capitalize(currentSelectedOption);
    };
</script>

<Select.Root type="single" name="deliveryFilterOptions" bind:value={currentSelectedOption}>
    <Select.Trigger class="w-45">{updateSelectedOptionDisplay()}</Select.Trigger>
    <Select.Content class="">
        <Select.Group class="">
            <Select.Label class="">Filter</Select.Label>

            {#each Object.values(SelectTypes) as selectType (selectType)}
                <Select.Item class="" value={selectType} label={selectType}>
                    {functions.capitalize(selectType)}
                </Select.Item>
            {/each}
        </Select.Group>
    </Select.Content>
</Select.Root>
