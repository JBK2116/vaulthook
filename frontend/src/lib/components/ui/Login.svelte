<script lang="ts">
    import { Button } from '$lib/components/ui/button/index.js';
    import { Label } from '$lib/components/ui/label/index.js';
    import { Input } from '$lib/components/ui/input/index.js';
    import * as Card from '$lib/components/ui/card/index.js';

    // STATE
    let error: String = $state('');
    let isLoginAttemptLoading: boolean = $state(false);

    let { onForgotPassword }: { onForgotPassword: () => void } = $props();

    const onSubmit = async (e: { preventDefault: () => void }) => {
        isLoginAttemptLoading = true;
        e.preventDefault();
        try {
            // TODO:  Replace this and implement it to work with the go backend
            await new Promise((res) => setTimeout(res, 1000));
            const success = false;

            if (!success) {
                throw new Error('Invalid email or password');
            }
        } catch (err: any) {
            error = err.message;
        } finally {
            isLoginAttemptLoading = false;
        }
    };
</script>

<div class="flex flex-col gap-6">
    <Card.Root class="shadow-lg">
        <Card.Header class="">
            <Card.Title class="">Login To Your Account</Card.Title>
            <Card.Description class=""
                >Enter your credentials to begin using HooksVault</Card.Description
            >
        </Card.Header>
        <Card.Content class="">
            <form onsubmit={onSubmit} id="submit-form">
                <div class="flex flex-col gap-6">
                    <div class="grid gap-2">
                        <Label for="email" class="">Email</Label>
                        <Input
                            id="email"
                            type="email"
                            placeholder="john@example.com"
                            required
                            class=""
                        />
                    </div>
                    <div class="grid gap-2">
                        <div class="flex items-center">
                            <Label for="password" class="">Password</Label>
                            <a
                                href="##"
                                onclick={onForgotPassword}
                                class="ms-auto inline-block text-sm underline-offset-4 hover:underline"
                            >
                                Forgot your password?
                            </a>
                        </div>
                        <Input id="password" type="password" required class="" />
                    </div>
                </div>
            </form>
        </Card.Content>
        <Card.Footer class="flex-col gap-2">
            {#if error}
                <p class="w-full text-red-500" id="error-text">{error}</p>
            {/if}
            <Button type="submit" class="w-full" form="submit-form" disabled={isLoginAttemptLoading}
                >{#if isLoginAttemptLoading}
                    Logging In
                {:else}
                    Login
                {/if}</Button
            >
        </Card.Footer>
    </Card.Root>
</div>
