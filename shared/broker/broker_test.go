package broker

import (
	"errors"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

type fakeChannel struct {
	calls     []string
	failAt    string
	queueArgs amqp.Table
	bindings  []string
	exchanges []string
	queues    []string
}

func (f *fakeChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	f.calls = append(f.calls, "exchange:"+name)
	f.exchanges = append(f.exchanges, name)
	return f.fail("exchange:" + name)
}

func (f *fakeChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	f.calls = append(f.calls, "queue:"+name)
	f.queues = append(f.queues, name)
	if args != nil {
		f.queueArgs = args
	}
	return amqp.Queue{Name: name}, f.fail("queue:" + name)
}

func (f *fakeChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	f.calls = append(f.calls, "bind:"+name+"->"+key)
	f.bindings = append(f.bindings, name+"->"+key)
	return f.fail("bind:" + name)
}

func (f *fakeChannel) fail(stage string) error {
	if f.failAt == stage {
		return errors.New("fail")
	}
	return nil
}

func TestDeclareTopology(t *testing.T) {
	ch := &fakeChannel{}
	require.NoError(t, DeclareTopology(ch))
	require.Equal(t, []string{
		"exchange:tasks.dlx",
		"queue:task.events.dlq",
		"bind:task.events.dlq->",
		"exchange:tasks",
		"queue:task.events",
		"bind:task.events->task.created",
	}, ch.calls)
	require.Equal(t, amqp.Table{"x-dead-letter-exchange": "tasks.dlx"}, ch.queueArgs)
}

func TestDeclareTopologyPropagatesErrors(t *testing.T) {
	for _, stage := range []string{
		"exchange:tasks.dlx",
		"queue:task.events.dlq",
		"bind:task.events.dlq",
		"exchange:tasks",
		"queue:task.events",
		"bind:task.events",
	} {
		t.Run(stage, func(t *testing.T) {
			require.Error(t, DeclareTopology(&fakeChannel{failAt: stage}))
		})
	}
}
