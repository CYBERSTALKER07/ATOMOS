package connectors

import (
	"context"

	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner/adapters/onec"
)

func collectFullCommerceMLStreams(ctx context.Context, c *OnecConnector, rawXML []byte, out *PullResult) error {
	if _, err := onec.ParseCommerceMLStreams(rawXML, func(streamName string, _ []string, chunk []parser.ParsedRow) error {
		canonicalStream := mapCommerceMLStreamName(streamName)
		if canonicalStream == "" {
			return nil
		}
		rows := make([]map[string]any, 0, len(chunk))
		for _, pr := range chunk {
			rows = append(rows, pr.Fields)
		}
		out.Streams[canonicalStream] = append(out.Streams[canonicalStream], rows...)
		out.StreamCounts[canonicalStream] += int64(len(rows))
		out.RawCount += int64(len(rows))
		mapped, mapErr := c.MapSchema(ctx, canonicalStream, rows)
		if mapErr != nil {
			return mapErr
		}
		out.Canonical = append(out.Canonical, mapped...)
		return nil
	}); err != nil {
		return err
	}
	out.RecordsCount = int64(len(out.Canonical))
	return nil
}
