import http from 'k6/http';
import { check } from 'k6';
import { Trend, Rate } from 'k6/metrics';

const dbOnlyLatency = new Trend('db_only_latency', true);
const cacheLatency = new Trend('cache_aside_latency', true);
const errors = new Rate('request_errors');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
    scenarios: {
        db_only: {
            executor: 'constant-arrival-rate',

            rate: 100,
            timeUnit: '1s',

            duration: '2m',

            preAllocatedVUs: 20,
            maxVUs: 100,

            exec: 'dbOnly',
        },

        cache_aside: {
            executor: 'constant-arrival-rate',

            rate: 100,
            timeUnit: '1s',

            duration: '2m',

            preAllocatedVUs: 20,
            maxVUs: 100,

            exec: 'cacheAside',

            // Don't run both workloads at the same time.
            startTime: '2m10s',
        },
    },

    thresholds: {
        request_errors: ['rate<0.01'],
    },
};

export function dbOnly() {
    const id = hotProductID();

    const res = http.get(
        `${BASE_URL}/api/products/${id}?cache=false`,
        {
            tags: {
                test_type: 'db_only',
            },
        },
    );

    dbOnlyLatency.add(res.timings.duration);

    errors.add(res.status !== 200);

    check(res, {
        'DB-only status = 200': (r) => r.status === 200,
    });
}

export function cacheAside() {
    const id = hotProductID();

    const res = http.get(
        `${BASE_URL}/api/products/${id}`,
        {
            tags: {
                test_type: 'cache_aside',
            },
        },
    );

    cacheLatency.add(res.timings.duration);

    errors.add(res.status !== 200);

    check(res, {
        'Cache-aside status = 200': (r) => r.status === 200,
    });
}

// 80% of requests target only 100 products.
// 20% target the entire dataset.
//
// This creates realistic-ish hot-key locality.
function hotProductID() {
    if (Math.random() < 0.80) {
        return randomInt(1, 100);
    }

    return randomInt(1, 100000);
}

function randomInt(min, max) {
    return Math.floor(
        Math.random() * (max - min + 1),
    ) + min;
}