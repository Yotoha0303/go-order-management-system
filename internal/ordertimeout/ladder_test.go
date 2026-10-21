package ordertimeout

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-order-management-system/internal/model"

	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/gorm"
)

func TestMatchDelayLadder_ProductionLadders(t *testing.T) {
	tests := []struct {
		name        string
		delay       time.Duration
		wantKey     string
		wantQueue   string
		wantDur     time.Duration
	}{
		{
			name:      "negative delay matches smallest ladder (5m)",
			delay:     -10 * time.Second,
			wantKey:   "delay.5m",
			wantQueue: "order.timeout.delay.5m",
			wantDur:   5 * time.Minute,
		},
		{
			name:      "zero delay matches smallest ladder (5m)",
			delay:     0,
			wantKey:   "delay.5m",
			wantQueue: "order.timeout.delay.5m",
			wantDur:   5 * time.Minute,
		},
		{
			name:      "1 minute delay matches smallest ladder (5m)",
			delay:     1 * time.Minute,
			wantKey:   "delay.5m",
			wantQueue: "order.timeout.delay.5m",
			wantDur:   5 * time.Minute,
		},
		{
			name:      "exactly 5 minute delay matches 5m ladder",
			delay:     5 * time.Minute,
			wantKey:   "delay.5m",
			wantQueue: "order.timeout.delay.5m",
			wantDur:   5 * time.Minute,
		},
		{
			name:      "6 minute delay matches middle ladder (15m)",
			delay:     6 * time.Minute,
			wantKey:   "delay.15m",
			wantQueue: "order.timeout.delay.15m",
			wantDur:   15 * time.Minute,
		},
		{
			name:      "exactly 15 minute delay matches 15m ladder",
			delay:     15 * time.Minute,
			wantKey:   "delay.15m",
			wantQueue: "order.timeout.delay.15m",
			wantDur:   15 * time.Minute,
		},
		{
			name:      "16 minute delay matches largest ladder (30m)",
			delay:     16 * time.Minute,
			wantKey:   "delay.30m",
			wantQueue: "order.timeout.delay.30m",
			wantDur:   30 * time.Minute,
		},
		{
			name:      "exactly 30 minute delay matches 30m ladder",
			delay:     30 * time.Minute,
			wantKey:   "delay.30m",
			wantQueue: "order.timeout.delay.30m",
			wantDur:   30 * time.Minute,
		},
		{
			name:      "large 2 hour delay matches largest ladder (30m)",
			delay:     2 * time.Hour,
			wantKey:   "delay.30m",
			wantQueue: "order.timeout.delay.30m",
			wantDur:   30 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchDelayLadder(defaultProductionLadders, tt.delay)
			if got.RoutingKey != tt.wantKey {
				t.Errorf("matchDelayLadder(%v) RoutingKey = %q, want %q", tt.delay, got.RoutingKey, tt.wantKey)
			}
			if got.QueueName != tt.wantQueue {
				t.Errorf("matchDelayLadder(%v) QueueName = %q, want %q", tt.delay, got.QueueName, tt.wantQueue)
			}
			if got.Duration != tt.wantDur {
				t.Errorf("matchDelayLadder(%v) Duration = %v, want %v", tt.delay, got.Duration, tt.wantDur)
			}
		})
	}
}

func TestMatchDelayLadder_CustomTestLadders(t *testing.T) {
	customLadders := []DelayLadder{
		{Duration: 5 * time.Second, RoutingKey: "delay.5s", QueueName: "test.delay.5s"},
		{Duration: 15 * time.Second, RoutingKey: "delay.15s", QueueName: "test.delay.15s"},
		{Duration: 30 * time.Second, RoutingKey: "delay.30s", QueueName: "test.delay.30s"},
	}

	tests := []struct {
		name    string
		delay   time.Duration
		wantKey string
	}{
		{"2s matches 5s", 2 * time.Second, "delay.5s"},
		{"5s matches 5s", 5 * time.Second, "delay.5s"},
		{"6s matches 15s", 6 * time.Second, "delay.15s"},
		{"15s matches 15s", 15 * time.Second, "delay.15s"},
		{"20s matches 30s", 20 * time.Second, "delay.30s"},
		{"30s matches 30s", 30 * time.Second, "delay.30s"},
		{"90s matches 30s", 90 * time.Second, "delay.30s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchDelayLadder(customLadders, tt.delay)
			if got.RoutingKey != tt.wantKey {
				t.Errorf("matchDelayLadder(%v) = %q, want %q", tt.delay, got.RoutingKey, tt.wantKey)
			}
		})
	}
}

func TestMatchDelayLadder_EmptyLaddersFallback(t *testing.T) {
	got := matchDelayLadder(nil, 10*time.Minute)
	if got.RoutingKey != "delay.30m" {
		t.Errorf("expected fallback to delay.30m, got: %s", got.RoutingKey)
	}
}

func TestNewWorker_CustomLaddersSortingAndConfig(t *testing.T) {
	// 乱序传入自定义阶梯，验证 NewWorker 能够自动按 Duration 升序排序
	unsortedLadders := []DelayLadder{
		{Duration: 30 * time.Second, RoutingKey: "delay.30s", QueueName: "order.timeout.delay.30s"},
		{Duration: 5 * time.Second, RoutingKey: "delay.5s", QueueName: "order.timeout.delay.5s"},
		{Duration: 15 * time.Second, RoutingKey: "delay.15s", QueueName: "order.timeout.delay.15s"},
	}

	worker, err := NewWorker(Config{
		URL:                "amqp://guest:guest@localhost:5672/",
		ConnectTimeout:     time.Second,
		ReconnectDelay:     time.Second,
		OutboxPollInterval: time.Second,
		OutboxRetryDelay:   time.Second,
		PublishBatchSize:   10,
		ConsumerPrefetch:   1,
		DelayLadders:       unsortedLadders,
	}, &gorm.DB{}, &stubCanceller{}, slog.Default())
	if err != nil {
		t.Fatalf("NewWorker failed: %v", err)
	}

	if len(worker.topology.ladders) != 3 {
		t.Fatalf("expected 3 ladders, got: %d", len(worker.topology.ladders))
	}

	// 断言升序已正确重排
	if worker.topology.ladders[0].Duration != 5*time.Second ||
		worker.topology.ladders[1].Duration != 15*time.Second ||
		worker.topology.ladders[2].Duration != 30*time.Second {
		t.Errorf("ladders not properly sorted: %+v", worker.topology.ladders)
	}

	// 验证匹配正确
	matched := worker.matchDelayLadder(4 * time.Second)
	if matched.RoutingKey != "delay.5s" {
		t.Errorf("expected delay.5s, got: %s", matched.RoutingKey)
	}
}

func TestLadderTopology_QueueArgumentsValidation(t *testing.T) {
	for _, ladder := range defaultProductionLadders {
		args := buildLadderQueueArguments(ladder, productionTopology)

		if args["x-dead-letter-exchange"] != "order.timeout" {
			t.Errorf("invalid DLX for ladder %s: %v", ladder.QueueName, args["x-dead-letter-exchange"])
		}
		if args["x-dead-letter-routing-key"] != "cancel" {
			t.Errorf("invalid DLK for ladder %s: %v", ladder.QueueName, args["x-dead-letter-routing-key"])
		}

		ttlMs, ok := args["x-message-ttl"].(int64)
		if !ok {
			t.Fatalf("expected int64 for x-message-ttl, got: %T (%v)", args["x-message-ttl"], args["x-message-ttl"])
		}

		switch ladder.RoutingKey {
		case "delay.5m":
			if ttlMs != 300000 {
				t.Errorf("expected 300000ms for 5m, got: %d", ttlMs)
			}
		case "delay.15m":
			if ttlMs != 900000 {
				t.Errorf("expected 900000ms for 15m, got: %d", ttlMs)
			}
		case "delay.30m":
			if ttlMs != 1800000 {
				t.Errorf("expected 1800000ms for 30m, got: %d", ttlMs)
			}
		default:
			t.Errorf("unexpected routing key: %s", ladder.RoutingKey)
		}
	}
}

func TestInterleavedDelay_HeadOfLineBlocking_EliminationProof(t *testing.T) {
	// 证明数学定理：
	// 在单一共享队列模型下：Msg A (TTL=30s) 在 t=0 入队，Msg B (TTL=5s) 在 t=1 入队；
	// 因为 RabbitMQ 仅检查队头，Msg B 即使在 t=6 到期，也必须等待 Msg A 在 t=30 到期出队，产生 24 秒队头阻塞 (HoLB)！
	// 在阶梯队列模型下：Msg A 路由至 30s 阶梯队列，Msg B 路由至 5s 阶梯队列；
	// 5s 阶梯队列独立出队，Msg B 在 t=6s 准时到期，完全不受 Msg A 影响！

	type simulatedMessage struct {
		id         string
		enqueuedAt time.Duration
		ttl        time.Duration
	}

	msgA := simulatedMessage{id: "order-A-normal", enqueuedAt: 0 * time.Minute, ttl: 30 * time.Minute}
	msgB := simulatedMessage{id: "order-B-flash-sale", enqueuedAt: 1 * time.Minute, ttl: 5 * time.Minute}

	// 1. 单一队列仿真（复现 HoLB 缺陷）
	singleQueueHeadExpiresAt := msgA.enqueuedAt + msgA.ttl // 30m
	singleQueueMsgBExpiry := msgB.enqueuedAt + msgB.ttl    // 物理应在 6m 到期
	singleQueueMsgBActualDeparture := singleQueueMsgBExpiry
	if singleQueueMsgBActualDeparture < singleQueueHeadExpiresAt {
		// 被队头阻塞，必须等到队头 Msg A 离开后才被检查
		singleQueueMsgBActualDeparture = singleQueueHeadExpiresAt
	}
	if singleQueueMsgBActualDeparture != 30*time.Minute {
		t.Fatalf("expected single queue to suffer HoLB and delay Msg B to 30m, got: %v", singleQueueMsgBActualDeparture)
	}

	// 2. 阶梯分级队列（消除 HoLB 证明）
	worker := &Worker{
		topology: productionTopology,
	}

	ladderA := worker.matchDelayLadder(msgA.ttl)
	ladderB := worker.matchDelayLadder(msgB.ttl)

	if ladderA.QueueName == ladderB.QueueName {
		t.Fatalf("Ladder A and Ladder B must reside in distinct physical queues, got same: %s", ladderA.QueueName)
	}

	// 阶梯队列各自独立 FIFO
	ladderQueueMsgBActualDeparture := msgB.enqueuedAt + ladderB.Duration
	if ladderQueueMsgBActualDeparture > 6*time.Minute {
		// 5m 阶梯在 5m 时出队，绝不会被 30m 阶梯延误到 30m
		t.Fatalf("expected ladder queue Msg B to depart on its own 5m ladder, got: %v", ladderQueueMsgBActualDeparture)
	}
}

func TestPublishEvent_OmitsExpirationInLadderMode(t *testing.T) {
	now := time.Now().UTC()
	event := Event{
		EventID:   "evt-ladder-check",
		OrderID:   1001,
		UserID:    2002,
		TimeoutAt: now.Add(5 * time.Minute),
	}

	// 1. 阶梯模式验证：buildOrderTimeoutPublishing 必须严格将 Expiration 置空，消除单消息 TTL 与队头阻塞
	publishing, routingKey, err := buildOrderTimeoutPublishing(now, event, productionTopology)
	if err != nil {
		t.Fatalf("buildOrderTimeoutPublishing failed: %v", err)
	}
	if publishing.Expiration != "" {
		t.Errorf("expected empty Expiration in ladder mode, got: %q", publishing.Expiration)
	}
	if routingKey != "delay.5m" {
		t.Errorf("expected routingKey 'delay.5m', got: %q", routingKey)
	}
	if publishing.MessageId != event.EventID {
		t.Errorf("expected MessageId %q, got: %q", event.EventID, publishing.MessageId)
	}
	if publishing.DeliveryMode != amqp.Persistent {
		t.Errorf("expected Persistent delivery mode, got: %v", publishing.DeliveryMode)
	}

	// 2. 对比断言：旧版单队列模式（ladders 为空且 delayQueue != ""），Expiration 必须不为空且数值大于 0
	legacyTop := topology{
		exchange:        exchangeName,
		ladders:         nil,
		delayQueue:      delayQueueName,
		delayRoutingKey: delayRoutingKey,
		cancelQueue:     cancelQueueName,
		failedQueue:     failedQueueName,
	}
	legacyPub, legacyRoutingKey, err := buildOrderTimeoutPublishing(now, event, legacyTop)
	if err != nil {
		t.Fatalf("buildOrderTimeoutPublishing legacy mode failed: %v", err)
	}
	if legacyPub.Expiration == "" {
		t.Errorf("expected non-empty Expiration in legacy single-queue mode")
	} else {
		ms, parseErr := strconv.ParseInt(legacyPub.Expiration, 10, 64)
		if parseErr != nil {
			t.Errorf("invalid integer Expiration %q: %v", legacyPub.Expiration, parseErr)
		} else if ms <= 0 {
			t.Errorf("expected Expiration > 0, got: %d", ms)
		}
	}
	if legacyRoutingKey != delayRoutingKey {
		t.Errorf("expected legacy routingKey %q, got: %q", delayRoutingKey, legacyRoutingKey)
	}

	// 3. 运行态防御性校验：Worker.publishEvent 传入 nil channel 时拦截退出
	worker := &Worker{
		now:      func() time.Time { return now },
		topology: productionTopology,
	}
	outbox := model.OrderTimeoutOutbox{
		EventID:   event.EventID,
		OrderID:   event.OrderID,
		UserID:    event.UserID,
		TimeoutAt: event.TimeoutAt,
	}
	publishErr := worker.publishEvent(context.Background(), nil, outbox)
	if publishErr == nil || !strings.Contains(publishErr.Error(), "channel is nil") {
		t.Fatalf("expected channel is nil error, got: %v", publishErr)
	}
}
