package pool

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type Config struct {
	Min int32
	Max int32
}

func NewConfig(min, max int32) *Config {
	return &Config{
		Min: min,
		Max: max,
	}
}

type control struct {
	CurrRunNum   int32
	MinThreshold int32
	MaxThreshold int32
}

type Pool struct {
	cfg *Config
	ctl *control
	ext chan bool
	stp chan struct{}
	log *logrus.Entry
	ctx context.Context
	bch chan interface{}
	dos func(data interface{})
}

func NewPool(ctx context.Context, cfg *Config, bch chan interface{}, dos func(data interface{})) *Pool {
	p := &Pool{
		ctx: ctx,
		cfg: cfg,
		bch: bch,
		dos: dos,
		ext: make(chan bool, 1),
		stp: make(chan struct{}),
	}

	p.init()

	return p
}

func (p *Pool) init() {
	if p.cfg.Min > p.cfg.Max {
		p.cfg.Min = p.cfg.Max / 3
	}

	if 0 == p.cfg.Min {
		p.cfg.Min = 1
	}

	if 0 == p.cfg.Max {
		p.cfg.Max = p.cfg.Min * 3
	}

	p.ctl = &control{
		CurrRunNum:   0,
		MinThreshold: int32(cap(p.bch)) / 3,
		MaxThreshold: int32(cap(p.bch)) * 2 / 3,
	}

	return
}

func (p *Pool) handle() {
	atomic.AddInt32(&p.ctl.CurrRunNum, 1)
	defer atomic.AddInt32(&p.ctl.CurrRunNum, -1)

	tid := time.Now().UnixNano()
	p.record(fmt.Sprintf("thread id:%d is started.", tid))
	defer p.record(fmt.Sprintf("thread id:%d is stoped.", tid))

	for num := 1; ; num++ {
		select {
		case <-p.ctx.Done():
			return
		case <-p.ext:
			return
		case data, ok := <-p.bch:
			if !ok {
				return
			}
			p.dos(data)
			p.record(fmt.Sprintf("thread id:%d, package num:%d is done.", tid, num))
		}
	}

	return
}

func (p *Pool) record(value string) {
	if "" == value {
		return
	}
	if nil == p.log {
		fmt.Println(value)
		return
	}
	p.log.Debug(value)
}

func (p *Pool) check() {
	if atomic.LoadInt32(&p.ctl.CurrRunNum) >= p.cfg.Max {
		return
	}

	if len(p.bch) > int(p.ctl.MaxThreshold) || atomic.LoadInt32(&p.ctl.CurrRunNum) < p.cfg.Min {
		go p.handle()
		return
	}

	if len(p.bch) < int(p.ctl.MinThreshold) && atomic.LoadInt32(&p.ctl.CurrRunNum) > p.cfg.Min {
		p.ext <- true
	}

	return
}

func (p *Pool) SetLogHandle(log *logrus.Entry) {
	p.log = log
}

func (p *Pool) Start() {
	go func() {
		tick := time.Tick(time.Millisecond)
		for {
			select {
			case <-p.ctx.Done():
				return
			case <-p.stp:
				return
			case <-tick:
				p.check()
			}
		}
	}()
}

func (p *Pool) Stop() {
	close(p.stp)
	time.Sleep(time.Second)
	for i := int32(0); i < atomic.LoadInt32(&p.ctl.CurrRunNum); i++ {
		p.ext <- true
	}
}
