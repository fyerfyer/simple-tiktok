package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/IBM/sarama"
	"github.com/go-kratos/kratos/v2/log"
)

// Producer Kafka生产者接口
type Producer interface {
	SendMessage(ctx context.Context, topic string, message *BaseMessage) error
	SendMessageWithKey(ctx context.Context, topic, key string, message *BaseMessage) error
	Close() error
}

// KafkaProducer Kafka生产者实现
type KafkaProducer struct {
	producer sarama.SyncProducer
	log      *log.Helper
}

// ProducerConfig 生产者配置
type ProducerConfig struct {
	Brokers         []string
	RetryMax        int
	BatchSize       int
	LingerMs        int
	CompressionType string
}

// NewKafkaProducer 创建Kafka生产者
func NewKafkaProducer(config *ProducerConfig, logger log.Logger) (*KafkaProducer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Return.Errors = true
	saramaConfig.Producer.Retry.Max = config.RetryMax
	saramaConfig.Producer.Flush.Bytes = config.BatchSize
	saramaConfig.Producer.Flush.Frequency = time.Duration(config.LingerMs) * time.Millisecond

	// 设置压缩类型
	switch config.CompressionType {
	case "gzip":
		saramaConfig.Producer.Compression = sarama.CompressionGZIP
	case "snappy":
		saramaConfig.Producer.Compression = sarama.CompressionSnappy
	case "lz4":
		saramaConfig.Producer.Compression = sarama.CompressionLZ4
	case "zstd":
		saramaConfig.Producer.Compression = sarama.CompressionZSTD
	default:
		saramaConfig.Producer.Compression = sarama.CompressionNone
	}

	producer, err := sarama.NewSyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	return &KafkaProducer{
		producer: producer,
		log:      log.NewHelper(logger),
	}, nil
}

// SendMessage 发送消息
func (p *KafkaProducer) SendMessage(ctx context.Context, topic string, message *BaseMessage) error {
	return p.SendMessageWithKey(ctx, topic, "", message)
}

// SendMessageWithKey 带key发送消息
func (p *KafkaProducer) SendMessageWithKey(ctx context.Context, topic, key string, message *BaseMessage) error {
	data, err := message.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(data),
	}

	if key != "" {
		msg.Key = sarama.StringEncoder(key)
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.log.WithContext(ctx).Errorf("failed to send message to kafka: %v", err)
		return fmt.Errorf("failed to send message: %w", err)
	}

	p.log.WithContext(ctx).Debugf("message sent successfully to partition %d, offset %d", partition, offset)
	return nil
}

// Close 关闭生产者
func (p *KafkaProducer) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}
	return nil
}
