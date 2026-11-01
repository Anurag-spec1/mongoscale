import { thresholdsFor, rampingScenario } from './scenarios/config.js';
import { setupTarget, runJourney } from './scenarios/journey.js';

export const options = {
  scenarios: rampingScenario([
    { duration: '5s', target: 10 },
    { duration: '20s', target: 10 },
    { duration: '5s', target: 0 },
  ]),
  thresholds: thresholdsFor(800, 0.01),
};

export function setup() {
  return setupTarget();
}

export default function (data) {
  runJourney(data);
}
