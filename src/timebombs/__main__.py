import sys
from datetime import datetime
from importlib import import_module
from typing import Annotated, NoReturn

import pytz
import typer

from . import Registry


def main(
    module: str = typer.Argument(
        ...,
        help="Reference to the registry variable for collecting timebombs (ex. `module.submodule:variable`).",  # noqa
    ),
    include_disarmed: Annotated[
        bool,
        typer.Option(
            "--include-disarmed", "-d", help="Considers disarmed timebombs."
        ),
    ] = False,
    skip_armed: Annotated[
        bool, typer.Option("--skip-armed", "-a", help="Skips armed timebombs.")
    ] = False,
    skip_exploded: Annotated[
        bool,
        typer.Option(
            "--skip-exploded", "-e", help="Skips exploded timebombs."
        ),
    ] = False,
    max_disarmed: Annotated[
        int,
        typer.Option(
            "--max-disarmed",
            "-D",
            help="Maximum allowed number of disarmed timebombs left.",
        ),
    ] = 0,
    max_armed: Annotated[
        int,
        typer.Option(
            "--max-armed",
            "-A",
            help="Maximum allowed number of armed timebombs left.",
        ),
    ] = 0,
    max_exploded: Annotated[
        int,
        typer.Option(
            "--max-exploded",
            "-E",
            help="Maximum allowed number of exploded timebombs left.",
        ),
    ] = 0,
    at: Annotated[
        datetime,
        typer.Option(
            "-t",
            "--at-time",
            help="The moment at which to check for timebombs state.",
            show_default=f"output of datetime.now() (e.g. {datetime.now()}",
        ),
    ] = datetime.now(),
    timezone: Annotated[
        str | None,
        typer.Option(
            "--timezone",
            "-t",
            help="Timezone (e.g. Europe/Amsterdam) used to determine bomb state.",  # noqa
        ),
    ] = None,
) -> NoReturn:
    mod, reg = module.split(":")
    registry: Registry = getattr(import_module(mod), reg)
    if timezone:
        local_tz = datetime.now().astimezone().tzinfo
        at.replace(tzinfo=local_tz)
        at = at.astimezone(pytz.timezone(timezone))

    disarmed, armed, exploded, total = 0, 0, 0, 0
    if include_disarmed:
        disarmed = len(list(registry.bombs_disarmed(at)))
        total += disarmed
    if not skip_armed:
        armed = len(list(registry.bombs_armed(at)))
        total += armed
    if not skip_exploded:
        exploded = len(list(registry.bombs_exploded(at)))
        total += exploded

    sys.exit(total)


if __name__ == "__main__":
    typer.run(main)
