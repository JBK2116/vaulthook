<script lang="ts">
    import { goto } from '$app/navigation';
    import { page } from '$app/state';
    import type { ConnState } from '$lib/utils/types';

    import Button from './button/button.svelte';
    import ConnIndicator from './ConnIndicator.svelte';

    interface Props {
        connState?: ConnState;
    }
    let { connState }: Props = $props();

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
    <a href="/" class="flex items-center gap-2 font-semibold tracking-tight">
        <span class="text-primary">🟈</span>
        VaultHook
    </a>
{#if !isLogin}
        <div class="flex items-center gap-3">
            {#if connState}
                <ConnIndicator {connState} />
            {/if}
            <Button
                variant="link"
                type="button"
                class="text-sm cursor-pointer"
                size="lg"
                onclick={gotoProviders}
            >
                Providers
            </Button>
            <Button
                variant="link"
                type="button"
                class="text-sm cursor-pointer"
                size="lg"
                onclick={logout}
            >
                Logout
            </Button>
        </div>
    {/if}
</nav>
