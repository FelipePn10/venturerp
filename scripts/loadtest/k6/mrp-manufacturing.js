import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

const errors = new Rate('mrp_manufacturing_errors');
const latency = new Trend('mrp_manufacturing_latency', true);
const base = __ENV.BASE_URL || 'http://localhost:5070';
const token = __ENV.TOKEN || '';
const planCode = __ENV.PLAN_CODE || '1001';

export const options = {
  scenarios: {
    factory_shift: {
      executor: 'ramping-vus',
      startVUs: 5,
      stages: [
        { duration: '30s', target: Number(__ENV.VUS || 50) },
        { duration: __ENV.DURATION || '5m', target: Number(__ENV.VUS || 50) },
        { duration: '30s', target: 0 },
      ],
    },
  },
  thresholds: {
    mrp_manufacturing_errors: ['rate<0.01'],
    mrp_manufacturing_latency: ['p(95)<1500', 'p(99)<3000'],
    http_req_failed: ['rate<0.01'],
  },
};

const endpoints = [
  '/api/mrp-reports/reorder-point?limit=100',
  `/api/mrp-reports/grouped-needs?plan_code=${encodeURIComponent(planCode)}&limit=100`,
  '/api/production-order/maintenance',
  '/api/production-order/delivery-candidates',
  '/api/purchase-order/consultation?limit=100',
];

export default function () {
  const response = http.get(`${base}${endpoints[Math.floor(Math.random() * endpoints.length)]}`, {
    headers: { Authorization: `Bearer ${token}` },
    tags: { domain: 'mrp-manufacturing' },
  });
  latency.add(response.timings.duration);
  const ok = check(response, { 'status 200': (r) => r.status === 200 });
  errors.add(!ok);
  sleep(0.2);
}
