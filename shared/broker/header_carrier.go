package broker

import (
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
)

// HeaderCarrier adapta headers AMQP ao propagador OpenTelemetry.
type HeaderCarrier amqp.Table

func (c HeaderCarrier) Get(key string) string {
	for header, value := range c {
		if strings.EqualFold(header, key) {
			text, _ := value.(string)
			return text
		}
	}
	return ""
}

func (c HeaderCarrier) Set(key, value string) { c[key] = value }

func (c HeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	return keys
}
