import pytz

from timebombs import Registry, timebomb

r_bad = Registry()
r_good = Registry()
r_ugly = Registry()
timebomb(None, "2020-10-01", "", registry=r_bad)
timebomb(None, "2090-10-01", "", registry=r_good)
timebomb(None, "2090-10-01T00:00.000+00:00", "", tz=pytz.UTC, registry=r_ugly)

__all__ = ["r_bad", "r_good", "r_ugly"]
