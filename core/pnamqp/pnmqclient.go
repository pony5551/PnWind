package pnamqp

import (
	log "github.com/pony5551/PnWind/core/pnlogrus"
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// 改进的发布函数 - 修复原代码中的时间单位错误和状态检查
func publish(done chan struct{}, mqConfig MQConfig) {
	mqServer := fmt.Sprintf("amqp://%s:%s@%s/%s", mqConfig.User, mqConfig.Pass, mqConfig.Server, mqConfig.Vhost)
	log.Infof("Publisher connecting to: %s", mqServer)
	queue := NewPnMQClient("publish", mqServer, mqConfig.QueueName, mqConfig.ExchangeName, "topic", mqConfig.Topic)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*20))
	defer cancel()

	// 等待连接建立
	for i := 0; i < 30; i++ {
		if queue.GetIsReady() {
			log.Infoln("Publisher connected successfully")
			break
		}
		time.Sleep(time.Second)
	}

	if !queue.GetIsReady() {
		log.Errorln("Publisher failed to establish connection within timeout")
		close(done)
		return
	}

	defer func() {
		if err := queue.Close(); err != nil {
			log.Errorf("Publisher close failed: %v", err)
		}
		close(done)
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !queue.GetIsReady() {
				log.Warnln("Publisher not ready, skipping...")
				continue
			}

			messageStr := fmt.Sprintf("msg%d", time.Now().UnixMilli())
			if err := queue.Push([]byte(messageStr)); err != nil {
				log.Errorf("push failed: %v", err)
			} else {
				log.Infof("push succeeded: %s", messageStr)
			}
		case <-ctx.Done():
			log.Infoln("Publisher context cancelled")
			return
		}
	}
}

// 改进的消费函数 - 修复channel关闭处理和重连逻辑
func consume(done chan struct{}, mqConfig MQConfig) {
	mqServer := fmt.Sprintf("amqp://%s:%s@%s/%s", mqConfig.User, mqConfig.Pass, mqConfig.Server, mqConfig.Vhost)
	log.Infof("Consumer connecting to: %s", mqServer)
	queue := NewPnMQClient("consume", mqServer, mqConfig.QueueName, mqConfig.ExchangeName, "topic", mqConfig.Topic)

	defer func() {
		if err := queue.Close(); err != nil {
			log.Errorf("Consumer close failed: %v", err)
		}
		close(done)
	}()

	// 等待连接建立 - 原代码只等待1秒可能不够
	for i := 0; i < 30; i++ {
		if queue.GetIsReady() {
			log.Infoln("Consumer connected successfully")
			break
		}
		time.Sleep(time.Second)
	}

	if !queue.GetIsReady() {
		log.Errorln("Consumer failed to establish connection within timeout")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*25)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			log.Infoln("Consumer context cancelled")
			return
		default:
		}

		if !queue.GetIsReady() {
			log.Warnln("Queue not ready, waiting...")
			time.Sleep(2 * time.Second)
			continue
		}

		deliveries, err := queue.Consume()
		if err != nil {
			log.Errorf("could not start consuming: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Infoln("Started consuming messages...")

		if !queue.consumeMessages(ctx, deliveries) {
			return
		}

		// 如果上下文已取消，退出主循环
		if ctx.Err() != nil {
			return
		}

		// 等待一段时间后重新尝试消费，避免快速重试
		log.Infoln("Reconnecting consumer...")
		time.Sleep(2 * time.Second)
	}
}

// PnMQClient - 改进的客户端结构，增加原子操作和重试控制
type PnMQClient struct {
	Tag             string
	m               *sync.RWMutex // 使用读写锁提高性能
	QueueName       string
	Exchange        string
	ExchangeType    string
	RoutingKey      string
	Connection      *amqp.Connection
	Channel         *amqp.Channel
	Done            chan bool
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	notifyConfirm   chan amqp.Confirmation
	isReady         int32 // 使用原子操作避免竞态条件

	// 添加重试计数控制
	retryCount    int32
	lastRetryTime time.Time
	retryMutex    sync.Mutex
}

const (
	// 增加重连延迟，避免过于频繁的重试
	reconnectDelay = 10 * time.Second // 原来是5秒，增加到10秒
	reInitDelay    = 5 * time.Second  // 原来是2秒，增加到5秒
	resendDelay    = 5 * time.Second
	pushTimeout    = 10 * time.Second

	// 添加重试次数限制
	maxRetries         = 3
	retryResetInterval = 60 * time.Second
)

var (
	errNotConnected  = errors.New("not connected to a server")
	errAlreadyClosed = errors.New("already closed: not connected to the server")
	errShutdown      = errors.New("client is shutting down")
)

// NewPnMQClient - 保持原来的接口，内部逻辑改进
func NewPnMQClient(tag string, addr, queueName, exchangeName, exchangeType, routingKey string) *PnMQClient {
	client := PnMQClient{
		Tag:           tag,
		m:             &sync.RWMutex{},
		QueueName:     queueName,
		Exchange:      exchangeName,
		ExchangeType:  exchangeType,
		RoutingKey:    routingKey,
		Done:          make(chan bool),
		lastRetryTime: time.Now(),
	}
	go client.handleReconnect(addr)
	return &client
}

func (client *PnMQClient) GetIsReady() bool {
	return atomic.LoadInt32(&client.isReady) == 1
}

func (client *PnMQClient) setReady(ready bool) {
	if ready {
		atomic.StoreInt32(&client.isReady, 1)
	} else {
		atomic.StoreInt32(&client.isReady, 0)
	}
}

// 添加重试控制机制
func (client *PnMQClient) shouldRetry() bool {
	client.retryMutex.Lock()
	defer client.retryMutex.Unlock()

	now := time.Now()
	if now.Sub(client.lastRetryTime) > retryResetInterval {
		atomic.StoreInt32(&client.retryCount, 0)
		client.lastRetryTime = now
	}

	currentRetries := atomic.LoadInt32(&client.retryCount)
	if currentRetries >= maxRetries {
		log.Warnf("Max retries (%d) reached, waiting for reset interval", maxRetries)
		return false
	}

	atomic.AddInt32(&client.retryCount, 1)
	return true
}

// consumeMessages - 提取消息消费逻辑，避免代码重复
func (client *PnMQClient) consumeMessages(ctx context.Context, deliveries <-chan amqp.Delivery) bool {
	chClosedCh := make(chan *amqp.Error, 1)
	if client.Channel != nil {
		client.Channel.NotifyClose(chClosedCh)
	}

	for {
		select {
		case <-ctx.Done():
			return false

		case amqErr := <-chClosedCh:
			// 改进：更好的错误处理，避免nil错误和快速重试
			if amqErr != nil {
				log.Errorf("AMQP Channel closed due to: %v", amqErr)
			} else {
				log.Warnln("AMQP Channel closed")
			}
			// 跳出内层循环，等待重新连接而不是立即重试
			return true

		case delivery, ok := <-deliveries:
			if !ok {
				log.Warnln("Deliveries channel closed")
				return true
			}

			// 修复原代码：处理时间从10秒改为合理的100毫秒
			log.Infof("received message: %s", delivery.Body)
			if err := delivery.Ack(false); err != nil {
				log.Errorf("error acknowledging message: %v", err)
			}
			// 使用定时器代替阻塞式等待，避免内存泄漏
			timer := time.NewTimer(100 * time.Millisecond)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				return false
			}
			timer.Stop()
		}
	}
}

// handleReconnect - 改进重连逻辑，添加重试限制
func (client *PnMQClient) handleReconnect(addr string) {
	for {
		select {
		case <-client.Done:
			return
		default:
		}

		client.setReady(false)

		// 检查是否应该重试
		if !client.shouldRetry() {
			select {
			case <-client.Done:
				return
			case <-time.After(retryResetInterval):
				continue
			}
		}

		log.Infof("[%s] attempting to connect", client.Tag)
		conn, err := client.connect(addr)
		if err != nil {
			log.Errorf("failed to connect. Retrying in %v: %v", reconnectDelay, err)
			select {
			case <-client.Done:
				return
			case <-time.After(reconnectDelay):
			}
			continue
		}

		log.Infof("[%s] connected successfully", client.Tag)
		// 重置重试计数
		atomic.StoreInt32(&client.retryCount, 0)

		if done := client.handleReInit(conn); done {
			break
		}
	}
}

// connect - 改进连接配置，添加心跳机制
func (client *PnMQClient) connect(addr string) (*amqp.Connection, error) {
	// 添加连接配置，增加稳定性
	config := amqp.Config{
		Heartbeat: 10 * time.Second,
		Locale:    "en_US",
	}

	conn, err := amqp.DialConfig(addr, config)
	if err != nil {
		return nil, err
	}

	client.changeConnection(conn)
	return conn, nil
}

// handleReInit - 改进初始化逻辑
func (client *PnMQClient) handleReInit(conn *amqp.Connection) bool {
	for {
		select {
		case <-client.Done:
			return true
		default:
		}

		client.setReady(false)

		err := client.init(conn)
		if err != nil {
			log.Errorf("failed to initialize channel, retrying in %v: %v", reInitDelay, err)
			select {
			case <-client.Done:
				return true
			case <-client.notifyConnClose:
				log.Infof("[%s] connection closed, reconnecting...", client.Tag)
				return false
			case <-time.After(reInitDelay):
			}
			continue
		}

		select {
		case <-client.Done:
			return true
		case <-client.notifyConnClose:
			log.Infof("[%s] connection closed, reconnecting...", client.Tag)
			return false
		case <-client.notifyChanClose:
			log.Println("channel closed, re-running init...")
		}
	}
}

// init - 改进初始化逻辑，增加错误处理
func (client *PnMQClient) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	if err != nil {
		return err
	}

	err = ch.Confirm(false)
	if err != nil {
		ch.Close()
		return err
	}

	// 设置QoS
	err = ch.Qos(1, 0, false)
	if err != nil {
		ch.Close()
		return err
	}

	if client.Exchange != "" {
		err = ch.ExchangeDeclare(
			client.Exchange,
			client.ExchangeType,
			true,  // durable
			false, // auto-deleted
			false, // internal
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			ch.Close()
			return err
		}

		_, err = ch.QueueDeclare(
			client.QueueName,
			true,  // Durable
			false, // Delete when unused
			false, // Exclusive
			false, // No-wait
			nil,   // Arguments
		)
		if err != nil {
			ch.Close()
			return err
		}

		err = ch.QueueBind(
			client.QueueName,
			client.RoutingKey,
			client.Exchange,
			false,
			nil,
		)
		if err != nil {
			ch.Close()
			return err
		}
	} else {
		_, err = ch.QueueDeclare(
			client.QueueName,
			true,  // Durable
			false, // Delete when unused
			false, // Exclusive
			false, // No-wait
			nil,   // Arguments
		)
		if err != nil {
			ch.Close()
			return err
		}
	}

	client.changeChannel(ch)
	client.setReady(true)
	log.Infof("[%s] client init done", client.Tag)

	return nil
}

// changeConnection - 改进连接管理
func (client *PnMQClient) changeConnection(connection *amqp.Connection) {
	client.m.Lock()
	defer client.m.Unlock()

	client.Connection = connection
	client.notifyConnClose = make(chan *amqp.Error, 1)
	client.Connection.NotifyClose(client.notifyConnClose)
}

// changeChannel - 改进通道管理
func (client *PnMQClient) changeChannel(channel *amqp.Channel) {
	client.m.Lock()
	defer client.m.Unlock()

	client.Channel = channel
	client.notifyChanClose = make(chan *amqp.Error, 1)
	client.notifyConfirm = make(chan amqp.Confirmation, 10)
	client.Channel.NotifyClose(client.notifyChanClose)
	client.Channel.NotifyPublish(client.notifyConfirm)
}

// Push - 改进推送逻辑，增加超时和确认处理
func (client *PnMQClient) Push(data []byte) error {
	if !client.GetIsReady() {
		return errNotConnected
	}

	ticker := time.NewTicker(resendDelay)
	defer ticker.Stop()

	for {
		err := client.UnsafePush(data)
		if err != nil {
			log.Errorf("push failed. Retrying: %v", err)
			select {
			case <-client.Done:
				return errShutdown
			case <-ticker.C:
			}
			continue
		}

		select {
		case confirm := <-client.notifyConfirm:
			if confirm.Ack {
				log.Debugf("push confirmed [%d]", confirm.DeliveryTag)
				return nil
			}
			log.Warnf("push not acknowledged [%d]", confirm.DeliveryTag)
		case <-time.After(pushTimeout):
			log.Warnln("push confirmation timeout")
			return errors.New("push confirmation timeout")
		}
	}
}

// UnsafePush - 改进不安全推送，增加消息持久化
func (client *PnMQClient) UnsafePush(data []byte) error {
	if !client.GetIsReady() {
		return errNotConnected
	}

	client.m.RLock()
	channel := client.Channel
	client.m.RUnlock()

	if channel == nil {
		return errNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), pushTimeout)
	defer cancel()

	if client.Exchange != "" {
		return channel.PublishWithContext(
			ctx,
			client.Exchange,
			client.RoutingKey,
			false, // Mandatory
			false, // Immediate
			amqp.Publishing{
				ContentType:     "text/plain",
				Body:            data,
				ContentEncoding: "utf-8",
				Timestamp:       time.Now(),
				DeliveryMode:    amqp.Persistent, // Persistent=持久化消息  Transient=非持久
			},
		)
	} else {
		return channel.PublishWithContext(
			ctx,
			"",
			client.QueueName,
			false, // Mandatory
			false, // Immediate
			amqp.Publishing{
				ContentType:     "text/plain",
				Body:            data,
				ContentEncoding: "utf-8",
				Timestamp:       time.Now(),
				DeliveryMode:    amqp.Persistent, // Persistent=持久化消息  Transient=非持久
			},
		)
	}
}

// Consume - 改进消费逻辑，增加连接检查
func (client *PnMQClient) Consume() (<-chan amqp.Delivery, error) {
	if !client.GetIsReady() {
		return nil, errNotConnected
	}

	client.m.RLock()
	channel := client.Channel
	client.m.RUnlock()

	if channel == nil {
		return nil, errNotConnected
	}

	// QoS设置已经在init中处理，这里不再重复设置
	return channel.Consume(
		client.QueueName,
		"",    // Consumer
		false, // Auto-Ack
		false, // Exclusive
		false, // No-local
		false, // No-Wait
		nil,   // Args
	)
}

// Close - 改进关闭逻辑
func (client *PnMQClient) Close() error {
	if !client.GetIsReady() {
		return errAlreadyClosed
	}

	client.m.Lock()
	defer client.m.Unlock()

	close(client.Done)

	var err error
	if client.Channel != nil {
		if e := client.Channel.Close(); e != nil {
			err = e
		}
	}

	if client.Connection != nil {
		if e := client.Connection.Close(); e != nil && err == nil {
			err = e
		}
	}

	client.setReady(false)
	return err
}

type MQConfig struct {
	User         string `json:"user"`
	Pass         string `json:"pass"`
	Server       string `json:"server"`
	Vhost        string `json:"vhost"`
	ExchangeName string `json:"exchangename"`
	QueueName    string `json:"queuename"`
	Topic        string `json:"topic"`
}
