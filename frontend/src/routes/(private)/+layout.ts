// import { goto } from '$app/navigation';
// import { error } from '@sveltejs/kit';
//
// import type { LayoutLoad } from '../$types.js';
//
// export const load: LayoutLoad = async ({ fetch }) => {
//     try {
//         const response = await fetch('/api/me', { method: 'GET', credentials: 'include' });
//         if (!response.ok) {
//             goto('/login');
//         }
//     } catch (err: any) {
//         error(500);
//     }
// };
