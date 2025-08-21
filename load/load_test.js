import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '30s', target: 1000 },
  ],
};

export default function () {
  const res = http.post('http://localhost:8080/v1/holds', JSON.stringify({
    event_id: '00000000-0000-0000-0000-000000000000',
    seats: ['A1'],
    user_id: '00000000-0000-0000-0000-000000000000',
  }), {
    headers: { 'Content-Type': 'application/json', 'Idempotency-Key': `${Math.random()}` },
  });
  check(res, { 'status was 201': (r) => r.status == 201 });
  sleep(1);
}