// Mode: 429

import http from 'k6/http';
import crypto from 'k6/crypto';
import { check } from 'k6';

const secret =
    'whsec_5b5db0374a5cf98206d891a77ee2595be04c01a3d7710a9d5c726f52a3887c2f';

export const options = {
    scenarios: {
        stripe_429_backpressure: {
            executor: 'constant-arrival-rate',
            rate: 50,
            timeUnit: '1s',
            duration: '10m',
            preAllocatedVUs: 100,
            maxVUs: 300,
        },
    },
};

export default function () {
    const timestamp = Math.floor(Date.now() / 1000);
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

    const signature = crypto.hmac(
        'sha256',
        secret,
        `${timestamp}.${payload}`,
        'hex',
    );

    const res = http.post(
        'http://localhost:8080/api/webhooks/stripe',
        payload,
        {
            timeout: '5s',
            headers: {
                'Content-Type': 'application/json',
                'Stripe-Signature': `t=${timestamp},v1=${signature}`,
            },
        },
    );

    check(res, { 'status is 429': (r) => r.status === 429 });
}
