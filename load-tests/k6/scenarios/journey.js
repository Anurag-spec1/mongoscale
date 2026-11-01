import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter } from 'k6/metrics';
import {
  BASE_URL,
  ADMIN_EMAIL,
  ADMIN_PASSWORD,
  TARGET_MODE,
  SEEDED_USERS,
  SEED_USER_PASSWORD,
  THINK_TIME_MIN,
  THINK_TIME_MAX,
} from './config.js';

http.setResponseCallback(http.expectedStatuses({ min: 200, max: 399 }, 409));

export const journeyFailures = new Rate('journey_failures');
export const ordersCreated = new Counter('orders_created');
export const reviewsCreated = new Counter('reviews_created');

const SEARCH_TERMS = ['pro', 'smart', 'classic', 'ultra', 'eco', 'premium', 'compact', 'wireless'];
const SORTS = ['price_asc', 'price_desc', 'rating', 'newest'];

let session = null;

function jsonHeaders(extra) {
  return Object.assign({ 'Content-Type': 'application/json', 'X-Virtual-User': `k6-${__VU}` }, extra || {});
}

function authHeaders() {
  return jsonHeaders({ Authorization: `Bearer ${session.token}` });
}

function pick(list) {
  return list[Math.floor(Math.random() * list.length)];
}

function seededEmail(index) {
  return `user${String(index).padStart(5, '0')}@example.com`;
}

function record(response, name, expectedStatuses) {
  const accepted = expectedStatuses || [200];
  const ok = check(response, { [`${name} succeeded`]: (r) => accepted.includes(r.status) });
  journeyFailures.add(!ok);
  return ok;
}

export function setupTarget() {
  const login = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email: ADMIN_EMAIL, password: ADMIN_PASSWORD }),
    { headers: jsonHeaders(), tags: { name: 'admin_login' } },
  );
  if (login.status !== 200) {
    throw new Error(`admin login failed with status ${login.status}`);
  }
  const adminToken = login.json('data.token');
  const adminHeaders = jsonHeaders({ Authorization: `Bearer ${adminToken}` });

  if (TARGET_MODE) {
    const toggle = http.post(`${BASE_URL}/api/v1/system/optimizations/${TARGET_MODE}/toggle`, null, {
      headers: adminHeaders,
      tags: { name: 'toggle_mode' },
      timeout: '120s',
    });
    if (toggle.status !== 200) {
      throw new Error(`switching to mode ${TARGET_MODE} failed with status ${toggle.status}: ${toggle.body}`);
    }
    console.log(`optimization mode set to ${TARGET_MODE}`);
  }

  const products = [];
  let cursor = '';
  for (let page = 1; page <= 10; page += 1) {
    const query = cursor ? `limit=100&cursor=${cursor}` : `limit=100&page=${page}`;
    const response = http.get(`${BASE_URL}/api/v1/products?${query}`, {
      headers: jsonHeaders(),
      tags: { name: 'catalog_bootstrap' },
    });
    if (response.status !== 200) {
      break;
    }
    const body = response.json('data');
    body.items.forEach((item) => products.push({ id: item.id, category: item.category }));
    if (!body.hasMore) {
      break;
    }
    cursor = body.nextCursor || '';
  }
  if (products.length === 0) {
    throw new Error('no products available, run the seed first');
  }
  return { products };
}

function ensureSession() {
  if (session) {
    return true;
  }
  if (__VU % 10 === 0) {
    const email = `k6-${__VU}-${Date.now()}@example.com`;
    const registration = http.post(
      `${BASE_URL}/api/v1/auth/register`,
      JSON.stringify({ name: `K6 User ${__VU}`, email, password: SEED_USER_PASSWORD }),
      { headers: jsonHeaders(), tags: { name: 'auth_register' } },
    );
    if (record(registration, 'register', [201])) {
      session = { token: registration.json('data.token') };
      return true;
    }
    return false;
  }
  const index = ((__VU - 1) % SEEDED_USERS) + 1;
  const login = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email: seededEmail(index), password: SEED_USER_PASSWORD }),
    { headers: jsonHeaders(), tags: { name: 'auth_login' } },
  );
  if (record(login, 'login')) {
    session = { token: login.json('data.token') };
    return true;
  }
  return false;
}

function browseProducts() {
  const response = http.get(`${BASE_URL}/api/v1/products?limit=20&sort=${pick(SORTS)}`, {
    headers: authHeaders(),
    tags: { name: 'products_list' },
  });
  record(response, 'browse products');
}

function browseCategory(data) {
  const category = encodeURIComponent(pick(data.products).category);
  const response = http.get(`${BASE_URL}/api/v1/products/category/${category}?limit=20&sort=price_asc`, {
    headers: authHeaders(),
    tags: { name: 'products_by_category' },
  });
  record(response, 'category browse');
}

function searchProducts() {
  const response = http.get(`${BASE_URL}/api/v1/products?limit=20&q=${pick(SEARCH_TERMS)}`, {
    headers: authHeaders(),
    tags: { name: 'products_search' },
  });
  record(response, 'search');
}

function viewProduct(data) {
  const product = pick(data.products);
  const detail = http.get(`${BASE_URL}/api/v1/products/${product.id}`, {
    headers: authHeaders(),
    tags: { name: 'product_detail' },
  });
  record(detail, 'product detail');
  const reviews = http.get(`${BASE_URL}/api/v1/reviews/product/${product.id}?limit=10`, {
    headers: authHeaders(),
    tags: { name: 'reviews_list' },
  });
  record(reviews, 'reviews list');
}

function viewSummary(data) {
  const product = pick(data.products);
  const response = http.get(`${BASE_URL}/api/v1/reviews/product/${product.id}/summary`, {
    headers: authHeaders(),
    tags: { name: 'reviews_summary' },
  });
  record(response, 'review summary');
}

function submitReview(data) {
  const product = pick(data.products);
  const response = http.post(
    `${BASE_URL}/api/v1/reviews`,
    JSON.stringify({ productId: product.id, rating: 1 + Math.floor(Math.random() * 5), comment: 'Load test review from k6.' }),
    { headers: authHeaders(), tags: { name: 'reviews_create' } },
  );
  if (record(response, 'create review', [201])) {
    reviewsCreated.add(1);
  }
}

function placeOrder(data) {
  const lines = 1 + Math.floor(Math.random() * 3);
  const chosen = {};
  const items = [];
  while (items.length < lines) {
    const product = pick(data.products);
    if (!chosen[product.id]) {
      chosen[product.id] = true;
      items.push({ productId: product.id, quantity: 1 + Math.floor(Math.random() * 2) });
    }
  }
  const response = http.post(
    `${BASE_URL}/api/v1/orders`,
    JSON.stringify({
      items,
      shippingAddress: {
        fullName: 'K6 Load Tester',
        line1: '12 Benchmark Street',
        city: 'Bengaluru',
        state: 'Karnataka',
        postalCode: '560001',
        country: 'India',
      },
    }),
    { headers: authHeaders(), tags: { name: 'orders_create' } },
  );
  if (record(response, 'create order', [201, 409])) {
    ordersCreated.add(1);
  }
}

function orderHistory() {
  const response = http.get(`${BASE_URL}/api/v1/orders?limit=10`, {
    headers: authHeaders(),
    tags: { name: 'orders_history' },
  });
  record(response, 'order history');
}

export function runJourney(data) {
  if (!ensureSession()) {
    sleep(1);
    return;
  }
  const roll = Math.random() * 100;
  if (roll < 22) {
    browseProducts();
  } else if (roll < 38) {
    browseCategory(data);
  } else if (roll < 48) {
    searchProducts();
  } else if (roll < 63) {
    viewProduct(data);
  } else if (roll < 71) {
    viewSummary(data);
  } else if (roll < 77) {
    submitReview(data);
  } else if (roll < 85) {
    placeOrder(data);
  } else {
    orderHistory();
  }
  sleep(THINK_TIME_MIN + Math.random() * (THINK_TIME_MAX - THINK_TIME_MIN));
}
