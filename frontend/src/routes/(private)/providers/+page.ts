import type { Provider } from '$lib/utils/types';
import { error, redirect } from '@sveltejs/kit';

import type { PageLoad } from './$types';

// Load all providers from the database. Error 500 if unsuccessful.
export const load: PageLoad = async ({ fetch }) => {
    try {
        const res = await fetch('/api/providers', { method: 'GET', credentials: 'include' });
        if (!res.ok) {
            // attempt to refresh the access token
            if (res.status === 401) {
                const resTwo = await fetch('/api/refresh', {
                    method: 'POST',
                    credentials: 'include',
                });
                if (!resTwo.ok) {
                    redirect(301, '/login');
                }
                // try again
                const resThree = await fetch('/api/providers', {
                    method: 'GET',
                    credentials: 'include',
                });
                if (!resThree.ok) {
                    error(500, 'Internal Server Error');
                }

                const providers: Provider[] = await resThree.json();
                providers.sort((a, b) => a.name.localeCompare(b.name));
                return { providers };
            } else {
                error(500, 'Internal Server Error');
            }
        }
        const providers: Provider[] = await res.json();
        providers.sort((a, b) => a.name.localeCompare(b.name));
        return { providers };
    } catch (err: any) {
        error(500, 'Failed to load providers');
    }
};
