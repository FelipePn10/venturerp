import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const base = __ENV.BASE_URL || 'http://localhost:5070';
const token = __ENV.TOKEN || '';
const group = Number(__ENV.RESOURCE_GROUP_ID || 1);
const errors = new Rate('sequencing_errors');
const latency = new Trend('sequencing_latency', true);

export const options = {
  scenarios: { planners: { executor: 'ramping-vus', startVUs: 2, stages: [
    { duration: '20s', target: Number(__ENV.VUS || 30) },
    { duration: __ENV.DURATION || '3m', target: Number(__ENV.VUS || 30) },
    { duration: '20s', target: 0 },
  ] } },
  thresholds: { sequencing_errors: ['rate<0.01'], sequencing_latency: ['p(95)<1500','p(99)<3000'], http_req_failed: ['rate<0.01'] },
};

export default function () {
  const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };
  const now = new Date(); const to = new Date(now.getTime() + 30 * 86400000);
  const calls = [
    () => http.get(`${base}/api/aps/sequence/resources`, { headers }),
    () => http.get(`${base}/api/aps/resource-groups`, { headers }),
    () => http.get(`${base}/api/aps/machine-calendars`, { headers }),
    () => http.post(`${base}/api/aps/sequence/view`, JSON.stringify({ from: now.toISOString(), to: to.toISOString(), resource_group_id: group, time_unit: 'MINUTE', refresh_value: 12 }), { headers }),
  ];
  const response = calls[Math.floor(Math.random() * calls.length)]();
  latency.add(response.timings.duration); const ok = check(response, { 'status 200': r => r.status === 200 }); errors.add(!ok); sleep(0.2);
}
