import { thresholdsFor, rampingScenario } from './scenarios/config.js';
import { setupTarget, runJourney } from './scenarios/journey.js';

export const options = {
  scenarios: rampingScenario([
    { duration: '5m', target: 10000 },
    { duration: '5m', target: 10000 },
    { duration: '2m', target: 0 },
  ]),
  thresholds: thresholdsFor(1500, 0.05),
  noConnectionReuse: false,
  discardResponseBodies: false,
};

export function setup() {
  return setupTarget();
}

export default function (data) {
  runJourney(data);
}
