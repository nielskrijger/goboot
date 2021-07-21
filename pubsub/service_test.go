package pubsub_test

import (
	"context"
	"errors"
	"math"
	"os"
	"testing"
	"time"

	appcontext "github.com/nielskrijger/goboot/context"
	"github.com/nielskrijger/goboot/pubsub"
	"github.com/nielskrijger/goboot/utils"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

var (
	topicID           = "test-topic"
	subID             = "test-subscription"
	topicID2          = "test-topic-without-subscription"
	deadLetterTopicID = "dead-letter-topic"
	deadLetterSubID   = "dead-letter-subscription"

	errTest  = errors.New("test error")
	errTest2 = errors.New("test error 2")
)

func newPubSubEmulatorService(t *testing.T, deadLetter bool) (*pubsub.Service, *utils.TestLogger) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	if _, exists := os.LookupEnv("PUBSUB_EMULATOR_HOST"); !exists {
		_ = os.Setenv("PUBSUB_EMULATOR_HOST", "localhost:8085")
	}

	if _, exists := os.LookupEnv("PUBSUB_PROJECT_ID"); !exists {
		_ = os.Setenv("PUBSUB_PROJECT_ID", "metrix-io")
	}

	opts := []pubsub.Option{
		pubsub.WithChannel(&pubsub.Channel{ID: "test-channel", TopicID: topicID, SubscriptionID: subID}),
		pubsub.WithChannel(&pubsub.Channel{ID: "without-subscription", TopicID: topicID2}),
	}

	if deadLetter {
		opts = append(opts, pubsub.WithDeadLetter(
			&pubsub.Channel{TopicID: deadLetterTopicID, SubscriptionID: deadLetterSubID}))
	}

	// configure pubsub Service with appcontext
	s := pubsub.NewPubSubService("metrix-io", opts...)
	appctx := appcontext.NewAppContext("../testdata/conf", "postgres-invalid")

	testLogger := &utils.TestLogger{}
	appctx.Log = zerolog.New(testLogger)

	assert.Nil(t, s.Configure(appctx))

	// Recreate all topics and subscriptions for each test
	if err := s.DeleteAll(); err != nil {
		panic(err)
	}

	assert.Nil(t, s.Init())

	return s, testLogger
}

func TestReceiveAll_Success(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)
	defer s.Close()

	ctx := context.Background()
	_ = s.PublishEvent(ctx, "test-channel", "ev1", "test message")
	_ = s.PublishEvent(ctx, "test-channel", "ev2", "test message2")

	msgs, _ := s.ReceiveNr(ctx, "test-channel", 2)

	// First published event
	ev1 := findEvent(msgs, "ev1")
	assert.NotNil(t, ev1)
	assert.Equal(t, "\"test message\"", string(ev1.Data))

	// Second published event
	ev2 := findEvent(msgs, "ev2")
	assert.NotNil(t, ev2)
	assert.Equal(t, "\"test message2\"", string(ev2.Data))
}

func findEvent(msgs []*pubsub.RichMessage, eventName string) *pubsub.RichMessage {
	for _, msg := range msgs {
		if msg.Attributes["event"] == eventName {
			return msg
		}
	}

	return nil
}

func TestReceiveAll_ChannelDoesNotExist(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)
	defer s.Close()

	ctx := context.Background()
	_, err := s.ReceiveNr(ctx, "unknown", 1)

	assert.Equal(t, "channel \"unknown\" not found", err.Error())
}

func TestReceiveAll_ContextClosed(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)
	assert.Nil(t, s.Close())

	ctx := context.Background()
	_, err := s.ReceiveNr(ctx, "test-channel", 1)

	assert.Equal(t, pubsub.ErrClosed, err)
}

func TestPublishEvent_ChannelDoesNotExist(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)
	ctx := context.Background()

	err := s.PublishEvent(ctx, "unknown", "ev1", "test message")

	assert.Equal(t, "channel \"unknown\" not found", err.Error())
}

func TestPublishEvent_MarshalError(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)
	ctx := context.Background()

	err := s.PublishEvent(ctx, "test-channel", "ev1", math.Inf(1))

	assert.Contains(t, err.Error(), "failed to marshal payload")
}

func TestPublishEvent_ContextClosed(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)

	tout, _ := time.ParseDuration("1ms")

	ctx, cancel := context.WithTimeout(context.Background(), tout)
	defer cancel()

	err := s.PublishEvent(ctx, "test-channel", "ev1", "test message")

	assert.Equal(t, pubsub.ErrClosed, err)
}

func TestReceive_Success(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)
	ctx := context.Background()
	c := make(chan *pubsub.RichMessage)

	go func() {
		_ = s.Receive(ctx, "test-channel", func(ctx context.Context, m *pubsub.RichMessage) {
			c <- m
		})
	}()

	_ = s.PublishEvent(ctx, "test-channel", "ev1", "test message")
	msg := <-c

	assert.Equal(t, "ev1", msg.Attributes["event"])
	assert.Equal(t, "\"test message\"", string(msg.Data))
}

func TestReceive_ChannelDoesNotExit(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)
	ctx := context.Background()

	err := s.Receive(ctx, "unknown", func(context.Context, *pubsub.RichMessage) {})

	assert.Equal(t, "channel \"unknown\" not found", err.Error())
}

func TestReceive_ChannelWithoutSubscription(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)
	ctx := context.Background()

	err := s.Receive(ctx, "without-subscription", func(context.Context, *pubsub.RichMessage) {})

	assert.Equal(t, "channel \"without-subscription\" does not have a subscription", err.Error())
}

func TestDeleteChannel_ChannelDoesNotExist(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)

	err := s.DeleteChannel("unknown")

	assert.Equal(t, "channel \"unknown\" not found", err.Error())
}

func TestDeleteChannel_ServiceClosed(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)
	assert.Nil(t, s.Close())

	err := s.DeleteChannel("test-channel")

	assert.Equal(t, pubsub.ErrClosed, err)
}

func TestDeleteAll_ServiceClosed(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)
	assert.Nil(t, s.Close())

	err := s.DeleteAll()

	assert.Equal(t, pubsub.ErrClosed, err)
}

func TestTryClose_LogErrorOnFailure(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)
	assert.Nil(t, s.Close())
	assert.EqualError(t, s.Close(), "pubsub publisher closing error: rpc error: code = Canceled desc = grpc: the client connection is closing")
}

func TestDeadLetter_Success(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, true)
	ctx := context.Background()

	_ = s.PublishEvent(ctx, "test-channel", "ev1", "test message")
	msgs, _ := s.ReceiveNr(ctx, "test-channel", 1)
	originalMessageID := msgs[0].ID

	_ = msgs[0].DeadLetter(ctx, errTest)

	// One message in dead letter channel
	msgs, _ = s.ReceiveNr(ctx, "dead-letter", 1)

	attr := msgs[0].Attributes
	assert.Equal(t, "\"test message\"", string(msgs[0].Data))
	assert.Equal(t, "1", attr["deadLetterCount"])
	assert.Equal(t, topicID, attr["originalTopicID"])
	assert.Equal(t, subID, attr["originalSubscriptionID"])
	assert.Equal(t, originalMessageID, attr["originalMessageID"])
	assert.Equal(t, "test error", attr["error"])
}

func TestDeadLetter_IncrementDeadLetterCounter(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, true)
	ctx := context.Background()

	// Publish an event and dead letter it twice=
	_ = s.PublishEvent(ctx, "test-channel", "ev1", "test message")
	msgs, _ := s.ReceiveNr(ctx, "test-channel", 1)
	_ = msgs[0].DeadLetter(ctx, errTest)
	msgs, _ = s.ReceiveNr(ctx, "dead-letter", 1)
	_ = msgs[0].DeadLetter(ctx, errTest2)

	// One message in dead letter channel
	msgs, _ = s.ReceiveNr(ctx, "dead-letter", 1)

	attr := msgs[0].Attributes
	assert.Equal(t, "\"test message\"", string(msgs[0].Data))
	assert.Equal(t, "2", attr["deadLetterCount"])
	assert.Equal(t, "test error 2", attr["error"])
}

func TestDeadLetter_ErrorOnFailure(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, false)

	msg := &pubsub.RichMessage{Service: s}
	err := msg.DeadLetter(context.Background(), errTest)

	assert.Equal(t, "no deadletter channel configured", err.Error())
}

func TestRetryableError_Success(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, true)
	ctx := context.Background()
	_ = s.PublishEvent(ctx, "test-channel", "ev1", "test message")
	msgs, _ := s.ReceiveNr(ctx, "test-channel", 1)
	err := msgs[0].RetryableError(ctx, errTest)

	assert.Nil(t, err)

	// No messages in dead letter channel
	cctx, cancel := context.WithTimeout(ctx, time.Duration(100)*time.Millisecond)
	defer cancel()

	msgs, err = s.ReceiveNr(cctx, "dead-letter", 1)

	assert.Nil(t, err)
	assert.Len(t, msgs, 0)
}

func TestRetryableError_MaxRetryAgeExpired(t *testing.T) {
	s, _ := newPubSubEmulatorService(t, true)
	ctx := context.Background()
	_ = s.PublishEvent(ctx, "test-channel", "ev1", "test message")
	msgs, _ := s.ReceiveNr(ctx, "test-channel", 1)
	msgs[0].PublishTime = time.Now().Add(time.Duration(-121) * time.Second)

	err := msgs[0].RetryableError(ctx, errTest)
	assert.Nil(t, err)

	dead, _ := s.ReceiveNr(ctx, "dead-letter", 1)
	assert.Equal(t, msgs[0].ID, dead[0].Attributes["originalMessageID"])
}

var trimTests = []struct {
	in       string
	maxBytes int
	out      string
}{
	{"日本語", 0, ""},
	{"日本語", 1, ""},
	{"日本語", 2, ""},
	{"日本語", 3, "日"},
	{"日本語", 4, "日"},
	{"日本語", 5, "日"},
	{"日本語", 6, "日本"},
}

func TestStringTrimLeftBytes(t *testing.T) {
	for _, tt := range trimTests {
		assert.Equal(t, tt.out, pubsub.TrimLeftBytes(tt.in, tt.maxBytes))
	}
}