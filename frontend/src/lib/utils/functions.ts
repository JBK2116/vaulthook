import { type CalendarDate, getLocalTimeZone } from '@internationalized/date';

import {
    DeliveryStatusColors,
    DeliveryStatusTypes,
    ProviderTypes,
    SelectTypes,
    type WebHookEvent,
} from './types';

/**
 * Returns an array containing all events that match the provided option filter and search string
 */
export function getDisplayedEvents(
    currentSelectedOption: string,
    currentSearchString: string,
    data: WebHookEvent[],
): WebHookEvent[] {
    if (
        (currentSelectedOption === SelectTypes.All && currentSearchString === '') ||
        data.length <= 0
    ) {
        return data;
    }
    if (currentSelectedOption === SelectTypes.All) {
        return data.filter((e) =>
            e.event_type.toLowerCase().includes(currentSearchString.toLowerCase()),
        );
    }
    return data.filter(
        (e) =>
            e.delivery_status === currentSelectedOption &&
            e.event_type.toLowerCase().includes(currentSearchString.toLowerCase()),
    );
}

/**
 * Returns the total number of webhook events stored in the frontend
 */
export function getTotalEvents(events: WebHookEvent[]): number {
    return events.length;
}

/**
 * Returns the total number of Delivered events stored in the frontend
 */
export function getTotalDeliveredEvents(events: WebHookEvent[]): number {
    return events.filter((e) => e.delivery_status === DeliveryStatusTypes.Delivered).length;
}

/**
 * Returns the total number of Retrying events stored in the frontend
 */
export function getTotalRetryingEvents(events: WebHookEvent[]): number {
    return events.filter((e) => e.delivery_status === DeliveryStatusTypes.Retrying).length;
}

/**
 * Returns the total number of Queued events stored in the frontend
 */
export function getTotalQueuedEvents(events: WebHookEvent[]): number {
    return events.filter((e) => e.delivery_status === DeliveryStatusTypes.Queued).length;
}

/**
 * Returns the total number of Failed events stored in the frontend
 */
export function getTotalFailedEvents(events: WebHookEvent[]): number {
    return events.filter((e) => e.delivery_status === DeliveryStatusTypes.Failed).length;
}

/**
 * Capitalizes the first letter of the provided string
 */
export function capitalize(str: string): string {
    return str ? str.charAt(0).toUpperCase() + str.slice(1) : '';
}

/**
 * Returns the css background color associated with the passed in provider
 */
export function getProviderBackgroundColor(provider: string): string {
    switch (provider) {
        case ProviderTypes.Stripe:
            return 'var(--provider-stripe)';
        case ProviderTypes.Github:
            return 'var(--provider-github)';
        default:
            return 'var(--provider-default)';
    }
}

/**
 * Returns the css text color associated with the passed in provider
 */
export function getProviderTextColor(provider: string): string {
    switch (provider) {
        case ProviderTypes.Stripe:
            return 'var(--provider-stripe-foreground)';
        case ProviderTypes.Github:
            return 'var(--provider-github-foreground)';
        default:
            return 'var(--provider-default-foreground)';
    }
}

/**
 * Returns the css text color associated with the provided delivery status string
 */
export function getDeliveryStatusTextColor(delivery_status: string) {
    switch (delivery_status) {
        case DeliveryStatusTypes.Delivered:
            return DeliveryStatusColors.delivered;
        case DeliveryStatusTypes.Failed:
            return DeliveryStatusColors.failed;
        case DeliveryStatusTypes.Queued:
            return DeliveryStatusColors.queued;
        case DeliveryStatusTypes.Retrying:
            return DeliveryStatusColors.retrying;
        default:
            return '';
    }
}

/**
 * Returns the css color associated with the provided response code
 *
 * If the provided response_code is null, then it returns a text color variant (ie: text-muted-foreground)
 */
export function getResponseCodeColor(response_code: number | null): string {
    if (typeof response_code === 'number') {
        if (response_code >= 200 && response_code < 300) {
            return DeliveryStatusColors.delivered;
        } else {
            return DeliveryStatusColors.failed;
        }
    } else {
        return 'text-muted-foreground';
    }
}

/**
 * Parse an ISO 8601 / RFC 3339 timestamp string ensuring UTC interpretation.
 * If the string lacks a timezone designator (Z or ±HH:MM), 'Z' is appended
 * so that `new Date()` treats it as UTC rather than local time.
 */
function parseUTCDate(dateStr: string): Date {
    // RFC 3339 timezone offset always starts with Z, +, or - in the last 6 chars.
    // ISO 8601 without designator is ambiguous; force UTC.
    if (!/[+\-Zz]/.test(dateStr.slice(-6))) {
        dateStr += 'Z';
    }
    return new Date(dateStr);
}

/**
 * Returns a formatted time string in the user's local timezone (24h format).
 * Used in the event table column.
 */
export function formatReceivedAtTimeForTable(received_at: string, timezone: string): string {
    const date = parseUTCDate(received_at);
    if (isNaN(date.getTime())) return received_at;

    return new Intl.DateTimeFormat(navigator.language, {
        timeZone: timezone,
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
    }).format(date);
}

/**
 * Returns a full date/time string in the user's local timezone.
 * Used in the event detail sidebar.
 */
export function formatReceivedAtTimeForSidebar(received_at: string, timezone: string): string {
    const date = parseUTCDate(received_at);
    if (isNaN(date.getTime())) return received_at;

    return new Intl.DateTimeFormat(navigator.language, {
        timeZone: timezone,
        day: '2-digit',
        month: 'long',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
    }).format(date);
}

/**
 * Formats a CalendarDate and time string into an ISO 8601 date-time string.
 * Combines the date part from the CalendarDate and the hours/minutes from the time string.
 * Returns null if no date is provided.
 *
 * @param {CalendarDate | undefined} date - The calendar date to format.
 * @param {string} time - A string in the format "HH:MM" representing the time.
 * @returns {string | null} The combined date-time as an ISO string, or null if date is undefined.
 */
export function formatDateTime(date: CalendarDate | undefined, time: string): string | null {
    if (!date) {
        return null;
    }
    const d = date.toDate(getLocalTimeZone());
    const [h, m, s = 0] = time.split(':').map(Number);
    d.setHours(h, m, s, 0);
    return d.toISOString();
}

/** Capitalizes the first letter in a string
 *
 * @returns {string} The full string with the first char being capitalized
 * */
export function firstToUpper(str: string): string {
    return str.charAt(0).toUpperCase() + str.slice(1);
}
