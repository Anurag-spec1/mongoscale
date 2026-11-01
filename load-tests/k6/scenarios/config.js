export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
export const ADMIN_EMAIL = __ENV.ADMIN_EMAIL || 'admin@mongoscale.dev';
export const ADMIN_PASSWORD = __ENV.ADMIN_PASSWORD || 'Admin@12345';
export const TARGET_MODE = __ENV.TARGET_MODE || '';
export const SEEDED_USERS = parseInt(__ENV.SEEDED_USERS || '5000', 10);
export const SEED_USER_PASSWORD = __ENV.SEED_USER_PASSWORD || 'Password123!';
export const THINK_TIME_MIN = parseFloat(__ENV.THINK_TIME_MIN || '0.8');
export const THINK_TIME_MAX = parseFloat(__ENV.THINK_TIME_MAX || '2.5');

export function thresholdsFor(p95Milliseconds, maxFailureRate) {
  return {
    http_req_failed: [{ threshold: `rate<${maxFailureRate}`, abortOnFail: false }],
    http_req_duration: [
      { threshold: `p(95)<${p95Milliseconds}`, abortOnFail: false },
      { threshold: 'p(99)<5000', abortOnFail: false },
    ],
    journey_failures: [{ threshold: `rate<${maxFailureRate}`, abortOnFail: false }],
  };
}

export function rampingScenario(stages) {
  return {
    user_journey: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages,
      gracefulRampDown: '15s',
      gracefulStop: '30s',
      exec: 'default',
    },
  };
}
