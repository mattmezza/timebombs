def legacy():
    # TIMEBOMB(2025-09-01): Remove v1 endpoints after migration complete.
    #   The new v2 endpoints are already serving 90% of traffic.
    #   Blocked by: mobile app rollout to force-update v1 clients.
    pass

# TODO: this is not a timebomb.


def another():
    # TIMEBOMB(2030-12-31, PY-1): Rewrite with async.
    return 1
