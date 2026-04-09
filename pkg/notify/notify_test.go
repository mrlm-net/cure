package notify

import (
	"context"
	"testing"
	"time"
)

type mockChannel struct {
	name      string
	sent      []Notification
	responses chan Response
}

func newMockChannel(name string, bidir bool) *mockChannel {
	m := &mockChannel{name: name}
	if bidir {
		m.responses = make(chan Response, 10)
	}
	return m
}

func (m *mockChannel) Name() string { return m.name }
func (m *mockChannel) Send(_ context.Context, n Notification) (string, error) {
	m.sent = append(m.sent, n)
	return "receipt-1", nil
}
func (m *mockChannel) Responses() <-chan Response {
	return m.responses
}

func TestDispatcherNotify(t *testing.T) {
	ch1 := newMockChannel("ch1", false)
	ch2 := newMockChannel("ch2", false)
	d := NewDispatcher(ch1, ch2)

	n := Notification{SessionID: "s1", Summary: "done"}
	if err := d.Notify(context.Background(), n); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	if len(ch1.sent) != 1 || len(ch2.sent) != 1 {
		t.Errorf("ch1=%d, ch2=%d, want 1 each", len(ch1.sent), len(ch2.sent))
	}
}

func TestDispatcherWaitResponse(t *testing.T) {
	ch := newMockChannel("bidir", true)
	d := NewDispatcher(ch)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	d.StartListening(ctx)

	go func() {
		time.Sleep(50 * time.Millisecond)
		ch.responses <- Response{SessionID: "s1", ChannelID: "bidir", Text: "yes"}
	}()

	resp, err := d.WaitResponse(ctx, "s1")
	if err != nil {
		t.Fatalf("WaitResponse: %v", err)
	}
	if resp.Text != "yes" {
		t.Errorf("text = %q, want %q", resp.Text, "yes")
	}
}

func TestDispatcherTimeout(t *testing.T) {
	d := NewDispatcher()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := d.WaitResponse(ctx, "s1")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestDeliverResponseNoWaiter(t *testing.T) {
	d := NewDispatcher()
	ok := d.DeliverResponse(Response{SessionID: "nobody"})
	if ok {
		t.Error("should return false when no waiter")
	}
}
