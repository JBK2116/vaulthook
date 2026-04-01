<script lang="ts">
    interface Props {
        label: string;
        valueNumber: number;
        valueNumberColor: string;
    }
    let { label, valueNumber, valueNumberColor }: Props = $props();
</script>

<div class="border-border flex flex-col gap-1 border-r px-5 py-3 last:border-r-0">
    <span class="text-muted-foreground text-[10px] tracking-widest uppercase">
        {label}
    </span>
    <span class={`text-xl font-medium tabular-nums ${valueNumberColor}`}>
        {valueNumber.toLocaleString()}
    </span>
</div>
