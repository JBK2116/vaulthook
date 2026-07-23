import http from "k6/http";
import crypto from "k6/crypto";
import { check } from "k6";
import { Rate, Trend, Counter } from "k6/metrics";

const secret =
  "whsec_5b5db0374a5cf98206d891a77ee2595be04c01a3d7710a9d5c726f52a3887c2f";
const BASE_URL = "http://localhost:8080/api/webhooks/stripe";

const errorRate = new Rate("errors");
const serverErrors = new Rate("server_errors");
const webhookDuration = new Trend("webhook_duration");
const payloadSize = new Trend("payload_size_bytes");
const droppedCounter = new Counter("dropped_iterations_custom");

export const options = {
  scenarios: {
    // Find the real ceiling: keep climbing until something breaks
    ramping_stress: {
      executor: "ramping-arrival-rate",
      startRate: 50,
      timeUnit: "1s",
      preAllocatedVUs: 200,
      maxVUs: 3000,
      stages: [
        { duration: "1m", target: 200 },
        { duration: "2m", target: 500 },
        { duration: "2m", target: 1000 },
        { duration: "2m", target: 1500 },
        { duration: "2m", target: 2000 },
        { duration: "2m", target: 2500 },
        { duration: "1m", target: 0 }, // recovery
      ],
    },
    // Concurrency/lock/connection-pool test, fired at peak load
    burst_spike: {
      executor: "per-vu-iterations",
      vus: 300,
      iterations: 50,
      startTime: "7m", // lines up with the 1500-2000 target window
    },
  },
  thresholds: {
    http_req_duration: ["p(95)<200"],
    server_errors: ["rate<0.01"],
    errors: ["rate<0.05"],
    dropped_iterations: ["count<50"],
  },
};

export default function () {
  const timestamp = Math.floor(Date.now() / 1000);
  const eventId = `evt_max_${__VU}_${__ITER}_${Date.now()}`;

  const paddingSize = Math.floor(Math.random() * 8000);
  const padding = "x".repeat(paddingSize);

  const payload = JSON.stringify({
    id: eventId,
    object: "event",
    api_version: "2026-04-22.dahlia",
    created: timestamp,
    data: {
      object: {
        id: `seti_${__VU}_${__ITER}`,
        status: "requires_confirmation",
        metadata: {
          vu: String(__VU),
          iter: String(__ITER),
          stress_padding: padding,
        },
      },
    },
    type: "setup_intent.created",
  });

  const signedPayload = `${timestamp}.${payload}`;
  const signature = crypto.hmac("sha256", secret, signedPayload, "hex");

  const params = {
    timeout: "10s",
    headers: {
      "Content-Type": "application/json",
      "Stripe-Signature": `t=${timestamp},v1=${signature}`,
    },
    tags: { scenario: "max_stress" },
  };

  const res = http.post(BASE_URL, payload, params);

  webhookDuration.add(res.timings.duration);
  payloadSize.add(payload.length);
  errorRate.add(res.status !== 200);
  serverErrors.add(res.status >= 500);

  check(res, {
    "ingestion status 200": (r) => r.status === 200,
    "backend accepted": (r) => {
      try {
        return r.json("status") === "queued";
      } catch (_) {
        return false;
      }
    },
  });
}

export function handleSummary(data) {
  return {
    "summary.json": JSON.stringify(data, null, 2),
    stdout: JSON.stringify(
      {
        p95_duration: data.metrics.http_req_duration?.values["p(95)"],
        p99_duration: data.metrics.http_req_duration?.values["p(99)"],
        error_rate: data.metrics.errors?.values.rate,
        server_error_rate: data.metrics.server_errors?.values.rate,
        dropped_iterations: data.metrics.dropped_iterations?.values.count,
        total_requests: data.metrics.http_reqs?.values.count,
        req_per_sec: data.metrics.http_reqs?.values.rate,
      },
      null,
      2,
    ),
  };
}
