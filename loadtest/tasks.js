import http from 'k6/http';
import { check, fail } from 'k6';
import { textSummary } from 'https://jslib.k6.io/k6-summary/0.1.0/index.js';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';
const DURATION = __ENV.DURATION || '2m';
const LIST_RATE = Number(__ENV.LIST_RATE || 50);
const CREATE_RATE = Number(__ENV.CREATE_RATE || 10);
const SEED_TASKS = Number(__ENV.SEED_TASKS || 50);

export const options = {
  scenarios: {
    list_tasks: {
      executor: 'constant-arrival-rate',
      exec: 'listTasks',
      rate: LIST_RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 20,
      maxVUs: 100,
    },
    create_tasks: {
      executor: 'constant-arrival-rate',
      exec: 'createTask',
      rate: CREATE_RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: 10,
      maxVUs: 50,
    },
  },
  summaryTrendStats: ['avg', 'min', 'med', 'p(90)', 'p(95)', 'p(99)', 'max'],
  thresholds: {
    'http_req_duration{endpoint:list_tasks}': ['p(95)<200', 'p(99)<500'],
    'http_req_duration{endpoint:create_task}': ['p(95)<300'],
    'http_req_failed{scenario:list_tasks}': ['rate<0.01'],
    'http_req_failed{scenario:create_tasks}': ['rate<0.01'],
    'http_reqs{endpoint:list_tasks}': ['count>0'],
    'http_reqs{endpoint:create_task}': ['count>0'],
  },
};

const JSON_HEADERS = { 'Content-Type': 'application/json' };

function authHeaders(token) {
  return { headers: { ...JSON_HEADERS, Authorization: `Bearer ${token}` } };
}

function taskPayload(i) {
  const priorities = ['low', 'medium', 'high', 'urgent'];
  return JSON.stringify({
    title: `Carga ${i}`,
    description: 'Tarefa criada pelo teste de carga',
    status: 'todo',
    priority: priorities[i % priorities.length],
  });
}

export function setup() {
  const credentials = JSON.stringify({
    email: `k6-${Date.now()}@loadtest.local`,
    password: 'loadtest-password',
  });
  const res = http.post(`${BASE_URL}/api/identity/register`, credentials, {
    headers: JSON_HEADERS,
    tags: { endpoint: 'register' },
  });
  if (res.status !== 201) {
    fail(`registro falhou: status ${res.status} body ${res.body}`);
  }
  const token = res.json('token');
  for (let i = 0; i < SEED_TASKS; i++) {
    const seed = http.post(`${BASE_URL}/api/tasks/tasks`, taskPayload(i), {
      ...authHeaders(token),
      tags: { endpoint: 'seed' },
    });
    if (seed.status !== 201) {
      fail(`seed falhou: status ${seed.status} body ${seed.body}`);
    }
  }
  return { token };
}

export function listTasks(data) {
  const res = http.get(`${BASE_URL}/api/tasks/tasks`, {
    ...authHeaders(data.token),
    tags: { endpoint: 'list_tasks' },
  });
  check(res, { 'GET /tasks 200': (r) => r.status === 200 });
}

export function createTask(data) {
  const res = http.post(`${BASE_URL}/api/tasks/tasks`, taskPayload(__ITER), {
    ...authHeaders(data.token),
    tags: { endpoint: 'create_task' },
  });
  check(res, { 'POST /tasks 201': (r) => r.status === 201 });
}

function endpointStats(data, endpoint) {
  const duration = data.metrics[`http_req_duration{endpoint:${endpoint}}`];
  const reqs = data.metrics[`http_reqs{endpoint:${endpoint}}`];
  if (!duration || !reqs) {
    return null;
  }
  return {
    p50_ms: duration.values.med,
    p95_ms: duration.values['p(95)'],
    p99_ms: duration.values['p(99)'],
    requests: reqs.values.count,
    throughput_rps: reqs.values.count / (data.state.testRunDurationMs / 1000),
  };
}

export function handleSummary(data) {
  const thresholds = {};
  for (const [name, metric] of Object.entries(data.metrics)) {
    if (metric.thresholds) {
      thresholds[name] = Object.fromEntries(
        Object.entries(metric.thresholds).map(([rule, result]) => [rule, result.ok]),
      );
    }
  }
  const report = {
    base_url: BASE_URL,
    duration: DURATION,
    list_rate_rps: LIST_RATE,
    create_rate_rps: CREATE_RATE,
    endpoints: {
      'GET /tasks': endpointStats(data, 'list_tasks'),
      'POST /tasks': endpointStats(data, 'create_task'),
    },
    thresholds,
  };
  return {
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
    'results-summary.json': JSON.stringify(report, null, 2),
  };
}
