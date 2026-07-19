<script lang="ts">
    interface Props {
        label: string;
        valueNumber: number;
        valueNumberColor: string;
    }
    let { label, valueNumber, valueNumberColor }: Props = $props();
</script>

<div class="border-border flex flex-col gap-1 border-r px-2 py-1.5 last:border-r-0 sm:px-5 sm:py-3">
    <span class="text-muted-foreground text-[8px] sm:text-[10px] tracking-widest uppercase">
        {label}
    </span>
    <span class={`text-base sm:text-xl font-medium tabular-nums ${valueNumberColor}`}>
        {valueNumber.toLocaleString()}
    </span>
</div>
