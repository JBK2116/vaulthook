// Mode 200

import http from 'k6/http';
import crypto from 'k6/crypto';
import { check } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const secret =
    'whsec_5b5db0374a5cf98206d891a77ee2595be04c01a3d7710a9d5c726f52a3887c2f'; // change this accordingly to your dev secret ... No need to hide it as prod secret is different

const errorRate = new Rate('errors');
const serverErrors = new Rate('server_errors');
const clientErrors = new Rate('client_errors');
const webhookDuration = new Trend('webhook_duration');

export const options = {
    scenarios: {
        stripe_ingestion: {
            executor: 'constant-arrival-rate',
            rate: 25, // 25 requests/sec baseline
            timeUnit: '1s',
            duration: '3m',
            preAllocatedVUs: 50,
            maxVUs: 200,
        },
    },
    thresholds: {
        http_req_duration: ['p(95)<150'], // ingestion should stay very fast
        errors: ['rate<0.01'], // less than 1% total failures
        server_errors: ['rate<0.01'], // backend should almost never 5xx
    },
};

export default function () {
    const timestamp = Math.floor(Date.now() / 1000);
    // unique event IDs prevent unrealistic caching behavior
    // and better simulate real Stripe traffic
    const eventId = `evt_${__VU}_${__ITER}_${Date.now()}`;
    const payload = JSON.stringify({
        id: eventId,
        object: 'event',
        api_version: '2026-04-22.dahlia',
        created: timestamp,
        data: {
            object: {
                id: `seti_${__VU}_${__ITER}`,
                object: 'setup_intent',
                status: 'requires_confirmation',
                payment_method_types: ['acss_debit'],
                livemode: false,
                metadata: { vu: String(__VU), iter: String(__ITER) },
            },
        },
        livemode: false,
        pending_webhooks: 0,
        request: { id: null, idempotency_key: null },
        type: 'setup_intent.created',
    });
    const signedPayload = `${timestamp}.${payload}`;
    const signature = crypto.hmac('sha256', secret, signedPayload, 'hex');
    const res = http.post(
        'http://localhost:8080/api/webhooks/stripe',
        payload,
        {
            timeout: '5s',
            headers: {
                'Content-Type': 'application/json',
                'Stripe-Signature': `t=${timestamp},v1=${signature}`,
            },
            tags: { provider: 'stripe', endpoint: 'ingestion' },
        },
    );

    webhookDuration.add(res.timings.duration);
    errorRate.add(res.status !== 200);
    serverErrors.add(res.status >= 500);
    clientErrors.add(res.status >= 400 && res.status < 500);
    check(res, {
        'status is 200': (r) => r.status === 200,
        'response time < 150ms': (r) => r.timings.duration < 150,
        'response queued': (r) => {
            try {
                return r.json('status') === 'queued';
            } catch (_) {
                return false;
            }
        },
    });
}
