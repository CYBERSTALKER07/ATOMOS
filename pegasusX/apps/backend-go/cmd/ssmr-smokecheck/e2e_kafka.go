package main

import (
	"fmt"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

func assertDomainTopicDispatchMarker() error {
	if !events.ConsumeDomainTopics() {
		return nil
	}
	topics := events.DispatcherConsumerTopics()
	if len(topics) < 2 {
		return fmt.Errorf("domain topic dispatch: expected fan-in topics, got %v", topics)
	}
	fmt.Println("PX_E2E_DOMAIN_TOPIC_DISPATCH_OK")
	return nil
}
