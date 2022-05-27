package interfacer

import (
	natsPool "natsPool.xupeichun.github.com"
)

type IPool interface {
	Get() (natsPool.NatsConn, error)
	Stop()
	Len() int
}
