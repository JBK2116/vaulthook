<script lang="ts">
    import { goto } from '$app/navigation';
    import { page } from '$app/state';
    import { Search } from '@lucide/svelte';

    import Button from './button/button.svelte';

    const pathname: String = $derived(String(page.url.pathname));
    const isLogin: boolean = $derived(pathname === '/login');

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

    async function gotoProviders(): Promise<void> {
        goto('/providers');
    }
</script>

<nav
    class="border-border/60 bg-background/80 sticky top-0 z-50 flex h-16 items-center justify-between border-b px-6 backdrop-blur-sm"
>
    <!-- Left: Brand -->
    <a href="/" class="flex shrink-0 items-center gap-2 font-semibold tracking-tight">
        <span class="text-primary">🟈</span>
        VaultHook
    </a>

    <!-- Center: Search -->
    {#if !isLogin}
        <div class="flex flex-1 items-center justify-center">
            <button
                onclick={() => goto('/search')}
                class="border-border dark:bg-input/30 dark:hover:bg-input/50 inline-flex items-center gap-2 rounded-sm border px-3 py-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
            >
                <Search class="size-3.5" />
                <span class="hidden sm:inline">Search</span>
            </button>
        </div>
    {/if}

    <!-- Right: Navigation buttons -->
    {#if !isLogin}
        <div class="flex shrink-0 items-center gap-2">
            <Button
                variant="link"
                type="button"
                class="text-sm cursor-pointer text-white"
                size="lg"
                onclick={gotoProviders}
            >
                Providers
            </Button>
            <Button
                variant="link"
                type="button"
                class="text-sm cursor-pointer text-white"
                size="lg"
                onclick={logout}
            >
                Logout
            </Button>
        </div>
    {/if}
</nav>
