// Edit by Dongcf at 2017-01-06

// Package log_system is used to deal log file.
package log_system

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// out interface
var Log_Handle LogHandle

// inner para for reduce new objects to create
var curr_time time.Time
var once_write sync.Once
var once_update sync.Once
var buff_create bytes.Buffer

const (
	LL_ERROR = iota + 1
	LL_WARN
	LL_INFO
	LL_DEBUG
	LL_ALL
)

const (
	BY_THIRTY_MINUTES = iota + 1
	BY_ONE_HOURS
	BY_TWO_HOURS
	BY_SIX_HOURS
	BY_DAY
	BY_ALL
)

type LogHandle struct {
	log_name             string
	log_file_name        string
	log_dynamic_name     string
	log_source_name      string
	log_file_handle      *os.File
	log_level            int
	log_policy           int
	log_chan_info        chan string
	log_file_modify_time int64
	log_stop_lable       bool
	log_rwMutex          sync.RWMutex
}

type LogConfigure struct {
	XmlName    xml.Name  `xml:"configure"`
	LogSection LogConfig `xml:"log"`
}

type LogConfig struct {
	XmlName    xml.Name `xml:"log"`
	Log_name   string   `xml:"log_name"`
	Log_level  int      `xml:"log_level"`
	Log_policy int      `xml:"log_policy"`
}

func (log *LogHandle) createFileHandle() *os.File {
	var err error = nil
	curr_time = time.Now().In(time.FixedZone("CST", 28800))
	// year := curr_time.Year()
	// month := curr_time.Month()
	// day := curr_time.Day()
	// hour := curr_time.Hour()
	// minute := hour := curr_time.Minute()
	// switch log.log_policy {
	// case BY_DAY:
	buff_create.Reset()
	buff_create.WriteString(log.log_name)
	buff_create.WriteString("_")
	buff_create.WriteString(curr_time.Format("20060102"))
	log.log_dynamic_name = buff_create.String()
	// }
	if log.log_dynamic_name != log.log_source_name {
		if log.log_file_handle != nil {
			var value string = ""
			for {
				if (len(log.log_chan_info)) > 0 {
					value = <-log.log_chan_info
					if strings.Contains(value, curr_time.AddDate(0, 0, -1).Format("20060102")) {
						_, err = log.log_file_handle.WriteString(value)
						if err != nil {
							fmt.Println(err.Error())
						}
					} else {
						break
					}
				} else {
					break
				}
			}
			log.log_file_handle.Close()
			log.log_file_handle = nil
		}
		log.log_rwMutex.Lock()
		log.log_file_handle, err = os.OpenFile(log.log_dynamic_name, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
		if err != nil {
			fmt.Println("Faild to create the outfile:", log.log_dynamic_name, ", error info:", err.Error(), ".")
			return nil
		}
		log.log_source_name = log.log_dynamic_name
		log.log_rwMutex.Unlock()
	} else {
		_, err := os.Stat(log.log_dynamic_name)
		if err != nil {
			if os.IsNotExist(err) {
				log.log_file_handle, err = os.OpenFile(log.log_dynamic_name, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
				if err != nil {
					fmt.Println("Faild to create the outfile:", log.log_dynamic_name, ", error info:", err.Error(), ".")
					return nil
				}
			}
		}
		if nil == log.log_file_handle {
			log.log_file_handle, err = os.OpenFile(log.log_dynamic_name, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
			if err != nil {
				fmt.Println("Faild to create the outfile:", log.log_dynamic_name, ", error info:", err.Error(), ".")
				return nil
			}
		}
	}
	return log.log_file_handle
}

func (log *LogHandle) writeLog() {
	var start_succ_chan chan bool = make(chan bool)
	go func() {
		var log_handle *os.File = nil
		var value string = ""
		var num int = 0
		var ok bool = true
		var err error = nil
		if nil != log.log_chan_info {
			start_succ_chan <- true
			for {
				value, ok = <-log.log_chan_info
				if !ok && 0 == len(log.log_chan_info) {
					log.log_rwMutex.Lock()
					log.log_stop_lable = true
					log.log_rwMutex.Unlock()
					break
				}
				log_handle = log.createFileHandle()
				if value != "" {
					num, err = log_handle.WriteString(value)
					if 0 == num || err != nil {
						if 0 == num {
							log_handle = log.createFileHandle()
							_, err = log_handle.WriteString(value)
							if err != nil {
								fmt.Println(err.Error())
							}
						} else {
							fmt.Println(err.Error())
						}
					}
				} else {
					break
				}
			}
		}
	}()
	<-start_succ_chan
	return
}

func (log *LogHandle) LogSystemStart(file_name string) error {
	var errinfo string = ""
	file_name = strings.Trim(file_name, " ")
	if file_name != log.log_file_name {
		log.log_file_name = file_name
	}
	if "" == log.log_file_name {
		file_name = "../etc/log.xml"
	}
	fileInfo, err := os.Stat(file_name)
	if err != nil {
		if os.IsNotExist(err) {
			log.log_file_name = ""
			err = nil
			// fmt.Println("log config file is not exist, the system will use default config value.")
			// fmt.Println("check file name:", log.log_file_name, " error, err info:", err.Error())
		} else {
			fmt.Println(err)
			return err
		}
	} else {
		alter_time := reflect.ValueOf(fileInfo.Sys()).Elem().FieldByName("Mtim").Field(0).Int()
		if log.log_file_modify_time != alter_time {
			log.log_file_modify_time = alter_time
			log.log_file_name = file_name
		} else {
			return err
		}
	}
	if "" != log.log_file_name {
		fmt.Println("log config 'log_file_name':", log.log_file_name)
		file, err := os.Open(log.log_file_name)
		if err != nil && os.IsNotExist(err) {
			errinfo = fmt.Sprintf("file name:%s is not exist, error info:%s, please check it.", log.log_file_name, err.Error())
			err = errors.New(errinfo)
			return err
		}
		defer file.Close()
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}
		system_config := LogConfigure{}
		err = xml.Unmarshal(data, &system_config)
		if err != nil {
			return err
		}
		log_name := system_config.LogSection.Log_name
		log_name = strings.Trim(log_name, " ")
		if "" == log_name {
			if "" == log.log_name {
				log.log_name = "./log.log"
				fmt.Println("log config 'log_name':", log.log_name)
			}
		} else {
			if log_name != log.log_name {
				log.log_rwMutex.Lock()
				if "" != log.log_name {
					log.log_file_handle.Close()
					log.log_file_handle = nil
				}
				log.log_name = log_name
				fmt.Println("log config 'log_name':", log.log_name)
				log.log_rwMutex.Unlock()
			}
		}

		if system_config.LogSection.Log_level < LL_ERROR || system_config.LogSection.Log_level > LL_ALL {
			if log.log_level >= LL_ERROR && log.log_level <= LL_ALL {
				fmt.Println("log level is valide data between 1 and 5, you curr config value is ", system_config.LogSection.Log_level)
			} else {
				if 0 == log.log_level {
					log.log_level = LL_DEBUG
					fmt.Println("log config 'log_level':", log.log_level)
				}
			}
		} else {
			if log.log_level != system_config.LogSection.Log_level {
				log.log_rwMutex.Lock()
				log.log_level = system_config.LogSection.Log_level
				fmt.Println("log config 'log_level':", log.log_level)
				log.log_rwMutex.Unlock()
			}
		}
		if system_config.LogSection.Log_policy < BY_THIRTY_MINUTES || system_config.LogSection.Log_policy > BY_ALL {
			if 0 == log.log_policy {
				log.log_policy = BY_DAY
				fmt.Println("log config 'log_policy':", log.log_policy)
			}
		} else {
			if log.log_policy != system_config.LogSection.Log_policy {
				log.log_rwMutex.Lock()
				log.log_policy = system_config.LogSection.Log_policy
				fmt.Println("log config 'log_policy':", log.log_policy)
				log.log_rwMutex.Unlock()
			}
		}
	} else {
		if "" == log.log_name {
			log.log_rwMutex.Lock()
			if nil != log.log_file_handle {
				log.log_file_handle.Close()
				log.log_file_handle = nil
			}
			log.log_name = "./log.log"
			fmt.Println("log config 'log_name':", log.log_name, ", use default value.")
			log.log_rwMutex.Unlock()
		}
		if log.log_level <= LL_ERROR || log.log_level >= LL_ALL {
			log.log_rwMutex.Lock()
			log.log_level = LL_DEBUG
			fmt.Println("log config 'log_level':", log.log_level, ", use default value.")
			log.log_rwMutex.Unlock()
		}
		if log.log_policy < BY_THIRTY_MINUTES || log.log_policy > BY_ALL {
			log.log_rwMutex.Lock()
			log.log_policy = BY_DAY
			fmt.Println("log config 'log_policy':", log.log_policy, ", use default value.")
			log.log_rwMutex.Unlock()
		}
	}
	if nil == log.log_chan_info {
		log.log_chan_info = make(chan string, 1000)
	}
	once_write.Do(log.writeLog)
	once_update.Do(log.autoUpdateFile)
	return nil
}

func (log *LogHandle) autoUpdateFile() {
	go func() {
		var err error = nil
		for {
			if log.log_stop_lable {
				break
			}
			err = log.LogSystemStart(log.log_file_name)
			if err != nil {
				Log(LL_ERROR, "log init error, file name:", log.log_file_name, ", err info:", err.Error())
			}
			time.Sleep(2 * 1e9)
		}
	}()
	return
}

func Log(level int, args ...interface{}) {
	if level > Log_Handle.log_level {
		return
	}
	file_ptr, file_name, file_line, _ := runtime.Caller(1)
	func_name := runtime.FuncForPC(file_ptr).Name()
	buff := new(bytes.Buffer)
	buff.WriteString(time.Now().In(time.FixedZone("CST", 28800)).Format("2006-01-02 15:04:05.00000"))
	buff.WriteString(" <")
	buff.WriteString(Log_Handle.logLevel(level))
	buff.WriteString("> ")
	buff.WriteString(file_name)
	buff.WriteString(":")
	buff.WriteString(func_name)
	buff.WriteString(":")
	buff.WriteString(strconv.Itoa(file_line))
	buff.WriteString(", ")
	for _, arg := range args {
		switch arg.(type) {
		case int:
			buff.WriteString(strconv.Itoa(arg.(int)))
		case int32:
			buff.WriteString(strconv.FormatInt(arg.(int64), 10))
		case int64:
			buff.WriteString(strconv.FormatInt(arg.(int64), 10))
		case time.Duration:
			buff.WriteString(fmt.Sprintf("%v", arg))
		case string:
			buff.WriteString(arg.(string))
		case float32:
			buff.WriteString(strconv.FormatFloat(arg.(float64), 'f', 8, 32))
		case float64:
			buff.WriteString(strconv.FormatFloat(arg.(float64), 'f', 8, 64))
		default:
			buff.WriteString(fmt.Sprintf("%v", arg))
		}
	}
	buff.WriteString(".\n")
	if Log_Handle.log_chan_info != nil {
		Log_Handle.log_chan_info <- buff.String()
	} else {
		fmt.Println(buff.String())
	}
	return
}

func (log *LogHandle) logLevel(level int) string {
	if 1 == level {
		return "LL_ERROR"
	} else if 2 == level {
		return "LL_WARN"
	} else if 3 == level {
		return "LL_INFO"
	} else if 4 == level {
		return "LL_DEBUG"
	} else if 5 == level {
		return "LL_ALL"
	} else {
		return ""
	}
}

func (log *LogHandle) LogSystemStop() {
	if nil != log.log_chan_info {
		log.log_rwMutex.Lock()
		close(log.log_chan_info)
		log.log_chan_info = nil
		log.log_rwMutex.Unlock()
	}
	for {
		if log.log_stop_lable {
			break
		}
		time.Sleep(1e8)
	}
	if nil != log.log_file_handle {
		log.log_rwMutex.Lock()
		log.log_file_handle.Close()
		log.log_file_handle = nil
		log.log_rwMutex.Unlock()
	}
}
