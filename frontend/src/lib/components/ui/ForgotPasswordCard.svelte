<script lang="ts">
    import * as Card from '$lib/components/ui/card/index.js';
    let { showModal, onClose }: { showModal: boolean; onClose: () => void } = $props();
</script>

{#if showModal}
    <div
        role="presentation"
        onkeydown={onClose}
        onclick={onClose}
        class="fixed inset-0 z-50 flex h-full w-full items-start justify-center overflow-auto bg-black/40 pt-24"
    >
        <div
            role="presentation"
            class="modal-content w-full max-w-sm"
            onclick={(e) => e.stopPropagation()}
            onkeydown={(e) => e.stopPropagation()}
        >
            <Card.Root class="w-full">
                <Card.Header class="">
                    <Card.Title class="">Forgot Password</Card.Title>
                    <Card.Description class="">Check your .env file</Card.Description>
                </Card.Header>
                <Card.Content class="">
                    <p>
                        Ensure that ADMIN_PASSWORD and ADMIN_EMAIL is set in the .env file. If it is
                        already set, update it to change your credentials for the application.
                    </p>
                    <button onclick={onClose} class="mt-4 w-full rounded bg-black py-2 text-white">
                        Close
                    </button>
                </Card.Content>
            </Card.Root>
        </div>
    </div>
{/if}

<style>
    .modal-content {
        animation: animatetop 0.4s ease;
    }
    @keyframes animatetop {
        from {
            top: -300px;
            opacity: 0;
        }
        to {
            top: 0;
            opacity: 1;
        }
    }
</style>
