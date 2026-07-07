// Soak Test – carga sustentada por periodo longo para detectar memory leaks.
// Uso: docker run --rm -i -v $(pwd):/scripts -e TARGET_URL=https://api.venturerp.com grafana/k6 run /scripts/soak-test.js

import http from "k6/http";
import { check, sleep } from "k6";
import { Counter, Rate } from "k6/metrics";

const BASE_URL = __ENV.TARGET_URL || "https://api.venturerp.com";
const errors = new Counter("errors");
const errorRate = new Rate("error_rate");

export const options = {
  stages: [
    { duration: "1m",  target: 30 },
    { duration: "8m",  target: 30 },
    { duration: "1m",  target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<2000"],
    http_req_failed:   ["rate<0.05"],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/health`);
  const ok = check(res, { "status 200": (r) => r.status === 200 });
  if (!ok) { errors.add(1); errorRate.add(1); }
  else { errorRate.add(0); }

  sleep(1);
}

export function handleSummary(data) {
  const m = data.metrics;
  const total = m.http_reqs ? m.http_reqs.values.count : 0;
  const failed = m.http_req_failed ? m.http_req_failed.values.passes : 0;
  const durAvg = m.http_req_duration ? m.http_req_duration.values.avg.toFixed(1) : "N/A";
  const dur95 = m.http_req_duration ? m.http_req_duration.values["p(95)"].toFixed(1) : "N/A";
  const durMin = m.http_req_duration ? m.http_req_duration.values.min.toFixed(1) : "N/A";
  const durMax = m.http_req_duration ? m.http_req_duration.values.max.toFixed(1) : "N/A";

  // Detectar tendencia de degradacao (memory leak)
  let trend = "STABLE";
  if (m.http_req_duration) {
    const medians = m.http_req_duration.values.medians || [];
    if (medians.length > 2) {
      const first = medians.slice(0, 5).reduce((a, b) => a + b, 0) / 5;
      const last = medians.slice(-5).reduce((a, b) => a + b, 0) / 5;
      if (last > first * 1.5) trend = "DEGRADING \u26a0";
      else if (last < first * 0.8) trend = "IMPROVING";
    }
  }

  const report = `
╔══════════════════════════════════════════════════════════════╗
║              SOAK TEST REPORT (10 min sustained)             ║
╠══════════════════════════════════════════════════════════════╣
║  Total Requests:    ${String(total).padStart(8)}                              ║
║  Failed:            ${String(failed).padStart(8)}                              ║
║  Avg Duration:      ${String(durAvg).padStart(8)} ms                           ║
║  Min Duration:      ${String(durMin).padStart(8)} ms                           ║
║  Max Duration:      ${String(durMax).padStart(8)} ms                           ║
║  p95 Duration:      ${String(dur95).padStart(8)} ms                           ║
║  Degradation Trend: ${trend.padStart(8)}                                ║
╚══════════════════════════════════════════════════════════════╝
`;

  return {
    "stdout": report,
    "scripts/loadtest/results/soak-test-summary.json": JSON.stringify(data, null, 2),
  };
}
