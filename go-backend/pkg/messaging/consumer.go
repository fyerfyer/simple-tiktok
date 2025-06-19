package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/go-kratos/kratos/v2/log"
)

// MessageHandler 消息处理器
type MessageHandler func(ctx context.Context, message *BaseMessage) error

// Consumer Kafka消费者接口
type Consumer interface {
	Subscribe(topic string, handler MessageHandler) error
	Start(ctx context.Context) error
	Stop() error
}

// KafkaConsumer Kafka消费者实现
type KafkaConsumer struct {
	consumerGroup sarama.ConsumerGroup
	handlers      map[string]MessageHandler
	log           *log.Helper
	wg            sync.WaitGroup
	cancel        context.CancelFunc
}

// ConsumerConfig 消费者配置
type ConsumerConfig struct {
	Brokers        []string
	GroupID        string
	AutoCommit     bool
	SessionTimeout time.Duration
	FetchMinBytes  int32
	FetchMaxWait   time.Duration
}

// NewKafkaConsumer 创建Kafka消费者
func NewKafkaConsumer(config *ConsumerConfig, logger log.Logger) (*KafkaConsumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Consumer.Group.Session.Timeout = config.SessionTimeout
	saramaConfig.Consumer.Group.Heartbeat.Interval = config.SessionTimeout / 3
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaConfig.Consumer.Fetch.Min = config.FetchMinBytes
	saramaConfig.Consumer.Fetch.Max = int32(config.FetchMaxWait)

	if config.AutoCommit {
		saramaConfig.Consumer.Offsets.AutoCommit.Enable = true
		saramaConfig.Consumer.Offsets.AutoCommit.Interval = time.Second
	}

	consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, saramaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka consumer group: %w", err)
	}

	return &KafkaConsumer{
		consumerGroup: consumerGroup,
		handlers:      make(map[string]MessageHandler),
		log:           log.NewHelper(logger),
	}, nil
}

// Subscribe 订阅主题
func (c *KafkaConsumer) Subscribe(topic string, handler MessageHandler) error {
	c.handlers[topic] = handler
	return nil
}

// Start 启动消费者
func (c *KafkaConsumer) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	topics := make([]string, 0, len(c.handlers))
	for topic := range c.handlers {
		topics = append(topics, topic)
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := c.consumerGroup.Consume(ctx, topics, c); err != nil {
					c.log.Errorf("failed to consume messages: %v", err)
					time.Sleep(time.Second)
				}
			}
		}
	}()

	c.log.Info("kafka consumer started")
	return nil
}

// Stop 停止消费者
func (c *KafkaConsumer) Stop() error {
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()

	if c.consumerGroup != nil {
		return c.consumerGroup.Close()
	}
	return nil
}

// Setup 消费者组设置
func (c *KafkaConsumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup 消费者组清理
func (c *KafkaConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 消费消息
func (c *KafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}

			handler, exists := c.handlers[message.Topic]
			if !exists {
				c.log.Warnf("no handler for topic: %s", message.Topic)
				session.MarkMessage(message, "")
				continue
			}

			// 解析消息
			baseMessage := &BaseMessage{}
			if err := baseMessage.FromJSON(message.Value); err != nil {
				c.log.Errorf("failed to parse message: %v", err)
				session.MarkMessage(message, "")
				continue
			}

			// 处理消息
			ctx := context.Background()
			if err := handler(ctx, baseMessage); err != nil {
				c.log.Errorf("failed to handle message: %v", err)
				// 根据业务需求决定是否重试或跳过
				continue
			}

			session.MarkMessage(message, "")

		case <-session.Context().Done():
			return nil
		}
	}
}
