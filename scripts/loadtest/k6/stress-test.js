// Stress Test – sobe carga ate quebrar para achar o limite maximo da API.
// Uso: docker run --rm -i -v $(pwd):/scripts -e TARGET_URL=https://api.venturerp.com grafana/k6 run /scripts/stress-test.js

import http from "k6/http";
import { check, sleep, group } from "k6";
import { Counter, Rate } from "k6/metrics";

const BASE_URL = __ENV.TARGET_URL || "https://api.venturerp.com";
const errors = new Counter("errors");
const errorRate = new Rate("error_rate");

export const options = {
  stages: [
    { duration: "30s", target: 20 },
    { duration: "1m",  target: 100 },
    { duration: "1m",  target: 300 },
    { duration: "1m",  target: 500 },
    { duration: "1m",  target: 750 },
    { duration: "1m",  target: 1000 },
    { duration: "30s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<2000"],
  },
};

export default function () {
  const vuId = __VU;

  // Apenas /health — endpoint mais leve, ideal para testar throughput bruto
  const res = http.get(`${BASE_URL}/health`, { tags: { endpoint: "health" } });

  const ok = check(res, {
    "GET /health status 200": (r) => r.status === 200,
  });
  if (!ok) { errors.add(1); errorRate.add(1); }
  else { errorRate.add(0); }

  sleep(0.1);
}

export function handleSummary(data) {
  const m = data.metrics;
  const total = m.http_reqs ? m.http_reqs.values.count : 0;
  const failed = m.http_req_failed ? m.http_req_failed.values.passes : 0;
  const dur95 = m.http_req_duration ? m.http_req_duration.values["p(95)"].toFixed(1) : "N/A";
  const dur99 = m.http_req_duration ? m.http_req_duration.values["p(99)"].toFixed(1) : "N/A";
  const rps = m.http_reqs ? m.http_reqs.values.rate.toFixed(1) : "N/A";
  const maxRps = m.http_reqs ? m.http_reqs.values.max.toFixed(1) : "N/A";

  const report = `
╔══════════════════════════════════════════════════════════════╗
║            STRESS TEST REPORT (Breaking Point)               ║
╠══════════════════════════════════════════════════════════════╣
║  Total Requests:     ${String(total).padStart(8)}                             ║
║  Failed:             ${String(failed).padStart(8)}                             ║
║  Avg Requests/sec:   ${String(rps).padStart(8)}                             ║
║  Max Requests/sec:   ${String(maxRps).padStart(8)}                             ║
║  p95 Duration:       ${String(dur95 || "N/A").padStart(8)} ms                          ║
║  p99 Duration:       ${String(dur99 || "N/A").padStart(8)} ms                          ║
╚══════════════════════════════════════════════════════════════╝
`;

  return {
    "stdout": report,
    "scripts/loadtest/results/stress-test-summary.json": JSON.stringify(data, null, 2),
  };
}
