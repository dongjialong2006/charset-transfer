package transfer

import (
	"charset-transfer/config"
	"charset-transfer/log"
	"charset-transfer/pool"
	"context"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
)

type UDP struct {
	max  int32
	min  int32
	ctx  context.Context
	log  *logrus.Entry
	pool *pool.Pool
	cfg  *config.Configure
	ext  chan struct{}
}

func NewUDP(ctx context.Context, cfg *config.Configure, max, min int) *UDP {
	return &UDP{
		min: int32(min),
		max: int32(max),
		ctx: ctx,
		cfg: cfg,
		log: log.New("udp"),
		ext: make(chan struct{}),
	}
}

func (u *UDP) Start() error {
	if "" == u.cfg.Local.IP || "" == u.cfg.Local.Protocol {
		return fmt.Errorf("server addr is empty, please check it.")
	}

	ch := make(chan interface{}, 256)
	u.pool = pool.NewPool(u.ctx, pool.NewConfig(u.min, u.max), ch, u.do)
	u.pool.SetLogHandle(u.log)
	u.pool.Start()

	addr, err := net.ResolveUDPAddr(u.cfg.Local.Protocol, fmt.Sprintf("%s:%s", u.cfg.Local.IP, u.cfg.Local.Port))
	if err != nil {
		return err
	}
	listener, err := net.ListenUDP(u.cfg.Local.Protocol, addr)
	if err != nil {
		return err
	}

	go func() {
		buf := make([]byte, 2048)
		for {
			num, addr, err := listener.ReadFromUDP(buf)
			if err != nil {
				u.log.Error(err)
				continue
			}
			ch <- &Request{Conn: listener, Data: buf[:num], Addr: addr}
		}
	}()
	<-u.ext
	listener.Close()
	return nil
}

func (u *UDP) do(data interface{}) {
	if nil == data {
		return
	}
	req, ok := data.(*Request)
	if !ok {
		return
	}

	conn, err := dail(fmt.Sprintf("%s:%s", u.cfg.Remote.IP, u.cfg.Remote.Port), u.cfg.Remote.Protocol)
	if nil != err {
		u.log.Error(err)
		return
	}
	defer conn.Close()

	addr := req.Addr.(*net.UDPAddr)
	value, err := Transfer(addr.IP.String(), req.Data, u.cfg.Remote.Charset, false)
	if nil != err {
		u.log.Error(err)
		return
	}

	_, err = conn.Write(value)
	if nil != err {
		u.log.Error(err)
		return
	}

	var resp = make([]byte, 4096)
	num, err := conn.Read(resp)
	if nil != err {
		u.log.Error(err)
		return
	}

	if _, err = conn.Write(resp[:num]); nil != err {
		u.log.Error(err)
	}

	return
}

func (u *UDP) Stop() {
	if nil != u.pool {
		u.pool.Stop()
	}
	close(u.ext)
}
