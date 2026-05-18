from __future__ import annotations

from typing import Iterable


class BidirectionalIndexMap:
    """Deterministic UUID <-> index mapping for a single solver run."""

    def __init__(self, ordered_uuids: Iterable[str]) -> None:
        self._uuid_to_index: dict[str, int] = {}
        self._index_to_uuid: list[str] = []

        for raw_uuid in ordered_uuids:
            uuid = raw_uuid.strip()
            if not uuid:
                continue
            if uuid in self._uuid_to_index:
                raise ValueError(f"duplicate uuid in mapping input: {uuid}")
            self._uuid_to_index[uuid] = len(self._index_to_uuid)
            self._index_to_uuid.append(uuid)

    @property
    def size(self) -> int:
        return len(self._index_to_uuid)

    def index_of(self, uuid: str) -> int:
        return self._uuid_to_index[uuid]

    def uuid_of(self, index: int) -> str:
        return self._index_to_uuid[index]

    def ordered(self) -> list[str]:
        return list(self._index_to_uuid)

    def as_index_map(self) -> dict[str, int]:
        return dict(self._uuid_to_index)
