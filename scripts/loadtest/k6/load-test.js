// Load Test – rampa gradual para medir capacidade da API.
// Uso: docker run --rm -i -v $(pwd):/scripts -e TARGET_URL=https://api.venturerp.com grafana/k6 run /scripts/load-test.js
// Ou com k6 local: k6 run scripts/loadtest/k6/load-test.js

import http from "k6/http";
import { check, sleep, group } from "k6";
import { Trend, Counter, Rate } from "k6/metrics";

const BASE_URL = __ENV.TARGET_URL || "https://api.venturerp.com";
if (!BASE_URL) {
  console.error("TARGET_URL nao definido");
}

const responseTime = new Trend("response_time", true);
const errors = new Counter("errors");
const errorRate = new Rate("error_rate");

export const options = {
  stages: [
    { duration: "30s", target: 5 },   // aquecimento
    { duration: "1m",  target: 20 },  // carga leve
    { duration: "1m",  target: 50 },  // carga media
    { duration: "1m",  target: 100 }, // carga pesada
    { duration: "1m",  target: 200 }, // estresse
    { duration: "30s", target: 0 },   // cooldown
  ],
  thresholds: {
    http_req_duration: ["p(95)<1000"],  // 95% das reqs abaixo de 1s
    http_req_failed:   ["rate<0.05"],    // menos de 5% de falhas
  },
  summaryTrendStats: ["avg", "min", "med", "max", "p(90)", "p(95)", "p(99)"],
};

function loginPayload(idx) {
  return JSON.stringify({
    email: `loadtest_user_${idx}@venturerp.com`,
    password: "TestPass123!",
  });
}

function registerPayload(idx) {
  return JSON.stringify({
    email: `loadtest_new_${idx}_${Date.now()}@venturerp.com`,
    password: "TestPass123!",
  });
}

export default function () {
  const vuId = __VU;
  const iterId = __ITER;

  group("GET /health", function () {
    const res = http.get(`${BASE_URL}/health`, { tags: { endpoint: "health" } });
    responseTime.add(res.timings.duration);
    const ok = check(res, {
      "GET /health status 200": (r) => r.status === 200,
      "GET /health body ok": (r) => r.body.includes("ok"),
    });
    if (!ok) { errors.add(1); errorRate.add(1); }
    else { errorRate.add(0); }
  });

  sleep(0.5);

  group("POST /login", function () {
    const headers = { "Content-Type": "application/json" };
    const payload = loginPayload(vuId);
    const res = http.post(`${BASE_URL}/login`, payload, {
      headers,
      tags: { endpoint: "login" },
    });
    responseTime.add(res.timings.duration);
    const ok = check(res, {
      "POST /login <= 401": (r) => r.status <= 401, // normal: pode nao ter o user
    });
    if (!ok) { errors.add(1); errorRate.add(1); }
    else { errorRate.add(0); }
  });

  sleep(0.5);

  group("POST /register", function () {
    const headers = { "Content-Type": "application/json" };
    const payload = registerPayload(iterId);
    const res = http.post(`${BASE_URL}/register`, payload, {
      headers,
      tags: { endpoint: "register" },
    });
    responseTime.add(res.timings.duration);
    const ok = check(res, {
      "POST /register status 201": (r) => r.status === 201,
    });
    if (!ok) { errors.add(1); errorRate.add(1); }
    else { errorRate.add(0); }
  });

  sleep(1);
}

export function handleSummary(data) {
  const m = data.metrics;
  const total = m.http_reqs ? m.http_reqs.values.count : 0;
  const failed = m.http_req_failed ? m.http_req_failed.values.passes : 0;
  const dur95 = m.http_req_duration ? m.http_req_duration.values["p(95)"].toFixed(1) : "N/A";
  const dur99 = m.http_req_duration ? m.http_req_duration.values["p(99)"].toFixed(1) : "N/A";
  const avg = m.http_req_duration ? m.http_req_duration.values.avg.toFixed(1) : "N/A";
  const rps = m.http_reqs ? m.http_reqs.values.rate.toFixed(1) : "N/A";

  const report = `
╔══════════════════════════════════════════════════════════════╗
║               LOAD TEST REPORT                               ║
╠══════════════════════════════════════════════════════════════╣
║  Total Requests:    ${String(total).padStart(8)}                              ║
║  Failed:            ${String(failed).padStart(8)}                              ║
║  Requests/sec:      ${String(rps).padStart(8)}                              ║
║  Avg Duration:      ${String(avg).padStart(8)} ms                           ║
║  p95 Duration:      ${String(dur95).padStart(8)} ms                           ║
║  p99 Duration:      ${String(dur99).padStart(8)} ms                           ║
╠══════════════════════════════════════════════════════════════╣
║  Thresholds:                                                 ║
║  p(95) < 1000ms:   ${(m.http_req_duration && m.http_req_duration.values["p(95)"] < 1000) ? "PASS" : "FAIL"}                                     ║
║  error rate < 5%:  ${(m.http_req_failed && m.http_req_failed.values.rate < 0.05) ? "PASS" : "FAIL"}                                     ║
╚══════════════════════════════════════════════════════════════╝
`;

  return {
    "stdout": report,
    "scripts/loadtest/results/load-test-summary.json": JSON.stringify(data, null, 2),
  };
}
