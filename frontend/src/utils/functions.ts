import { DeliveryStatusTypes, type WebHookEvent } from './types.ts';

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
