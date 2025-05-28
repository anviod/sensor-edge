package sparkplugb

import (
	"sync"
	"time"
)

type Message struct {
	ID      string
	Topic   string
	QoS     byte
	Payload []byte
	Created time.Time
	Retry   int
}

type Storage interface {
	Save(msg Message) error
	Delete(id string) error
	LoadAll() ([]Message, error)
}

type OutboundQueue struct {
	queue   []Message
	storage Storage
	mu      sync.Mutex
}

func NewOutboundQueue(storage Storage) *OutboundQueue {
	msgs, _ := storage.LoadAll()
	return &OutboundQueue{queue: msgs, storage: storage}
}

func (q *OutboundQueue) Enqueue(msg Message) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.queue = append(q.queue, msg)
	q.storage.Save(msg)
}

func (q *OutboundQueue) Process(publish func(Message) error, maxRetry int, backoff func(int) time.Duration) {
	q.mu.Lock()
	defer q.mu.Unlock()
	var remain []Message
	for _, msg := range q.queue {
		err := publish(msg)
		if err != nil {
			msg.Retry++
			if msg.Retry < maxRetry {
				time.Sleep(backoff(msg.Retry))
				remain = append(remain, msg)
			} else {
				// 失败告警
			}
		} else {
			q.storage.Delete(msg.ID)
		}
	}
	q.queue = remain
}
