package config

import (
	"context"
	"fmt"
	"time"
)

type Limits struct {
	MaxBatch         int
	MaxMaterialGrams int
	RequestTimeout   time.Duration
	WorkerInterval   time.Duration
}

func DefaultLimits() Limits {
	return Limits{MaxBatch: 100, MaxMaterialGrams: 1000000, RequestTimeout: 15 * time.Second, WorkerInterval: time.Second}
}
func (l Limits) Validate() error {
	if l.MaxBatch <= 0 || l.MaxBatch > 10000 {
		return fmt.Errorf("max batch out of range")
	}
	if l.MaxMaterialGrams <= 0 {
		return fmt.Errorf("max material out of range")
	}
	if l.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout invalid")
	}
	if l.WorkerInterval <= 0 {
		return fmt.Errorf("worker interval invalid")
	}
	return nil
}
func (l Limits) AllowBatch(n int) bool    { return n > 0 && n <= l.MaxBatch }
func (l Limits) AllowMaterial(n int) bool { return n > 0 && n <= l.MaxMaterialGrams }
func (l Limits) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, l.RequestTimeout)
}
