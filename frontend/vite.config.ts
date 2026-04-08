import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
    plugins: [tailwindcss(), sveltekit()],
    server: {
        proxy: {
            '/api': {
                target: 'http://localhost:8080', // TODO: Update this line to point to the correct port in the golang backend
                changeOrigin: true,
            },
        },
    },
});
