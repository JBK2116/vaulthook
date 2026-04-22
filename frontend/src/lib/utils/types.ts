export interface WebHookEvent {
    id: string; // id of the event
    provider: string; // provider of the event
    event_type: string; // type of the event
    received_at: string; // received time of the event (ISO861 Format)
    response_code: number | null; // response code of the event
    retry_count: number; // amount of retries for the event
    forwarded_to: string; // forwared to server address of the event
    last_error: string | null; // last error that occurred with this event
    headers: Record<string, string>; // headers sent in event
    payload: Record<string, unknown>; // payload sent in event
    delivery_status: DeliveryStatusTypes; // delivery status of the event
}

export enum DeliveryStatusTypes {
    Delivered = 'delivered',
    Queued = 'queued',
    Retrying = 'retrying',
    Failed = 'failed',
}

export enum DeliveryStatusColors {
    delivered = 'text-emerald-400',
    failed = 'text-red-400',
    retrying = 'text-amber-400',
    queued = 'text-slate-400',
}

export enum SelectTypes {
    All = 'all',
    Delivered = 'delivered',
    Queued = 'queued',
    Retrying = 'retrying',
    Failed = 'failed',
}

export enum ProviderTypes {
    Stripe = 'stripe',
    Github = 'github',
    Sns = 'sns',
}
