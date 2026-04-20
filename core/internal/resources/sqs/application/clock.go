package application

import "time"

type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now().UTC()
}

func (realClock) Sleep(duration time.Duration) {
	if duration <= 0 {
		return
	}
	time.Sleep(duration)
}
