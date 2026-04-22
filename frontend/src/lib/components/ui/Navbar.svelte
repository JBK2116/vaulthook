<script lang="ts">
    import { goto } from '$app/navigation';
    import { page } from '$app/state';

    import Button from './button/button.svelte';

    const pathname: String = $derived(String(page.url.pathname));
    const isDashboard: boolean = $derived(pathname === '/');

    async function logout(): Promise<void> {
        try {
            const res = await fetch('/api/logout', { method: 'POST', credentials: 'include' });
            if (!res.ok) {
                throw new Error('Error occurred logging out...');
            }
            goto('/login');
        } catch (err: any) {
            goto('/login');
        }
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
            onclick={logout}>Logout</Button
        >
    {/if}
</nav>
