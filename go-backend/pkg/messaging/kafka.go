package messaging

import (
	"context"
	"fmt"

	"go-backend/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
)

// KafkaManager Kafka管理器
type KafkaManager struct {
	producer Producer
	consumer Consumer
	config   *conf.Data_Kafka
	log      *log.Helper
}

// NewKafkaManager 创建Kafka管理器
func NewKafkaManager(config *conf.Data_Kafka, logger log.Logger) (*KafkaManager, error) {
	producerConfig := &ProducerConfig{
		Brokers:         config.Brokers,
		RetryMax:        int(config.Producer.RetryMax),
		BatchSize:       int(config.Producer.BatchSize),
		LingerMs:        int(config.Producer.LingerMs),
		CompressionType: config.Producer.CompressionType,
	}

	producer, err := NewKafkaProducer(producerConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	consumerConfig := &ConsumerConfig{
		Brokers:        config.Brokers,
		GroupID:        config.Consumer.GroupId,
		AutoCommit:     config.Consumer.AutoCommit,
		SessionTimeout: config.Consumer.SessionTimeout.AsDuration(),
		FetchMinBytes:  config.Consumer.FetchMinBytes,
		FetchMaxWait:   config.Consumer.FetchMaxWait.AsDuration(),
	}

	consumer, err := NewKafkaConsumer(consumerConfig, logger)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	return &KafkaManager{
		producer: producer,
		consumer: consumer,
		config:   config,
		log:      log.NewHelper(logger),
	}, nil
}

// GetProducer 获取生产者
func (km *KafkaManager) GetProducer() Producer {
	return km.producer
}

// GetConsumer 获取消费者
func (km *KafkaManager) GetConsumer() Consumer {
	return km.consumer
}

// SendVideoUploadEvent 发送视频上传事件
func (km *KafkaManager) SendVideoUploadEvent(ctx context.Context, topic string, event *VideoUploadEvent) error {
	message := NewBaseMessage(VideoUploadMessage, event)
	return km.producer.SendMessage(ctx, topic, message)
}

// SendVideoProcessEvent 发送视频处理事件
func (km *KafkaManager) SendVideoProcessEvent(ctx context.Context, topic string, event *VideoProcessEvent) error {
	message := NewBaseMessage(VideoProcessMessage, event)
	return km.producer.SendMessage(ctx, topic, message)
}

// SendVideoStatsEvent 发送视频统计事件
func (km *KafkaManager) SendVideoStatsEvent(ctx context.Context, topic string, event *VideoStatsEvent) error {
	message := NewBaseMessage(VideoStatsMessage, event)
	return km.producer.SendMessage(ctx, topic, message)
}

// SendUserActionEvent 发送用户行为事件
func (km *KafkaManager) SendUserActionEvent(ctx context.Context, topic string, event *UserActionEvent) error {
	message := NewBaseMessage(UserActionMessage, event)
	return km.producer.SendMessage(ctx, topic, message)
}

// Close 关闭Kafka管理器
func (km *KafkaManager) Close() error {
	var err error
	if km.consumer != nil {
		if e := km.consumer.Stop(); e != nil {
			err = e
		}
	}
	if km.producer != nil {
		if e := km.producer.Close(); e != nil {
			err = e
		}
	}
	return err
}
