/** A webhook event received from the backend */
export interface WebHookEvent {
    id: string; // id of the event
    provider_id: string; // id of the provider attached to this event
    provider: string; // provider of the event
    event_id: string | null; // ID of the event, send by the provider
    event_type: string; // type of the event
    headers: Record<string, string[]>; // headers sent in event (http.Header)
    payload: Record<string, unknown>; // payload sent in event
    delivery_status: DeliveryStatusTypes; // delivery status of the event
    forwarded_to: string; // forwared to server address of the event
    response_code: number | null; // response code of the event
    retry_count: number; // amount of retries for the event
    next_retry_at: string | null; // next retry scheduled at time
    last_error: string | null; // last error that occurred with this event
    received_at: string; // received time of the event (ISO861 Format)
    updated_at: string; // updated at time of the event (ISO861 format)
    created_at: string; // created at time of the event (ISO861 Format)
}

/** A provider received from the backend */
export interface Provider {
    id: string; // id of the provider
    name: string; // name of the provider
    signing_secret: string; // signing_secret of the provider to validate webhooks
    destination_url: string; // destination address to forward webhooks
    is_configured: boolean; // boolean indicating if the user has configured this provider fully
    created_at: string; // time indicating when the provider was created at (ISO861 Format)
}

/** A stats object received from the backend */
export interface Stats {
    delivered: number;
    failed: number;
    retrying: number;
    queued: number;
}

/** A search payload sent to the backend */
export interface SearchPayload {
    // quick lookup options
    webhook_id: string | null;
    event_id: string | null;
    // filter options
    providers: string[];
    event_type: string | null;
    delivery_statuses: string[];
    response_code: number | null;
    from_time: string | null;
    to_time: string | null;
    payload_search: string | null;
    has_retries: boolean;
    has_error: boolean;
}

/** Available types of delivery statuses of a webhook */
export enum DeliveryStatusTypes {
    Queued = 'queued',
    Processing = 'processing',
    Delivered = 'delivered',
    Retrying = 'retrying',
    Failed = 'failed',
}

/** Colors pertaining to each `DeliveryStatusTypes` */
export enum DeliveryStatusColors {
    delivered = 'text-status-delivered-foreground',
    failed = 'text-status-failed-foreground',
    retrying = 'text-status-retrying-foreground',
    queued = 'text-status-queued-foreground',
}

/** Filter types for delivery status on SSE page */
export enum SelectTypes {
    All = 'all',
    Delivered = 'delivered',
    Queued = 'queued',
    Retrying = 'retrying',
    Failed = 'failed',
}

/** Provider names available in application */
export enum ProviderTypes {
    Stripe = 'Stripe',
    Github = 'Github',
}

/** SSE connection states */
export enum ConnState {
    Connected = 'connected',
    Connecting = 'connecting',
    Disconnected = 'disconnected',
    Unauthenticated = 'unauthenticated',
}
