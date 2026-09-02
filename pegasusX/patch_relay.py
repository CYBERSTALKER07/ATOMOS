import re

with open("apps/backend-go/outbox/relay.go", "r") as f:
    content = f.read()

pattern = re.compile(r'func \(r \*Relay\) publishWithRetry\(ctx context\.Context, e Event\) error \{\n\ttopics := events\.RelayPublishTopics\(e\.TopicName, e\.Payload\)\n\tvar lastErr error\n\tfor attempt := 1; attempt <= r\.cfg\.MaxPublishTries; attempt\+\+ \{\n\t\tif err := ctx\.Err\(\); err != nil \{\n\t\t\treturn err\n\t\t\}\n\t\tattemptErr := error\(nil\)\n\t\tfor _, topic := range topics \{\n\t\t\tpubCtx, cancel := context\.WithTimeout\(ctx, r\.cfg\.PublishTimeout\)\n\t\t\terr := publishOutboxEvent\(pubCtx, r\.publisher, topic, \[\]byte\(e\.AggregateID\), e\)\n\t\t\tcancel\(\)\n\t\t\tif err != nil \{\n\t\t\t\tattemptErr = err\n\t\t\t\tbreak\n\t\t\t\}\n\t\t\}')

replacement = r"""func (r *Relay) publishWithRetry(ctx context.Context, e Event) error {
	topics := events.RelayPublishTopics(e.TopicName, e.Payload)
	var lastErr error
	published := make(map[string]bool)
	for attempt := 1; attempt <= r.cfg.MaxPublishTries; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		attemptErr := error(nil)
		for _, topic := range topics {
			if published[topic] {
				continue
			}
			pubCtx, cancel := context.WithTimeout(ctx, r.cfg.PublishTimeout)
			err := publishOutboxEvent(pubCtx, r.publisher, topic, []byte(e.AggregateID), e)
			cancel()
			if err != nil {
				attemptErr = err
				break
			}
			published[topic] = true
		}"""

content = pattern.sub(replacement, content)

with open("apps/backend-go/outbox/relay.go", "w") as f:
    f.write(content)

