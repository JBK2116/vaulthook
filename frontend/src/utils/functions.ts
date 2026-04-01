import {
    DeliveryStatusColors,
    DeliveryStatusTypes,
    ProviderTypes,
    SelectTypes,
    type WebHookEvent,
} from './types.ts';

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
        return data.filter((e) => e.event_type.includes(currentSearchString));
    }
    return data.filter(
        (e) =>
            e.delivery_status === currentSelectedOption &&
            e.event_type.includes(currentSearchString),
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
            return '#1a1040'; // deep stripe purple
        case ProviderTypes.Github:
            return '#161b22'; // github dark
        case ProviderTypes.Sns:
            return '#1a2a0a'; // aws green dark
        default:
            return '#1a1a1a';
    }
}

/**
 * Returns the css text color associated with the passed in provider
 */
export function getProviderTextColor(provider: string): string {
    switch (provider) {
        case ProviderTypes.Stripe:
            return '#7c6af7'; // stripe purple
        case ProviderTypes.Github:
            return '#e6edf3'; // github text white
        case ProviderTypes.Sns:
            return '#4caf50'; // aws green
        default:
            return '#a3a3a3';
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
 * Returns a formatted string in HH:MM:SS format using the provided received_at time
 */
export function formatReceivedAtTime(received_at: string): string {
    let date = new Date(received_at);
    return `${date.getHours()}:${date.getMinutes()}:${date.getSeconds()}`;
}
