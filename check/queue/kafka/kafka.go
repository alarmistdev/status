package kafka

import (
	"context"
	"fmt"

	"github.com/IBM/sarama"
	"github.com/alarmistdev/status/check"
)

// Check creates a health check for Kafka.
func Check(brokers []string, config check.Config) check.Check {
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
