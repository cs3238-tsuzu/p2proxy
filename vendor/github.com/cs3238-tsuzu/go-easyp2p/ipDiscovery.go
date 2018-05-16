package easyp2p

import (
	"net"
	"strconv"
	"time"

	"github.com/gortc/stun"
)

type connectionNopCloser struct {
	addr net.Addr
	conn net.PacketConn
}

func (c *connectionNopCloser) Read(b []byte) (int, error) {
	for {
		n, addr, err := c.conn.ReadFrom(b)

		if err != nil {
			return 0, err
		}

		if addr.String() == c.addr.String() {
			return n, nil
		}
	}
}

func (c *connectionNopCloser) Write(b []byte) (int, error) {
	return c.conn.WriteTo(b, c.addr)
}

func (c *connectionNopCloser) Close() error {
	return nil
}

func newConnectionNopCloser(conn net.PacketConn, addr string) (stun.Connection, error) {
	udp, err := net.ResolveUDPAddr("udp", addr)

	if err != nil {
		return nil, err
	}
	return &connectionNopCloser{
		conn: conn,
		addr: udp,
	}, nil
}

func discoverIPAddressesWithSTUN(addr string, conn net.PacketConn, timeout time.Duration) (string, error) {
	c, err := newConnectionNopCloser(conn, addr)

	if err != nil {
		return "", err
	}
	client, err := stun.NewClient(stun.ClientOptions{
		Connection:  c,
		TimeoutRate: 100 * time.Millisecond,
	})
	conn.SetDeadline(time.Now().Add(timeout))

	if err != nil {
		return "", err
	}

	defer func() {
		conn.SetDeadline(time.Now().Add(-1))
		client.Close()
		conn.SetDeadline(time.Time{})
	}()

	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	var retErr error
	var res string
	if err := client.Do(message, time.Now().Add(5*time.Second), func(event stun.Event) {
		if event.Error != nil {
			retErr = event.Error

			return
		}

		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(event.Message); err != nil {
			retErr = err

			return
		}

		res = xorAddr.IP.String() + ":" + strconv.FormatInt(int64(xorAddr.Port), 10)
	}); err != nil {
		return "", err
	}

	if retErr != nil {
		return "", retErr
	}

	return res, nil
}
