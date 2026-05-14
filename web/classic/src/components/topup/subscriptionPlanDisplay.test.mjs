import assert from 'node:assert/strict';
import { shouldHighlightSubscriptionPlan } from './subscriptionPlanDisplay.js';

assert.equal(shouldHighlightSubscriptionPlan({ title: 'First plan' }), false);
assert.equal(
  shouldHighlightSubscriptionPlan({
    title: 'Manual recommendation',
    recommended: true,
  }),
  true,
);
assert.equal(shouldHighlightSubscriptionPlan({ recommended: false }), false);

console.log('subscriptionPlanDisplay tests passed');
