package natsPool

import (
	"errors"
	"fmt"
	"github.com/nats-io/nats.go"
	"natsPool.xupeichun.github.com/interfacer"
	"sync"
)

var (
	// ErrClosed is the error resulting if the pool is closed via pool.Close().
	ErrClosed = errors.New("pool is closed")
)

/**
nats 连接生产工厂
*/
type Factory func() (nats.Conn, error)

/**
连接池 对象
*/
type channelPool struct {
	mux      sync.RWMutex
	connChan chan NatsConn
	factory  Factory
}

/**
连接池初始化
*/
func NewChannelPool(initialCap, maxCap int, factory Factory) (interfacer.IPool, error) {
	if initialCap < 0 || maxCap <= 0 || initialCap > maxCap {
		return nil, errors.New("invalid capacity settings")
	}
	c := &channelPool{
		connChan: make(chan NatsConn, maxCap),
		factory:  factory,
	}
	for i := 0; i < initialCap; i++ {
		conn, err := factory()
		if err != nil {
			c.Stop()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", err)
		}
		c.connChan <- NatsConn{
			Conn: conn,
		}
	}
	return c, nil
}

/**
现程安全获取 资源
*/
func (p *channelPool) getConnChanAndFactory() (chan NatsConn, Factory) {
	p.mux.Lock()
	connChan := p.connChan
	f := p.factory
	defer p.mux.Unlock()
	return connChan, f
}

/**
连接池中申请  资源
如果连接池为空，通过工厂临时生产 资源
*/
func (p *channelPool) Get() (NatsConn, error) {
	connChan, f := p.getConnChanAndFactory()
	if connChan == nil {
		return NatsConn{}, ErrClosed
	}

	select {
	case conn := <-connChan:
		if conn.IsClosed() {
			return NatsConn{}, ErrClosed
		}
		conn.pool = p
		return conn, nil
	default:
		c, err := f()
		if err != nil {
			return NatsConn{}, err
		}
		return NatsConn{
			Conn: c,
		}, nil
	}
}

/**
向连接池中归还 资源
如果 连接池 是满的,或 连接池关闭状态，将 当前 连接 关闭
*/
func (p *channelPool) put(conn NatsConn) error {
	if conn.IsClosed() {
		return fmt.Errorf("nats conn  is nil, rejected.")
	}
	p.mux.Lock()
	defer p.mux.Unlock()

	if p.connChan == nil {
		// 连接池被关闭，关闭剩余  连接
		p.Stop()
	}

	//回收连接资源，如果连接池满了，丢弃当前连接
	select {
	case p.connChan <- conn:
		return nil
	default:
		if !conn.IsClosed() {
			conn.Close()
		}
		return nil
	}
}

/**
连接池销毁
*/
func (p *channelPool) Stop() {
	p.mux.Lock()
	connChan := p.connChan
	p.connChan = nil
	p.factory = nil
	p.mux.Unlock()

	if connChan == nil {
		return
	}
	close(connChan)
	for conn := range connChan {
		conn.Close()
	}
}

/**
连接池中的长度
*/
func (p *channelPool) Len() int {
	conn, _ := p.getConnChanAndFactory()
	return len(conn)
}
