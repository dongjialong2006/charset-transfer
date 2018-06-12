package transfer

import (
	"charset-transfer-tool/config"
	"charset-transfer-tool/log"
	"charset-transfer-tool/pool"
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/sirupsen/logrus"
)

type TCP struct {
	max  int32
	min  int32
	ctx  context.Context
	log  *logrus.Entry
	cfg  *config.Configure
	ext  chan struct{}
	pool *pool.Pool
}

func NewTCP(ctx context.Context, cfg *config.Configure, max, min int) *TCP {
	return &TCP{
		min: int32(min),
		max: int32(max),
		ctx: ctx,
		cfg: cfg,
		log: log.New("tcp"),
		ext: make(chan struct{}),
	}
}

func (t *TCP) Start() error {
	if "" == t.cfg.Local.IP || "" == t.cfg.Local.Protocol {
		return fmt.Errorf("server addr is empty, please check it.")
	}

	ch := make(chan interface{}, 256)
	t.pool = pool.NewPool(t.ctx, pool.NewConfig(t.min, t.max), ch, t.do)
	t.pool.SetLogHandle(t.log)
	t.pool.Start()

	addr, err := net.ResolveTCPAddr(t.cfg.Local.Protocol, fmt.Sprintf("%s:%s", t.cfg.Local.IP, t.cfg.Local.Port))
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP(t.cfg.Local.Protocol, addr)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				if io.EOF == err {
					return
				}
				t.log.Error(err)
				continue
			}
			go t.handle(conn, ch)
		}
	}()
	<-t.ext
	listener.Close()
	return nil
}

func (t *TCP) handle(conn net.Conn, ch chan interface{}) {
	if nil == conn {
		t.log.Error("tcp connetion is nil, please check it.")
		return
	}
	defer conn.Close()

	ip := conn.RemoteAddr().String()
	ip = strings.Split(ip, ":")[0]

	var buf []byte = make([]byte, 4096)
	for {
		num, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				return
			}
			t.log.Error(err)
			return
		}
		ch <- NewRequest(conn, ip, buf[:num])
	}
	return
}

func (t *TCP) reply(conn net.Conn, ip string, buf []byte) error {
	data, err := Transfer(ip, buf, t.cfg.Remote.Charset, true)
	if nil != err {
		return err
	}

	_, err = conn.Write(data)
	if nil != err {
		return err
	}

	return nil
}

func (t *TCP) do(data interface{}) {
	if nil == data {
		return
	}
	req, ok := data.(*Request)
	if !ok {
		return
	}

	conn, err := dail(fmt.Sprintf("%s:%s", t.cfg.Remote.IP, t.cfg.Remote.Port), t.cfg.Remote.Protocol)
	if nil != err {
		t.log.Error(err)
		return
	}
	defer conn.Close()

	value, err := Transfer(req.Addr.(string), req.Data, t.cfg.Remote.Charset, false)
	if nil != err {
		t.log.Error(err)
		return
	}

	_, err = conn.Write(value)
	if nil != err {
		t.log.Error(err)
		return
	}

	var resp = make([]byte, 4096)
	num, err := conn.Read(resp)
	if nil != err {
		t.log.Error(err)
		return
	}

	if err = t.reply(req.Conn, req.Addr.(string), resp[:num]); nil != err {
		t.log.Error(err)
	}

	return
}

func (t *TCP) Stop() {
	if nil != t.pool {
		t.pool.Stop()
	}
	close(t.ext)
}
