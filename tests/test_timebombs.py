from datetime import datetime

import pytest
import pytz
from freezegun import freeze_time

from timebombs import Boom, Registry, __main__, timebomb

DISARMED, ARMED, EXPLODED = 0, 1, 2
stubs = (lambda _: DISARMED, lambda _: ARMED, lambda _: EXPLODED)  # noqa E731
stubs_msg = (
    lambda t: f"I: {t}",
    lambda t: f"W: {t}",
    lambda t: f"E: {t}",
)  # noqa E731
noop = lambda _: None  # noqa E731
ams = pytz.timezone("Europe/Amsterdam")


@freeze_time("2020-11-15")
@pytest.mark.parametrize(
    "exploding_on,expected",
    [
        ("2020-11-14", EXPLODED),
        ("2020-11-14T23:59:59.999", EXPLODED),
        ("2020-11-15T00:00:00.000", EXPLODED),
        ("2020-11-15T00:00:00.001", ARMED),
        ("2020-11-16", ARMED),
        ("2020-11-28T23:59:59.999", ARMED),
        ("2020-11-29T00:00:00.000", ARMED),
        ("2020-11-30T00:00:00.001", DISARMED),
        ("2020-11-30", DISARMED),
    ],
)
def test_state(exploding_on, expected):
    assert timebomb("0", exploding_on, "", *stubs)() == expected


@freeze_time("2020-11-15")
@pytest.mark.parametrize(
    "exploding_on,expected",
    [
        (datetime.fromisoformat("2019-11-14"), EXPLODED),
        (datetime.fromisoformat("2020-11-16"), ARMED),
        (datetime.fromisoformat("2021-11-30"), DISARMED),
    ],
)
def test_with_datetime(exploding_on, expected):
    assert timebomb("0", exploding_on, "", *stubs)() == expected


@freeze_time("2020-11-15")
@pytest.mark.parametrize(
    "exploding_on,arming_on,expected",
    [
        ("2020-11-17", datetime.fromisoformat("2020-11-14"), ARMED),
        ("2020-11-17", datetime.fromisoformat("2020-11-15"), ARMED),
        ("2020-11-17", datetime.fromisoformat("2020-11-16"), DISARMED),
        ("2020-11-14", datetime.fromisoformat("2020-11-10"), EXPLODED),
        ("2020-11-14", datetime.fromisoformat("2020-11-17"), EXPLODED),
    ],
)
def test_with_custom_arm_duration(exploding_on, arming_on, expected):
    assert (
        timebomb("0", exploding_on, "", *stubs, arming_on=arming_on)()
        == expected
    )


@freeze_time("2020-11-15")
@pytest.mark.parametrize(
    "exploding_on,tz,expected",
    [
        ("2020-11-14T00:00:00.000+01:00", ams, EXPLODED),
        ("2020-11-15T00:59:59.999+01:00", ams, EXPLODED),
        ("2020-11-15T01:00:00.000+01:00", ams, EXPLODED),
        ("2020-11-15T01:00:00.001+01:00", ams, ARMED),
        ("2020-11-16T01:00:00.000+01:00", ams, ARMED),
        ("2020-11-29T00:59:59.999+01:00", ams, ARMED),
        ("2020-11-29T01:00:00.000+01:00", ams, ARMED),
        ("2020-11-30T01:00:00.001+01:00", ams, DISARMED),
        ("2020-11-30T01:00:00.000+01:00", ams, DISARMED),
    ],
)
def test_tz(exploding_on, tz, expected):
    assert timebomb("0", exploding_on, "", *stubs, tz=tz)() == expected


@freeze_time("2020-11-15")
@pytest.mark.parametrize(
    "id,exploding_on,expected",
    [
        ("0", "2020-11-14", "E: Timebomb(#0,2020-11-14) 'd'."),
        (None, "2020-11-16", "W: Timebomb(2020-11-16) 'd'."),
        ("P-1", "2020-11-30", "I: Timebomb(#P-1,2020-11-30) 'd'."),
    ],
)
def test_msg(id, exploding_on, expected):
    assert timebomb(id, exploding_on, "d", *stubs_msg)() == expected


def test_registry():
    r = Registry()
    timebomb("0", "2020-11-14", "", registry=r)
    timebomb(None, "2020-11-16", "", registry=r)
    timebomb("P-1", "2020-11-30", "", registry=r)
    assert len(r.bombs) == 3
    now = datetime.fromisoformat("2020-11-15")
    assert len(list(r.bombs_exploded(now))) == 1
    assert next(r.bombs_exploded(now)).id == "0"
    assert len(list(r.bombs_armed(now))) == 1
    assert next(r.bombs_armed(now)).id is None
    assert len(list(r.bombs_disarmed(now))) == 1
    assert next(r.bombs_disarmed(now)).id == "P-1"


@freeze_time("2020-11-15")
def test_extreme():
    with pytest.raises(Boom) as excinfo:
        timebomb(None, "2020-11-14", "", extreme=True)()
    assert str(excinfo.value) == "Timebomb(2020-11-14) ''."


def test_main():
    assert __main__.main("tests", "r_bad", None) == 1
    assert __main__.main("tests", "r_good", None) == 0
    assert __main__.main("tests", "r_ugly", "Europe/Amsterdam") == 0
