package natsPool

import "github.com/nats-io/nats.go"

type IPool interface {
	Get() (*nats.Conn, error)
}
