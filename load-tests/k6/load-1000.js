import { thresholdsFor, rampingScenario } from './scenarios/config.js';
import { setupTarget, runJourney } from './scenarios/journey.js';

export const options = {
  scenarios: rampingScenario([
    { duration: '2m', target: 1000 },
    { duration: '3m', target: 1000 },
    { duration: '1m', target: 0 },
  ]),
  thresholds: thresholdsFor(1500, 0.05),
};

export function setup() {
  return setupTarget();
}

export default function (data) {
  runJourney(data);
}
