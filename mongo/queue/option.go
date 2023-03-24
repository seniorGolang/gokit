package queue

import (
	"time"

	"github.com/seniorGolang/gokit/types/uuid"
)

const (
	PriorityHigh = iota
	PriorityNormal
	PriorityLow
)

type option func(item *queueItem)

func ID(id uuid.UUID) option {
	return func(item *queueItem) {
		item.Id = id
	}
}

func Hold() option {
	return func(item *queueItem) {
		item.AutoRemove = false
	}
}

func Priority(priority int) option {
	return func(item *queueItem) {
		item.Priority = priority
	}
}

func DeferDuration(d time.Duration) option {
	return func(item *queueItem) {
		t := time.Now().Add(d)
		item.Relevant = &t
	}
}

func DeferTime(t time.Time) option {
	return func(item *queueItem) {
		item.Relevant = &t
	}
}

func Expire(d time.Duration) option {
	return func(item *queueItem) {
		t := time.Now().Add(d)
		item.Expire = &t
	}
}

func ExpireAt(t time.Time) option {
	return func(item *queueItem) {
		item.Expire = &t
	}
}
