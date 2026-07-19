/**
 * Attempts to refresh the access token via the refresh token cookie.
 * Returns true if successful, false otherwise.
 */
export async function reAuthenticate(): Promise<boolean> {
    try {
        const res = await fetch('/api/refresh', { method: 'POST', credentials: 'include' });
        return res.ok;
    } catch {
        return false;
    }
}
