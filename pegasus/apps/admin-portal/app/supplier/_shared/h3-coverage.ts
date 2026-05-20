import { uncompactCells } from 'h3-js';

type ResolveRenderableHexesOptions = {
	hexes?: string[];
	compactedHexes?: string[];
	resolution?: number;
};

export function resolveRenderableHexes({
	hexes,
	compactedHexes,
	resolution,
}: ResolveRenderableHexesOptions): string[] {
	const expandedHexes = Array.isArray(hexes) ? hexes : [];
	if (!Array.isArray(compactedHexes) || compactedHexes.length === 0 || typeof resolution !== 'number') {
		return expandedHexes;
	}

	try {
		return uncompactCells(compactedHexes, resolution);
	} catch {
		return expandedHexes;
	}
}