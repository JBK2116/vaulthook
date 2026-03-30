<script lang="ts">
    import { page } from '$app/state';
    import { goto } from '$app/navigation';
    import Button from './button/button.svelte';

    const pathname = $derived(String(page.url.pathname));
    const isDashboard = $derived(pathname === '/');

    function handleLogout() {
        // TODO: Call this function when the backend is ready.
        localStorage.removeItem('token');
        goto('/login');
    }

    function sayHi() {
        console.log('hi');
    }
</script>

<nav
    class="border-border/60 bg-background/80 sticky top-0 z-50 flex h-16 items-center justify-between border-b px-6 backdrop-blur-sm"
>
    <a href="/" class="flex items-center gap-2 font-semibold tracking-tight">
        <span class="text-primary">🟈</span>
        VaultHook
    </a>
    {#if isDashboard}
        <Button
            variant="link"
            type="button"
            class="text-sm"
            size="lg"
            aria-label="Submit"
            disabled={false}
            onclick={sayHi}>Logout</Button
        >
    {/if}
</nav>
