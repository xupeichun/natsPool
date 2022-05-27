package natsPool

import (
	"github.com/nats-io/nats.go"
	"sync"
)

type NatsConn struct {
	nats.Conn
	mux      sync.RWMutex
	pool     *channelPool
	unusable bool
}

/**
nats 连接如果可用放回资源池，如果不可用，及时销毁
*/
func (c *NatsConn) Close() error {
	c.mux.Lock()
	defer c.mux.Unlock()
	if c.unusable && !c.Conn.IsClosed() {
		c.Conn.Close()
		return nil
	}
	return c.pool.put(c.Conn)
}

/**
当  nats conn 不可用时，标记并及时销毁
*/
func (c *NatsConn) MarkUnusable() {
	c.mux.Lock()
	c.unusable = true
	defer c.mux.Unlock()
}
