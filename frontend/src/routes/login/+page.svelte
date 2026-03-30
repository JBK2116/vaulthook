<script lang="ts">
    import ForgotPasswordCard from '$lib/components/ui/ForgotPasswordCard.svelte';
    import Login from '$lib/components/ui/Login.svelte';
    import Navbar from '$lib/components/ui/Navbar.svelte';

    /** @type {boolean} */
    let showModal: boolean = $state(false);

    /** Closes the forgot password modal */
    const onCloseForgotPassword: () => void = () => {
        showModal = false;
    };

    /** Opens the forgot password modal */
    const onOpenForgotPassword: () => void = () => {
        showModal = true;
    };
</script>

<svelte:head>
    <title>Login - VaultHook</title>
    <meta name="Login" content="Login to HooksVault to begin monitoring your webhooks" />
</svelte:head>
<Navbar></Navbar>
<main class="flex min-h-[calc(100svh-64px)] items-center justify-center px-4 py-10">
    <div class="w-full max-w-sm">
        <Login onForgotPassword={onOpenForgotPassword}></Login>
    </div>
    <ForgotPasswordCard {showModal} onClose={onCloseForgotPassword}></ForgotPasswordCard>
</main>
