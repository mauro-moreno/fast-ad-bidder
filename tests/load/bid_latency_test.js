// T062: Performance test - p95 latency < 100ms validation with k6
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const bidLatency = new Trend('bid_latency');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 100 },   // Ramp up to 100 RPS
    { duration: '1m', target: 500 },    // Ramp to 500 RPS
    { duration: '2m', target: 1000 },   // Ramp to 1000 RPS (target load)
    { duration: '3m', target: 1000 },   // Hold at 1000 RPS
    { duration: '30s', target: 0 },     // Ramp down
  ],
  thresholds: {
    'http_req_duration': [
      'p(95)<100',  // 95th percentile must be below 100ms
      'p(99)<120',  // 99th percentile must be below 120ms
    ],
    'http_req_failed': ['rate<0.01'], // Error rate must be below 1%
    'errors': ['rate<0.01'],
  },
};

// Sample bid requests
const bidRequests = [
  {
    id: `auction-${__VU}-${__ITER}`,
    imp: [{
      id: 'imp-1',
      banner: { w: 300, h: 250 },
      bidfloor: 0.50,
    }],
    site: {
      domain: 'example.com',
      page: 'https://example.com/news',
    },
    device: {
      ip: '192.0.2.1',
      devicetype: 2,
      geo: { country: 'USA' },
    },
  },
  {
    id: `auction-${__VU}-${__ITER}`,
    imp: [{
      id: 'imp-2',
      banner: { w: 728, h: 90 },
      bidfloor: 1.00,
    }],
    site: {
      domain: 'news.example.com',
    },
    device: {
      ip: '192.0.2.2',
      devicetype: 2,
      geo: { country: 'USA', region: 'CA' },
    },
  },
  {
    id: `auction-${__VU}-${__ITER}`,
    imp: [{
      id: 'imp-3',
      banner: { w: 320, h: 50 },
      bidfloor: 0.75,
    }],
    app: {
      bundle: 'com.example.app',
      name: 'Example App',
    },
    device: {
      ip: '192.0.2.3',
      devicetype: 1,
      geo: { country: 'USA' },
    },
  },
];

export default function () {
  // Select random bid request
  const bidRequest = bidRequests[Math.floor(Math.random() * bidRequests.length)];

  // Update unique auction ID
  bidRequest.id = `auction-${__VU}-${__ITER}`;

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
  };

  const response = http.post(
    'http://localhost:8080/bid',
    JSON.stringify(bidRequest),
    params
  );

  // Record custom metrics
  bidLatency.add(response.timings.duration);

  // Validate response
  const success = check(response, {
    'status is 200': (r) => r.status === 200,
    'response is valid JSON': (r) => {
      try {
        JSON.parse(r.body);
        return true;
      } catch (e) {
        return false;
      }
    },
    'response has id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id !== undefined;
      } catch (e) {
        return false;
      }
    },
    'latency below 100ms': (r) => r.timings.duration < 100,
  });

  if (!success) {
    errorRate.add(1);
  } else {
    errorRate.add(0);
  }

  // Small sleep to simulate realistic request pacing
  sleep(0.01);
}

export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'results.json': JSON.stringify(data),
  };
}

function textSummary(data, options) {
  const { indent = '', enableColors = false } = options || {};
  let output = '';

  output += `${indent}Bid Latency Performance Test Results\n`;
  output += `${indent}=====================================\n\n`;

  if (data.metrics.http_req_duration) {
    const latency = data.metrics.http_req_duration.values;
    output += `${indent}Request Latency:\n`;
    output += `${indent}  p(50): ${latency['p(50)'].toFixed(2)}ms\n`;
    output += `${indent}  p(95): ${latency['p(95)'].toFixed(2)}ms ${latency['p(95)'] < 100 ? '✓' : '✗'}\n`;
    output += `${indent}  p(99): ${latency['p(99)'].toFixed(2)}ms ${latency['p(99)'] < 120 ? '✓' : '✗'}\n`;
    output += `${indent}  max:   ${latency.max.toFixed(2)}ms\n\n`;
  }

  if (data.metrics.http_reqs) {
    const requests = data.metrics.http_reqs.values;
    output += `${indent}Throughput:\n`;
    output += `${indent}  Total Requests: ${requests.count}\n`;
    output += `${indent}  Requests/sec:   ${requests.rate.toFixed(2)}\n\n`;
  }

  if (data.metrics.http_req_failed) {
    const failed = data.metrics.http_req_failed.values;
    output += `${indent}Error Rate:\n`;
    output += `${indent}  Failed: ${(failed.rate * 100).toFixed(2)}% ${failed.rate < 0.01 ? '✓' : '✗'}\n\n`;
  }

  return output;
}
