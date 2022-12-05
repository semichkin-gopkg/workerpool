package workerpool

import (
	"github.com/semichkin-gopkg/configurator"
	"golang.org/x/exp/constraints"
)

type Configuration struct {
	WorkersCount        uint
	JobsChannelCapacity uint
}

func WithWorkersCount(count uint) configurator.Updater[Configuration] {
	return func(c *Configuration) {
		c.WorkersCount = max(count, 1)
	}
}

func WithJobsChannelCapacity(capacity uint) configurator.Updater[Configuration] {
	return func(c *Configuration) {
		c.JobsChannelCapacity = max(capacity, 1)
	}
}

func max[T constraints.Ordered](s ...T) T {
	if len(s) == 0 {
		var zero T
		return zero
	}
	m := s[0]
	for _, v := range s {
		if v > m {
			m = v
		}
	}
	return m
}
