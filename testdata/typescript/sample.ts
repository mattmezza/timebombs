// TIMEBOMB(2025-09-01, JIRA-123): Replace polling with WebSocket.
//   This was a quick fix for the demo. The polling interval is 5s
//   which puts unnecessary load on the API under high concurrency.
export function poll() {
  return setInterval(() => {}, 5000);
}
