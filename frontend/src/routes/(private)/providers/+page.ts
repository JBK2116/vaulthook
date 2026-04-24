import type { Provider } from '$lib/utils/types';
import { error } from '@sveltejs/kit';

import type { PageLoad } from './$types';

// Load all providers from the database. Error 500 if unsuccessful.
export const load: PageLoad = async ({ fetch }) => {
    try {
        const url = '/api/providers';
        const res = await fetch(url, { method: 'GET', credentials: 'include' });
        if (!res.ok) {
            error(500, 'Internal Server Error');
        }
        const providers: Provider[] = await res.json();
        providers.sort((a, b) => a.name.localeCompare(b.name));
        return { providers };
    } catch (err: any) {
        error(500, 'Failed to load providers');
    }
};
