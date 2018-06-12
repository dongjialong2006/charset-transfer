package transfer

import (
	"charset-transfer/config"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-iconv/iconv"
)

type Server interface {
	Start() error
	Stop()
}

type Request struct {
	Addr interface{}
	Data []byte
	Conn net.Conn
}

func NewRequest(conn net.Conn, addr string, data []byte) *Request {
	return &Request{
		Conn: conn,
		Addr: addr,
		Data: data,
	}
}

func dail(addr string, procotol string) (net.Conn, error) {
	if "" == addr {
		return nil, fmt.Errorf("addr is empty.")
	}

	switch strings.ToUpper(procotol) {
	case "TCP", "UDP":
	default:
		return nil, fmt.Errorf("unknown protocol type:%s.", procotol)
	}

	return net.DialTimeout(procotol, addr, time.Minute)
}

func Transfer(dest string, data []byte, dCharset string, cvt bool) ([]byte, error) {
	var ilen int = 0
	if "" == dest {
		return nil, fmt.Errorf("dest ip is empty.")
	}
	if 0 == len(data) {
		return nil, fmt.Errorf("buf is empty, please check it.")
	}
	if "" == dCharset {
		return nil, fmt.Errorf("dest charset is empty, please check it.")
	}

	sCharset, ok := config.GetSourceAddrCharset(dest)
	if !ok {
		return nil, fmt.Errorf("ip:%s is not exist in config.", dest)
	}

	var buf = make([]byte, 4096)
	if cvt {
		tmp, err := iconv.Open(sCharset, dCharset)
		if nil != err {
			return nil, err
		}
		defer tmp.Close()

		_, ilen, err := tmp.Conv(data, buf)
		if nil != err {
			return nil, err
		}
		return buf[:ilen], nil
	}

	tmp, err := iconv.Open(dCharset, sCharset)
	if nil != err {
		return nil, err
	}
	defer tmp.Close()

	_, ilen, err = tmp.Conv(data, buf)
	if nil != err {
		return nil, err
	}

	return buf[:ilen], nil
}
