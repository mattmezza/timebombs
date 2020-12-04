import sys
from datetime import datetime, timedelta
from importlib import import_module
from typing import Optional

import pytz
import typer


def main(
    module: str = typer.Argument(
        ...,
        help="Reference to the registry variable for collecting timebombs (ex. `module.submodule:variable`).",  # noqa
    ),
    skip_disarmed: bool = typer.Option(
        False, "--skip-disarmed", "-D", help="Skips disarmed timebombs."
    ),
    skip_armed: bool = typer.Option(
        False, "--skip-armed", "-A", help="Skips armed timebombs."
    ),
    skip_exploded: bool = typer.Option(
        False, "--skip-exploded", "-E", help="Skips exploded timebombs."
    ),
    max_disarmed: int = typer.Option(
        0,
        "--max-disarmed",
        "-d",
        help="Maximum allowed number of disarmed timebombs left.",
    ),
    max_armed: int = typer.Option(
        0,
        "--max-armed",
        "-a",
        help="Maximum allowed number of armed timebombs left.",
    ),
    max_exploded: int = typer.Option(
        0,
        "--max-exploded",
        "-e",
        help="Maximum allowed number of exploded timebombs left.",
    ),
    lookahead: int = typer.Option(
        0,
        "--lookahead",
        "-l",
        help="Days in the future at which to check the bomb state.",
    ),
    timezone: Optional[str] = typer.Option(
        None,
        "--timezone",
        "-t",
        help="Timezone (e.g. Europe/Amsterdam) used to determine bomb state.",
    ),
):
    mod, reg = module.split(":")
    reg = getattr(import_module(mod), reg)
    at = datetime.now()
    if timezone:
        at = pytz.UTC.localize(datetime.utcnow()).astimezone(
            pytz.timezone(timezone)
        )

    at = at + timedelta(lookahead)

    disarmed, armed, exploded = 0, 0, 0
    if not skip_disarmed:
        disarmed = len(list(reg.bombs_disarmed(at)))
    if not skip_armed:
        armed = len(list(reg.bombs_armed(at)))
    if not skip_exploded:
        exploded = len(list(reg.bombs_exploded(at)))

    total = disarmed + armed + exploded

    if disarmed > max_disarmed or armed > max_armed or exploded > max_exploded:
        sys.exit(total)

    return sys.exit(0)


if __name__ == "__main__":
    typer.run(main)
