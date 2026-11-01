import { thresholdsFor, rampingScenario } from './scenarios/config.js';
import { setupTarget, runJourney } from './scenarios/journey.js';

export const options = {
  scenarios: rampingScenario([
    { duration: '30s', target: 100 },
    { duration: '1m', target: 100 },
    { duration: '10s', target: 5000 },
    { duration: '2m', target: 5000 },
    { duration: '10s', target: 100 },
    { duration: '1m', target: 100 },
    { duration: '20s', target: 0 },
  ]),
  thresholds: thresholdsFor(1500, 0.05),
};

export function setup() {
  return setupTarget();
}

export default function (data) {
  runJourney(data);
}
