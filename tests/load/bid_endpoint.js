// T107: k6 load test script for 1000 QPS validation
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

const errorRate = new Rate('errors');
const bidLatency = new Trend('bid_latency_ms');
const successfulBids = new Counter('successful_bids');

export const options = {
  stages: [
    { duration: '1m', target: 500 },
    { duration: '1m', target: 1000 },
    { duration: '5m', target: 1000 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    'http_req_duration': ['p(95)<100', 'p(99)<120'],
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const bidRequest = {
    id: 'test-auction',
    imp: [{
      id: 'imp-1',
      banner: { w: 300, h: 250 },
      bidfloor: 1.0,
    }],
    site: {
      domain: 'example.com',
      page: 'https://example.com/test',
    },
    device: {
      ip: '192.0.2.1',
      devicetype: 2,
      geo: { country: 'USA' },
    },
  };
  
  const response = http.post(
    BASE_URL + '/bid',
    JSON.stringify(bidRequest),
    { headers: { 'Content-Type': 'application/json' } }
  );

  bidLatency.add(response.timings.duration);

  const passed = check(response, {
    'status is 200': (r) => r.status === 200,
    'latency < 100ms': (r) => r.timings.duration < 100,
  });

  if (!passed) {
    errorRate.add(1);
  } else {
    errorRate.add(0);
  }
}
