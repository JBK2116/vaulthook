import http from 'k6/http';
import crypto from 'k6/crypto';
import { check } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const secret =
    'whsec_5b5db0374a5cf98206d891a77ee2595be04c01a3d7710a9d5c726f52a3887c2f'; // change this accordingly to your dev secret ... No need to hide it as prod secret is different
const errorRate = new Rate('errors');
const webhookDuration = new Trend('webhook_duration');

const payload = JSON.stringify({
    id: 'evt_1NG8Du2eZvKYlo2CUI79vXWy',
    object: 'event',
    api_version: '2026-04-22.dahlia',
    created: 1686089970,
    data: {
        object: {
            id: 'seti_1NG8Du2eZvKYlo2C9XMqbR0x',
            object: 'setup_intent',
            status: 'requires_confirmation',
            payment_method_types: ['acss_debit'],
            livemode: false,
            metadata: {},
        },
    },
    livemode: false,
    pending_webhooks: 0,
    request: { id: null, idempotency_key: null },
    type: 'setup_intent.created',
});

export let options = {
    stages: [
        { duration: '30s', target: 10 }, // ramp up
        { duration: '1m', target: 10 }, // sustained
        { duration: '30s', target: 50 }, // spike
        { duration: '30s', target: 0 }, // ramp down
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests under 500ms
        errors: ['rate<0.01'], // less than 1% errors
    },
};

export default function () {
    const timestamp = Math.floor(Date.now() / 1000);
    const signedPayload = `${timestamp}.${payload}`;
    const signature = crypto.hmac('sha256', secret, signedPayload, 'hex');

    const res = http.post(
        'http://localhost:8080/api/webhooks/stripe',
        payload,
        {
            headers: {
                'Content-Type': 'application/json',
                'Stripe-Signature': `t=${timestamp},v1=${signature}`,
            },
        },
    );

    webhookDuration.add(res.timings.duration);
    errorRate.add(res.status !== 200);

    check(res, {
        'status is 200': (r) => r.status === 200,
        'response time < 500ms': (r) => r.timings.duration < 500,
    });
}
