// test1 project main.go
package main

import (
	_ "bytes"
	_ "encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	. "log_system"
	"net"
	"os"
	. "parsing_config"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	iconv "github.com/go-iconv/iconv"
)

var wgDay sync.WaitGroup
var metux sync.RWMutex
var stop_run_time int = 0
var config *Config = nil

const (
	MAX_BYTES_LEN = 1024 * 256
)

const (
	SUCCESS = iota
	TCP_RESET_BY_PEER
	TCP_READ_TIMEOUT
	TCP_WRITE_BROKEN_PIPE
	TCP_UNKNOWN_ERROR
)

type PackageTcpData struct {
	data []byte
	ip   string
	conn net.Conn
}

type PackageUdpData struct {
	data []byte
	addr *net.UDPAddr
	conn *net.UDPConn
}

type createProcessGroup struct {
	process_num int
	start_time  time.Time
	mu          sync.Mutex
	total_num   int64
}

func main() {
	flag.String("t", "", "run time, unit: 'minute', data type: 'integer data'.")
	flag.String("n", "", "concurrence thread num, data type: 'integer data'.")
	flag.String("c", "", "config file path.")
	flag.Parse()
	num := 0
	run_time := 1000.0
	bFlag := false
	config_file := "./test_tool.xml"
	args := os.Args[1:]
	for i, value := range args {
		if "-n" == value {
			num, _ = strconv.Atoi(args[i+1])
		} else if "-t" == value {
			run_time, _ = strconv.ParseFloat(args[i+1], 64)
		} else if "-c" == value {
			config_file = args[i+1]
		} else {
			// do nothing
		}
	}
	if config_file == "" || run_time == 0.0 {
		fmt.Println("config_file:", config_file, ", run_time:", run_time, ".")
		bFlag = true
	}
	if run_time <= 0.5 {
		run_time = 1.0
		fmt.Println("the system will use default run time:1.0, because you input this value less than 0.5")
	}
	if bFlag {
		fmt.Println("command error, USAGE: test [-t|-n|-c] [value], -n default value:25, -t default value:1.")
		fmt.Println("command error, this command must have 2 or 4 parameters, if you want to start 'test' on background, please use this command:nohup ./test &")
		fmt.Println("or use the command:nohup ./test 1> server.out 2> server.err.")
		return
	}
	runtime.GOMAXPROCS(runtime.NumCPU())
	var status int = 0
	chan_close_flag := false
	update_chan := make(chan int, 1)
	err := ParsingXml(config_file, update_chan)
	if err != nil {
		fmt.Println(err)
		return
	}
	config = GetConfigInfo()
	if "tcp" == config.Local_protocol {
		packQueue, err := startTcpServer(config.Local_ip+":"+config.Local_port, config.Local_protocol)
		if err != nil || nil == packQueue {
			fmt.Println(err)
			return
		}
		for update_chan != nil {
			status, chan_close_flag = <-update_chan
			if !chan_close_flag {
				fmt.Println("parsing xml auto update channel is closed.")
				return
			}
			switch status {
			case CONFIG_XML_START:
				stopOldServices()
				update_chan <- SYSTEM_ALL_OLD_SERVICES_STOP
			case CONFIG_XML_SUCCESS:
				err = Log_Handle.LogSystemStart("")
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				if num <= 0 {
					go startTcpServices(config.Remote_thread_num, run_time, update_chan, packQueue)
				} else {
					go startTcpServices(num, run_time, update_chan, packQueue)
				}
			case CONFIG_XML_ERROR:
				fallthrough
			case SYSTEM_SHUT_DOWN:
				Log_Handle.LogSystemStop()
				ConfigFileAutoUpdateStop()
				break
			}
			time.Sleep(time.Second)
		}
	} else if "udp" == config.Local_protocol {
		packQueue, err := startUdpServer(config.Local_ip+":"+config.Local_port, config.Local_protocol)
		if err != nil || nil == packQueue {
			fmt.Println(err)
			return
		}
		for update_chan != nil {
			status, chan_close_flag = <-update_chan
			if !chan_close_flag {
				fmt.Println("parsing xml auto update channel is closed.")
				return
			}
			switch status {
			case CONFIG_XML_START:
				stopOldServices()
				update_chan <- SYSTEM_ALL_OLD_SERVICES_STOP
			case CONFIG_XML_SUCCESS:
				err = Log_Handle.LogSystemStart("")
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				go startUdpServices(num, run_time, update_chan, packQueue)
			case CONFIG_XML_ERROR:
				fallthrough
			case SYSTEM_SHUT_DOWN:
				Log_Handle.LogSystemStop()
				ConfigFileAutoUpdateStop()
				break
			}
			time.Sleep(time.Second)
		}
	} else {
		fmt.Println("error protocol, please check it.")
		return
	}
	return
}

func startTcpServer(server string, protocol string) (chan *PackageTcpData, error) {
	var err error = nil
	if server == "" {
		err = errors.New("server is nil, please check it.")
		return nil, err
	}
	ch := make(chan *PackageTcpData, 1000)
	result_chan := make(chan bool, 1)
	go func() {
		serverAddr, err := net.ResolveTCPAddr(protocol, server)
		if err != nil {
			Log(LL_ERROR, err)
			result_chan <- false
			return
		}
		listener, err := net.ListenTCP(protocol, serverAddr)
		if err != nil {
			Log(LL_ERROR, err)
			result_chan <- false
			return
		}
		defer listener.Close()
		result_chan <- true
		for {
			conn, err := listener.Accept()
			if err != nil {
				Log(LL_ERROR, err)
				time.Sleep(1)
				continue
			}
			go dealTcpConnHandler(conn, ch)
		}
	}()
	if !(<-result_chan) {
		err = errors.New("listen tcp server error, please check it.")
		return nil, err
	}
	return ch, err
}

func startUdpServer(server string, protocol string) (chan *PackageUdpData, error) {
	var err error = nil
	if server == "" {
		err = errors.New("server is nil, please check it.")
		return nil, err
	}
	ch := make(chan *PackageUdpData, 1000)
	result_chan := make(chan bool, 1)
	go func() {
		serverAddr, err := net.ResolveUDPAddr(protocol, server)
		if err != nil {
			Log(LL_ERROR, err)
			result_chan <- false
			return
		}
		listener, err := net.ListenUDP(protocol, serverAddr)
		if err != nil {
			Log(LL_ERROR, err)
			result_chan <- false
			return
		}
		result_chan <- true
		buf := make([]byte, 2048)
		flag := false
		ip := ""
		for {
			n, remoteAddr, err := listener.ReadFromUDP(buf)
			if err != nil {
				if err != io.EOF {
					Log(LL_ERROR, "error during read:", err)
				}
				break
			}
			ip = strings.Split(remoteAddr.String(), ":")[0]
			_, flag = config.Charset_transfer[ip]
			if !flag {
				Log(LL_ERROR, "curr ip:", remoteAddr.String(), ", is not find in charset_transfer config, please check it.")
				continue
			}
			if n != 0 {
				data := &PackageUdpData{conn: listener, data: buf[:n], addr: remoteAddr}
				ch <- data
			}
		}
		defer listener.Close()
	}()
	if !(<-result_chan) {
		err = errors.New("listen udp server error, please check it.")
		return nil, err
	}
	return ch, err
}

func dealTcpConnHandler(conn net.Conn, packQueue chan *PackageTcpData) {
	if nil == conn {
		Log(LL_ERROR, "tcp connetion is nil, please check it.")
		return
	}
	defer conn.Close()
	ip := conn.RemoteAddr().String()
	ip = strings.Split(ip, ":")[0]
	_, flag := config.Charset_transfer[ip]
	if !flag {
		Log(LL_ERROR, "curr ip:", ip, ", is not find in charset_transfer config, please check it.")
		return
	}
	var buf []byte = make([]byte, 4096)
	var err error = nil
	var ilen int = 0
	for {
		ilen, err = conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				Log(LL_ERROR, err)
			}
			break
		} else {
			if ilen > 3 {
				if "exit" == string(buf[:ilen-1]) || "exit" == string(buf[:ilen-2]) || "exit" == string(buf[:ilen]) {
					Log(LL_INFO, "client server:", conn.RemoteAddr().String(), " is closed now.")
					break
				}
				data := &PackageTcpData{conn: conn, data: buf[:ilen], ip: ip}
				packQueue <- data
			}
		}
	}
	return
}

func stopOldServices() bool {
	metux.Lock()
	stop_run_time = 0
	metux.Unlock()
	time.Sleep(time.Second)
	return true
}

func startTcpServices(num int, run_time float64, ch chan int, packQueue chan *PackageTcpData) {
	remote_server := ""
	config := GetConfigInfo()
	separator := 0
	value := ""
	port_len := len(config.Remote_port)
	threadPool := make([]*createProcessGroup, 0)
	if 0 == port_len {
		Log(LL_ERROR, "remote_port is null, please check it")
		goto Result
	}
	if nil == ch {
		goto Result
	}
	wgDay.Add(port_len)
	for _, value = range config.Remote_port {
		remote_server = strings.Trim(config.Remote_ip, " ") + ":" + value
		fmt.Println("remote_server:", remote_server, ", remote_protocol:", config.Remote_protocol, ", process num:", num, ", run_time:", int(run_time*60), "seconds.")
		Log(LL_INFO, "remote_server:", remote_server, ", remote_protocol:", config.Remote_protocol, ", process num:", num, ", run_time:", int(run_time*60), " seconds.")
		var group *createProcessGroup = &createProcessGroup{process_num: num, start_time: time.Now()}
		go group.processTcpGroup(remote_server, config.Remote_protocol, config.Remote_charset, separator, run_time, packQueue)
		threadPool = append(threadPool, group)
		separator += num
	}
	wgDay.Wait()
	for index, value := range config.Remote_port {
		remote_server = config.Remote_ip + ":" + value
		Log(LL_INFO, "remote_server:", remote_server, ", the total packages num/time:", (threadPool[index]).total_num, "ps/", int(run_time*60), "s, average num:", ((float64)((threadPool[index]).total_num))/(run_time*60))
	}
	if 0 != stop_run_time {
		fmt.Println("the system is already stoped all old services.")
		Log(LL_INFO, "the system is already stoped all old services")
	} else {
		fmt.Println("the system is already stoped curr services.")
		Log(LL_INFO, "the system is already stoped curr services")
	}

Result:
	ch <- SYSTEM_SHUT_DOWN
	return
}

func startUdpServices(num int, run_time float64, ch chan int, packQueue chan *PackageUdpData) {
	remote_server := ""
	config := GetConfigInfo()
	separator := 0
	value := ""
	port_len := len(config.Remote_port)
	threadPool := make([]*createProcessGroup, 0)
	if 0 == port_len {
		Log(LL_ERROR, "remote_port is null, please check it")
		goto Result
	}
	if nil == ch {
		goto Result
	}
	wgDay.Add(port_len)
	for _, value = range config.Remote_port {
		remote_server = strings.Trim(config.Remote_ip, " ") + ":" + value
		fmt.Println("remote_server:", remote_server, ", remote_protocol:", config.Remote_protocol, ", process num:", num, ", run_time:", int(run_time*60), "seconds.")
		Log(LL_INFO, "remote_server:", remote_server, ", process num:", num, ", run_time:", int(run_time*60), " seconds.")
		var group *createProcessGroup = &createProcessGroup{process_num: num}
		go group.processUdpGroup(remote_server, config.Remote_protocol, config.Remote_charset, separator, run_time, packQueue)
		threadPool = append(threadPool, group)
		separator += num
	}
	wgDay.Wait()
	for index, value := range config.Remote_port {
		remote_server = config.Remote_ip + ":" + value
		Log(LL_INFO, "remote_server:", remote_server, ", the total packages num/time:", (threadPool[index]).total_num, "ps/", int(run_time*60), "s, average num:", ((float64)((threadPool[index]).total_num))/(run_time*60))
	}
	if 0 != stop_run_time {
		fmt.Println("the system is already stoped all old services.")
		Log(LL_INFO, "the system is already stoped all old services")
	} else {
		fmt.Println("the system is already stoped curr services.")
		Log(LL_INFO, "the system is already stoped curr services")
	}

Result:
	ch <- SYSTEM_SHUT_DOWN
	return
}

func (group *createProcessGroup) processUdpGroup(server string, procotol string, destCharset string, separator int, run_time float64, packQueue chan *PackageUdpData) {
	num := group.process_num
	wgDay.Add(num)
	for i := separator; i < separator+num; i++ {
		go group.packageProcUdp(i, server, procotol, destCharset, run_time, packQueue)
	}
	fmt.Println("the system is already start all services.")
	Log(LL_INFO, "the system is already start all services")
	time.Sleep(2 * time.Second)
	wgDay.Done()
	return
}

func (group *createProcessGroup) processTcpGroup(server string, procotol string, destCharset string, separator int, run_time float64, packQueue chan *PackageTcpData) {
	num := group.process_num
	wgDay.Add(num)
	for i := separator; i < separator+num; i++ {
		go group.packageProcTcp(i, server, procotol, destCharset, run_time, packQueue)
	}
	fmt.Println("the system is already start all services.")
	Log(LL_INFO, "the system is already start all services")
	time.Sleep(2 * time.Second)
	wgDay.Done()
	return
}

func mark(process_id int, make_list []int64, spend_time float64, total_spend time.Duration) {
	if spend_time < 1 {
		spend_time = 1
	}
	if 0 != make_list[0] || 0 != make_list[1] {
		Log(LL_DEBUG, "thread id:", process_id, ", write succ package num:", (float64(make_list[0]))/spend_time, "ps/s, write error package num:", (float64(make_list[1]))/spend_time, "ps/s")
	}
	if 0 != make_list[2] || 0 != make_list[3] {
		Log(LL_DEBUG, "thread id:", process_id, ", read succ package num:", (float64(make_list[2]))/spend_time, "ps/s, read error package num:", (float64(make_list[3]))/spend_time, "ps/s")
	}
	if 0 != make_list[4] || 0 != make_list[5] {
		Log(LL_DEBUG, "thread id:", process_id, ", reply succ package num:", (float64(make_list[4]))/spend_time, "ps/s, reply error package num:", (float64(make_list[5]))/spend_time, "ps/s")
	}
	if 0 != make_list[6] {
		Log(LL_DEBUG, "thread id:", process_id, ", total package num:", make_list[6], ", total succ num:", make_list[7], ", total error num:", make_list[6]-make_list[7], ", total spend time:", total_spend)
	}
	return
}

func convertCharset(process_id int, ip string, data []byte, buf []byte, charset string, dest bool) int {
	var ilen int = 0
	if "" == ip {
		Log(LL_ERROR, "thread id:", process_id, "ip is empty, please check it.")
		return ilen
	}
	if 0 == len(data) {
		Log(LL_ERROR, "thread id:", process_id, ", buf is empty, please check it.")
		return ilen
	}
	if "" == charset {
		Log(LL_ERROR, "thread id:", process_id, ", charset is empty, please check it.")
		return ilen
	}
	source_charset, _ := config.Charset_transfer[ip]
	if "" == source_charset {
		Log(LL_ERROR, "thread id:", process_id, ", there is not find ip:", ip, ", in map, please check it.")
		return ilen
	}
	var err error = nil
	if dest {
		_, ilen, err = iconv.Convert(data, buf, source_charset, charset)
		if nil != err || 0 == ilen {
			Log(LL_ERROR, "thread id:", process_id, ", iconv convert error:", err, ", please check it")
			return ilen
		}

	} else {
		_, ilen, err = iconv.Convert(data, buf, charset, source_charset)
		if nil != err || 0 == ilen {
			Log(LL_ERROR, "thread id:", process_id, ", iconv convert error:", err, ", please check it")
			return ilen
		}
	}
	return ilen
}

func (group *createProcessGroup) checkStopTime(process_id int, stop_time int) bool {
	t1 := time.Now()
	if (t1.Sub(group.start_time)) >= time.Duration(stop_time*1e9) {
		if 0 == stop_time {
			Log(LL_INFO, "thread id:", process_id, ", must be stoped immediately.")
		} else {
			Log(LL_INFO, "thread id:", process_id, ", this thread run time is over.")
		}
		return true
	}
	return false
}

func writeData(conn net.Conn, process_id int, buf []byte, server string, protocol string) (n int, err error) {
	err = nil
	if 0 == len(buf) {
		err = errors.New("write data is empty, please check it.")
		return
	}
	if nil == conn {
		err = errors.New("connect is nil, please check it.")
		return
	}
	n, err = conn.Write(buf)
	if err != nil {
		if err == io.EOF || err == syscall.EINVAL {
			Log(LL_ERROR, "thread id:", process_id, ", connect is already closed.")
		} else {
			Log(LL_ERROR, "thread id:", process_id, ", ", err, ", please check it.")
		}
	} else {
		Log(LL_DEBUG, "thread id:", process_id, ", write data success, write data len:", n)
	}
	return
}

func readData(conn net.Conn, buf []byte, process_id int, server string) (int, error) {
	var n int = 0
	var err error = nil
	if nil == conn {
		err = errors.New("net conn is nil, please check it.")
		return n, err
	}
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, err = conn.Read(buf)
	if err != nil {
		if err == io.EOF || err == syscall.EINVAL {
			Log(LL_ERROR, "thread id:", process_id, ", connect is already closed.")
			if conn != nil {
				conn.Close()
				conn = nil
			}
			return n, err
		} else {
			Log(LL_ERROR, "thread id:", process_id, ", ", err, ", please check it.")
		}
	}
	return n, err
}

func (group *createProcessGroup) replyPackageTcp(process_id int, conn net.Conn, buf []byte, destCharset string, ip string) (int, error) {
	var err error = nil
	var n int = 0
	if nil == conn {
		err = errors.New("net conn is nil, please check it.")
		return n, err
	}
	buff := make([]byte, MAX_BYTES_LEN)
	n = convertCharset(process_id, ip, buf, buff, destCharset, false)
	if 0 == n {
		Log(LL_ERROR, "thread id:", process_id, ", reply package iconv convert error:", err, ", please check it.")
		return n, err
	}

	n, err = conn.Write(buff[:n])
	if nil != err {
		if err == io.EOF || err == syscall.EINVAL {
			Log(LL_ERROR, "thread id:", process_id, ", connect is already closed, client ip:", conn.RemoteAddr().String())
			if nil != conn {
				conn.Close()
				conn = nil
			}
			err = nil
		} else {
			Log(LL_ERROR, "thread id:", process_id, ", reply package error:", err, ", client ip:", conn.RemoteAddr().String())
		}
	}
	return n, err
}

func (group *createProcessGroup) replyPackageUdp(process_id int, conn *net.UDPConn, addr *net.UDPAddr, buf []byte) (int, error) {
	var err error = nil
	var n int = 0
	if nil == conn {
		err = errors.New("net conn is nil, please check it.")
		return n, err
	}

	n, err = conn.WriteToUDP(buf, addr)
	if nil != err {
		if err == io.EOF || err == syscall.EINVAL {
			Log(LL_ERROR, "thread id:", process_id, ", connect is already closed, client ip:", conn.RemoteAddr().String())
			if nil != conn {
				conn.Close()
				conn = nil
			}
			err = nil
		} else {
			Log(LL_ERROR, "thread id:", process_id, ", reply package error:", err, ", client ip:", conn.RemoteAddr().String())
		}
	}
	return n, err
}

func connectServer(process_id int, server string, procotol string) net.Conn {
	// Log(LL_INFO, "thread id:", process_id, ", tcp server:", server, " new connect")
	t1 := time.Now()
	t2 := time.Now()
	max_reconn_time := stop_run_time / 2
	if max_reconn_time > 60 {
		max_reconn_time = 60
	}
	var conn net.Conn = nil
	var err error = nil
	for {
		conn, err = net.DialTimeout(procotol, server, 10*time.Second)
		if err != nil {
			t2 = time.Now()
			if t2.Sub(t1) <= time.Duration(max_reconn_time*1e9) {
				time.Sleep(3 * time.Second)
				continue
			}
			Log(LL_ERROR, "thread id:", process_id, ", tcp connect error:", err, ", please check it.")
		}
		break
	}
	return conn
}

func (group *createProcessGroup) packageProcTcp(process_id int, server string, protocol string, destCharset string, run_time float64, packQueue chan *PackageTcpData) {
	ilen := 0
	num := 0
	write_buf := make([]byte, MAX_BYTES_LEN)
	read_buf := make([]byte, MAX_BYTES_LEN)
	t1 := time.Now()
	t2 := time.Now()
	t3 := time.Now()
	var retry_num int = 0
	var chan_flag bool = false
	var err error = nil
	var rece_data *PackageTcpData = nil
	var mark_list []int64 = make([]int64, 8)
	var conn net.Conn = connectServer(process_id, server, protocol)
	if conn == nil {
		Log(LL_ERROR, "thread id:", process_id, ", conn server error, please check log.")
		goto Result
	}
	if 0 == stop_run_time {
		metux.Lock()
		stop_run_time = int(60 * run_time)
		metux.Unlock()
	}
	for {
		select {
		case rece_data, chan_flag = <-packQueue:
			if nil == rece_data || !chan_flag {
				Log(LL_ERROR, "thread id:", process_id, ", channel packqueue is closed.")
				goto Result
			}
			break
		default:
			if group.checkStopTime(process_id, stop_run_time) {
				break
			}
			time.Sleep(1)
			continue
		}

		// mark
		retry_num = 0
		t2 = time.Now()
		if config.Log_level >= LL_DEBUG && (t2.Sub(t1)) >= 6*1e9 {
			mark(process_id, mark_list, 6.0, t2.Sub(t3))
			t1 = time.Now()
			for ilen = 0; ilen < len(mark_list)-2; ilen++ {
				mark_list[ilen] = 0
			}
			ilen = 0
		}
		// conn tcp server
		if group.checkStopTime(process_id, stop_run_time) {
			break
		}

		mark_list[6]++
		ilen = convertCharset(process_id, rece_data.ip, rece_data.data, write_buf, destCharset, true)
		if 0 == ilen {
			continue
		}

		for retry_num < 3 {
			// write data
			ilen, err = writeData(conn, process_id, write_buf[:ilen], server, protocol)
			if err != nil {
				mark_list[1]++
				if 0 == ilen {
					if nil != conn {
						conn.Close()
						conn = nil
					}
					conn = connectServer(process_id, server, protocol)
				}
			}
			mark_list[0]++

			retry_num++
			// read data
			num, err = readData(conn, read_buf, process_id, server)
			if err != nil {
				mark_list[3]++
				if TCP_RESET_BY_PEER == checkConnErrorType(err) {
					conn.Close()
					conn = nil
				}
				conn = connectServer(process_id, server, protocol)
				continue
			}
			if checkKeyInSlice(string(read_buf[:num])) {
				continue
			}
			break
		}
		if err != nil {
			continue
		}
		mark_list[2]++
		_, err = group.replyPackageTcp(process_id, rece_data.conn, read_buf[:num], destCharset, rece_data.ip)
		if nil != err {
			mark_list[5]++
			continue
		}
		mark_list[4]++
		mark_list[7]++
	}

Result:
	if nil != conn {
		conn.Close()
		conn = nil
	}
	if 0 != mark_list[6] {
		group.mu.Lock()
		group.total_num += mark_list[6]
		group.mu.Unlock()
	}
	t2 = time.Now()
	mark(process_id, mark_list, float64(t2.Sub(t1)/1e9), t2.Sub(t3))
	Log(LL_DEBUG, "thread id:", process_id, ", the current thread will be stoped.")
	wgDay.Done()
	return
}

func checkConnErrorType(err error) int {
	if nil == err || "" == err.Error() {
		return SUCCESS
	}
	if strings.Contains(err.Error(), "connection reset by peer") {
		return TCP_RESET_BY_PEER
	} else if strings.Contains(err.Error(), "broken pipe") {
		return TCP_WRITE_BROKEN_PIPE
	} else if strings.Contains(err.Error(), "i/o timeout") {
		return TCP_READ_TIMEOUT
	}
	return TCP_UNKNOWN_ERROR
}

func checkKeyInSlice(key string) bool {
	if "" == key {
		return true
	}
	if len(config.Remote_key_words_filter) > 0 {
		for _, value := range config.Remote_key_words_filter {
			if strings.Contains(key, value) {
				return true
			}
		}
	}
	return false
}

func (group *createProcessGroup) packageProcUdp(process_id int, server string, protocol string, destCharset string, run_time float64, packQueue chan *PackageUdpData) {
	ilen := 0
	num := 0
	read_buf := make([]byte, MAX_BYTES_LEN)
	t1 := time.Now()
	t2 := time.Now()
	t3 := time.Now()
	var chan_flag bool = false
	var err error = nil
	var rece_data *PackageUdpData = nil
	var mark_list []int64 = make([]int64, 8)
	var conn net.Conn = connectServer(process_id, server, protocol)
	if conn == nil {
		Log(LL_ERROR, "thread id:", process_id, ", conn server error, please check log.")
		goto Result
	}
	if 0 == stop_run_time {
		metux.Lock()
		stop_run_time = int(60 * run_time)
		metux.Unlock()
	}
	for {
		select {
		case rece_data, chan_flag = <-packQueue:
			if nil == rece_data || !chan_flag {
				Log(LL_ERROR, "thread id:", process_id, ", channel packqueue is closed.")
				goto Result
			}
			break
		default:
			if group.checkStopTime(process_id, stop_run_time) {
				break
			}
			time.Sleep(1)
			continue
		}
		if nil == rece_data.conn || nil == rece_data.addr {
			Log(LL_ERROR, "thread id:", process_id, ", recv package para is nil, please check it.")
			goto Result
		}
		// mark
		t2 = time.Now()
		if config.Log_level >= LL_DEBUG && (t2.Sub(t1)) >= 6*1e9 {
			mark(process_id, mark_list, 6.0, t2.Sub(t3))
			t1 = time.Now()
			for ilen = 0; ilen < len(mark_list)-2; ilen++ {
				mark_list[ilen] = 0
			}
			ilen = 0
		}
		// conn tcp server
		if group.checkStopTime(process_id, stop_run_time) {
			break
		}
		mark_list[6]++

		for {
			// write data
			ilen, err = writeData(conn, process_id, rece_data.data, server, protocol)
			if err != nil {
				mark_list[1]++
				if 0 == ilen {
					if nil != conn {
						conn.Close()

						conn = nil
					}
					conn = connectServer(process_id, server, protocol)
				}
			}
			mark_list[0]++
			// read data
			num, err = readData(conn, read_buf, process_id, server)
			if err != nil {
				mark_list[3]++
				continue
			}
			if "no response" == string(read_buf[:num]) {
				continue
			}
			break
		}
		if err != nil {
			continue
		}
		mark_list[2]++
		_, err = group.replyPackageUdp(process_id, rece_data.conn, rece_data.addr, read_buf[:num])
		if nil != err {
			mark_list[5]++
			continue
		}
		mark_list[4]++
		mark_list[7]++
	}

Result:
	if nil != conn {
		conn.Close()
		conn = nil
	}
	if 0 != mark_list[6] {
		group.mu.Lock()
		group.total_num += mark_list[6]
		group.mu.Unlock()
	}
	t2 = time.Now()
	mark(process_id, mark_list, float64(t2.Sub(t1))/1e9, t2.Sub(t3))
	Log(LL_DEBUG, "thread id:", process_id, ", the current thread will be stoped.")
	wgDay.Done()
	return
}
