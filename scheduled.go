package workers

import (
	"strings"
	"time"

	"github.com/garyburd/redigo/redis"
)

type scheduled struct {
	keys   []string
	closed bool
	exit   chan bool
}

func (s *scheduled) start() {
	go s.poll(true)
}

func (s *scheduled) quit() {
	s.closed = true
}

func (s *scheduled) poll(continuing bool) {

	for {
		if s.closed {
			return
		}

		conn := Config.Pool.Get()

		now := time.Now().Unix()

		for _, key := range s.keys {
			key = Config.Namespace + key
			for {
				messages, _ := redis.Strings(conn.Do("zrangebyscore", key, "-inf", now, "limit", 0, 1))

				if len(messages) == 0 {
					break
				}

				message, _ := NewMsg(messages[0])

				if removed, err := redis.Int(conn.Do("zrem", key, messages[0])); removed == 1 && err == nil {
					queue, _ := message.Get("queue").String()
					queue = strings.TrimPrefix(queue, Config.Namespace)
					conn.Do("lpush", Config.Namespace+"queue:"+queue, message.ToJson())
				}
			}
		}

		conn.Close()
		if continuing {
			time.Sleep(time.Duration(Config.PollInterval) * time.Millisecond)
			// s.poll(true)
		}

	}

}

func newScheduled(keys ...string) *scheduled {
	return &scheduled{keys, false, make(chan bool)}
}
