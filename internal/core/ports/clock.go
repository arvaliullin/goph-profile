package ports

import "time"

//go:generate mockgen -source=clock.go -destination=mocks/clock_mock.go -package=mocks

// Clock абстрагирует время для тестов.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// NewRealClock возвращает часы в UTC.
func NewRealClock() Clock { return realClock{} }
