import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    operational_queries: {
      executor: 'ramping-arrival-rate',
      startRate: 5,
      timeUnit: '1s',
      preAllocatedVUs: 10,
      maxVUs: 60,
      stages: [
        { target: Number(__ENV.TARGET_RPS || 25), duration: __ENV.RAMP_DURATION || '30s' },
        { target: Number(__ENV.TARGET_RPS || 25), duration: __ENV.HOLD_DURATION || '2m' },
        { target: 0, duration: '15s' },
      ],
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<750', 'p(99)<1500'],
    checks: ['rate>0.99'],
  },
};

const baseURL = __ENV.BASE_URL || 'http://localhost:5070';
const token = __ENV.TOKEN || '';
const params = { headers: { Authorization: `Bearer ${token}` } };

export default function () {
  const prices = http.get(`${baseURL}/api/third-party-services/prices?limit=100&price_type=BOTH`, params);
  check(prices, { 'prices 200': (r) => r.status === 200 });

  const orders = http.get(`${baseURL}/api/third-party-services/orders?limit=100&position=PENDING`, params);
  check(orders, { 'orders 200': (r) => r.status === 200 });

  const report = http.get(`${baseURL}/api/third-party-services/orders/report?limit=100&format=csv`, params);
  check(report, { 'report 200': (r) => r.status === 200 });

  const conversions = http.get(`${baseURL}/api/third-party-services/global-conversions`, params);
  check(conversions, { 'global conversions 200': (r) => r.status === 200 });

  if (__ENV.PLAN_CODE) {
    const planned = http.get(`${baseURL}/api/third-party-services/orders?plan_code=${__ENV.PLAN_CODE}&statuses=PLANNED&limit=100`, params);
    check(planned, { 'planned service orders 200': (r) => r.status === 200 });
  }

  if (__ENV.ITEM_CODE && __ENV.OPERATION_ID) {
    const cost = http.get(`${baseURL}/api/third-party-services/cost?item_code=${__ENV.ITEM_CODE}&operation_id=${__ENV.OPERATION_ID}&mode=STANDARD`, params);
    check(cost, { 'standard service cost 200': (r) => r.status === 200 });
  }
  sleep(0.1);
}
