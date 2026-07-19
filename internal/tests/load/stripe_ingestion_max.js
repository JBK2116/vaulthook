import http from 'k6/http';
import crypto from 'k6/crypto';
import { check } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Configuration constants
const secret =
    'whsec_5b5db0374a5cf98206d891a77ee2595be04c01a3d7710a9d5c726f52a3887c2f';
const BASE_URL = 'http://localhost:8080/api/webhooks/stripe';

// Custom metrics for deep analysis
const errorRate = new Rate('errors');
const serverErrors = new Rate('server_errors');
const webhookDuration = new Trend('webhook_duration');
const payloadSize = new Trend('payload_size_bytes');

export const options = {
    scenarios: {
        // Gradually increase load to find the "peak" of the latency curve
        ramping_stress: {
            executor: 'ramping-arrival-rate',
            startRate: 20,
            timeUnit: '1s',
            preAllocatedVUs: 100,
            maxVUs: 500,
            stages: [
                { duration: '1m', target: 100 }, // Warm up
                { duration: '2m', target: 400 }, // Push to 400 req/s
                { duration: '1m', target: 600 }, // Stress test limit
                { duration: '1m', target: 0 }, // Recovery
            ],
        },
        // Simultaneous burst of users to test locking and connection pooling
        burst_spike: {
            executor: 'per-vu-iterations',
            vus: 100,
            iterations: 50,
            startTime: '2m', // Fires when ramping_stress is hitting its peak
        },
    },
    thresholds: {
        http_req_duration: ['p(95)<200'], // Ingestion must stay snappy
        server_errors: ['rate<0.01'], // Stability: <1% 5xx errors
        errors: ['rate<0.05'], // Total failure threshold
    },
};

export default function () {
    const timestamp = Math.floor(Date.now() / 1000);
    const eventId = `evt_max_${__VU}_${__ITER}_${Date.now()}`;

    // Randomize payload size (0KB to 8KB) to stress DB storage and JSON parsing
    const paddingSize = Math.floor(Math.random() * 8000);
    const padding = 'x'.repeat(paddingSize);

    const payload = JSON.stringify({
        id: eventId,
        object: 'event',
        api_version: '2026-04-22.dahlia',
        created: timestamp,
        data: {
            object: {
                id: `seti_${__VU}_${__ITER}`,
                status: 'requires_confirmation',
                metadata: {
                    vu: String(__VU),
                    iter: String(__ITER),
                    stress_padding: padding,
                },
            },
        },
        type: 'setup_intent.created',
    });

    const signedPayload = `${timestamp}.${payload}`;
    const signature = crypto.hmac('sha256', secret, signedPayload, 'hex');

    const params = {
        timeout: '10s',
        headers: {
            'Content-Type': 'application/json',
            'Stripe-Signature': `t=${timestamp},v1=${signature}`,
        },
        tags: { scenario: 'max_stress' },
    };

    const res = http.post(BASE_URL, payload, params);

    // Record metrics
    webhookDuration.add(res.timings.duration);
    payloadSize.add(payload.length);
    errorRate.add(res.status !== 200);
    serverErrors.add(res.status >= 500);

    check(res, {
        'ingestion status 200': (r) => r.status === 200,
        'backend accepted': (r) => {
            try {
                return r.json('status') === 'queued';
            } catch (_) {
                return false;
            }
        },
    });
}
