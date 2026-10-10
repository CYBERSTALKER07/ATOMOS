"""CLI entry that defaults to E2E fleet mode."""

from __future__ import annotations

from void_deep_agents.ecosystem_audit import main as ecosystem_main


def main(argv: list[str] | None = None) -> int:
    import sys

    args = list(sys.argv[1:] if argv is None else argv)
    if "--fleet" not in args:
        args = ["--fleet", *args]
    return ecosystem_main(args)


if __name__ == "__main__":
    raise SystemExit(main())
