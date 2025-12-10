package kafka

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/alarmistdev/status/check"
)

const (
	// numGoroutines is the number of background goroutines for ping check (producer and consumer).
	numGoroutines = 2
	// pingIntervalDivisor divides staleAfter to determine ping interval (pings at half the stale interval).
	pingIntervalDivisor = 2
)

// TopicsCheck creates a health check for Kafka topics listing.
func TopicsCheck(brokers []string, config check.Config) check.Check {
	return check.CheckFunc(func(ctx context.Context) error {
		kafkaConfig := sarama.NewConfig()
		kafkaConfig.Version = sarama.V4_0_0_0
		kafkaConfig.Net.DialTimeout = config.Timeout
		kafkaConfig.Net.ReadTimeout = config.Timeout
		kafkaConfig.Net.WriteTimeout = config.Timeout

		client, err := sarama.NewClient(brokers, kafkaConfig)
		if err != nil {
			return fmt.Errorf("failed to connect to kafka: %w", err)
		}
		defer client.Close()

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, err = client.Topics()
		if err != nil {
			return fmt.Errorf("failed to list kafka topics: %w", err)
		}

		return nil
	})
}

// pingCheck implements a Kafka health check that continuously produces and consumes
// ping messages in the background, and verifies freshness on each Check() call.
type pingCheck struct {
	store             PingStore
	staleAfter        time.Duration
	producer          sarama.SyncProducer
	consumer          sarama.Consumer
	partitionConsumer sarama.PartitionConsumer
	topic             string
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

// newKafkaConfig creates a Kafka configuration with the provided timeout settings.
func newKafkaConfig(config check.Config) *sarama.Config {
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Version = sarama.V4_0_0_0
	kafkaConfig.Net.DialTimeout = config.Timeout
	kafkaConfig.Net.ReadTimeout = config.Timeout
	kafkaConfig.Net.WriteTimeout = config.Timeout
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Consumer.Return.Errors = true

	return kafkaConfig
}

// setupConsumer creates and configures a Kafka consumer and partition consumer.
func setupConsumer(
	brokers []string,
	topic string,
	kafkaConfig *sarama.Config,
) (sarama.Consumer, sarama.PartitionConsumer, error) {
	consumer, err := sarama.NewConsumer(brokers, kafkaConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	partitions, err := consumer.Partitions(topic)
	if err != nil {
		consumer.Close()

		return nil, nil, fmt.Errorf("failed to list partitions for topic %s: %w", topic, err)
	}
	if len(partitions) == 0 {
		consumer.Close()

		return nil, nil, fmt.Errorf("no partitions found for topic %s", topic)
	}

	partition := partitions[0]
	partitionConsumer, err := consumer.ConsumePartition(topic, partition, sarama.OffsetNewest)
	if err != nil {
		consumer.Close()

		return nil, nil, fmt.Errorf("failed to start consuming partition %d: %w", partition, err)
	}

	return consumer, partitionConsumer, nil
}

// setupProducer creates a Kafka sync producer.
func setupProducer(brokers []string, kafkaConfig *sarama.Config) (sarama.SyncProducer, error) {
	producer, err := sarama.NewSyncProducer(brokers, kafkaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	return producer, nil
}

// PingCheck creates a health check for Kafka that continuously produces and consumes ping messages.
func PingCheck(
	brokers []string,
	topic string,
	store PingStore,
	staleAfter time.Duration,
	config check.Config,
) (check.Check, error) {
	if store == nil {
		store = NewInMemoryPingStore()
	}

	ctx, cancel := context.WithCancel(context.Background())
	pc := &pingCheck{
		store:      store,
		staleAfter: staleAfter,
		topic:      topic,
		ctx:        ctx,
		cancel:     cancel,
	}

	kafkaConfig := newKafkaConfig(config)

	consumer, partitionConsumer, err := setupConsumer(brokers, topic, kafkaConfig)
	if err != nil {
		cancel()

		return nil, err
	}
	pc.consumer = consumer
	pc.partitionConsumer = partitionConsumer

	producer, err := setupProducer(brokers, kafkaConfig)
	if err != nil {
		partitionConsumer.Close()
		consumer.Close()
		cancel()

		return nil, err
	}
	pc.producer = producer

	pc.wg.Add(numGoroutines)
	go pc.produceLoop()
	go pc.consumeLoop()

	return pc, nil
}

// produceLoop continuously produces ping messages at regular intervals.
func (pc *pingCheck) produceLoop() {
	defer pc.wg.Done()

	// Send pings at half the stale interval to ensure freshness
	pingInterval := pc.staleAfter / pingIntervalDivisor
	if pingInterval < time.Second {
		pingInterval = time.Second
	}

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()

	pc.sendPing()

	for {
		select {
		case <-pc.ctx.Done():
			return
		case <-ticker.C:
			pc.sendPing()
		}
	}
}

// sendPing sends a single ping message.
func (pc *pingCheck) sendPing() {
	_, _, err := pc.producer.SendMessage(&sarama.ProducerMessage{
		Topic:     pc.topic,
		Value:     sarama.StringEncoder("ping"),
		Timestamp: time.Now().UTC(),
	})
	if err != nil {
		slog.Error("failed to send kafka message", "error", err)

		return
	}
}

// consumeLoop continuously consumes messages and updates the store.
func (pc *pingCheck) consumeLoop() {
	defer pc.wg.Done()

	for {
		select {
		case <-pc.ctx.Done():
			return
		case err := <-pc.partitionConsumer.Errors():
			if err != nil {
				slog.Error("failed to consume kafka message", "error", err)

				continue
			}
		case msg := <-pc.partitionConsumer.Messages():
			if msg == nil {
				continue
			}
			_ = pc.store.SetProcessed(pc.ctx, msg.Timestamp)
		}
	}
}

// Check implements the check.Check interface.
func (pc *pingCheck) Check(ctx context.Context) error {
	last, err := pc.store.LastProcessed(ctx)
	if err != nil {
		return fmt.Errorf("failed to read ping store: %w", err)
	}
	if last.IsZero() {
		return errors.New("ping store has no recorded timestamp")
	}

	if time.Since(last) > pc.staleAfter {
		return fmt.Errorf("last ping is stale: age=%s stale_after=%s", time.Since(last), pc.staleAfter)
	}

	return nil
}

// Close stops background goroutines and closes Kafka clients.
func (pc *pingCheck) Close() error {
	pc.cancel()
	pc.wg.Wait()

	var errs []error

	if pc.partitionConsumer != nil {
		if err := pc.partitionConsumer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if pc.consumer != nil {
		if err := pc.consumer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if pc.producer != nil {
		if err := pc.producer.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing ping check: %v", errs)
	}

	return nil
}
