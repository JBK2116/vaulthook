<script lang="ts">
    import { Toaster } from '$lib/components/ui/sonner';

    import './layout.css';

    const { children } = $props();
</script>

<Toaster position="bottom-right" expand={true} closeButton />
{@render children()}
