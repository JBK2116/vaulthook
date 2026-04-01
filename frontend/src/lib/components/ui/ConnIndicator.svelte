<script lang="ts">
    interface Props {
        isConnectedToBackend: boolean;
    }
    let { isConnectedToBackend }: Props = $props();
</script>

<div class="flex items-center justify-center gap-2">
    <div
        class={`mr-2 sm:mr-5 h-3 w-3 sm:h-5 sm:w-5 animate-pulse rounded-full ${isConnectedToBackend ? 'bg-[#66FF99]' : 'bg-red-500'}`}
    ></div>
</div>
