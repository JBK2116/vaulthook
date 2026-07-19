import { redirect } from '@sveltejs/kit';

import type { LayoutLoad } from '../$types.js';

export const load: LayoutLoad = async ({ fetch }) => {
    const res = await fetch('/api/me', { method: 'GET', credentials: 'include' });
    if (res.ok) {
        return;
    }

    const refresh = await fetch('/api/refresh', { method: 'POST', credentials: 'include' });
    if (!refresh.ok) {
        redirect(302, '/login');
    }
};
