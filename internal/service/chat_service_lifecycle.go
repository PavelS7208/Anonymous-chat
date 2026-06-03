package service

import (
	"context"
	"errors"
	"fmt"
)

type Starter interface {
	Start(ctx context.Context) error
}

type Closer interface {
	Close() error
}

type lifecycleEntry struct {
	starter Starter // nil если не умеет Start
	closer  Closer  // nil если не умеет Close
}

type ChatServiceLifecycle struct {
	entries []lifecycleEntry // один список, порядок сохранён
}

func (lc *ChatServiceLifecycle) Register(obj any) {
	entry := lifecycleEntry{}
	if s, ok := obj.(Starter); ok {
		entry.starter = s
	}
	if c, ok := obj.(Closer); ok {
		entry.closer = c
	}
	lc.entries = append(lc.entries, entry)
}

func (lc *ChatServiceLifecycle) Start(ctx context.Context) error {
	for i, entry := range lc.entries {
		if entry.starter == nil {
			continue // этот не умеет Start — пропускаем
		}
		if err := entry.starter.Start(ctx); err != nil {
			lc.rollback(i) // откатываем всё до i
			return fmt.Errorf("start %T: %w", entry.starter, err)
		}
	}
	return nil
}

// rollback — обратный порядок, только до failedAt
func (lc *ChatServiceLifecycle) rollback(failedAt int) {
	for i := failedAt - 1; i >= 0; i-- {
		if lc.entries[i].closer != nil {
			_ = lc.entries[i].closer.Close()
		}
	}
}

// Close — всё в обратном порядке, собираем ошибки
func (lc *ChatServiceLifecycle) Close() error {
	var errs []error
	for i := len(lc.entries) - 1; i >= 0; i-- {
		if lc.entries[i].closer == nil {
			continue
		}
		if err := lc.entries[i].closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
