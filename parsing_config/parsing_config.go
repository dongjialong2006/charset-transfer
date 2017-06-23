// Edit by Dongcf at 2016-10-13

// Package parsing_config is used to deal config file, the file is xml format.
package parsing_config

// parsing xml file
import (
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	_ "math/rand"
	"os"
	"reflect"
	_ "strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	Log_file_name           string
	Log_level               int
	Local_ip                string
	Local_port              string
	Local_protocol          string
	Local_charset           string
	Remote_ip               string
	Remote_port             []string
	Remote_protocol         string
	Remote_charset          string
	Remote_thread_num       int
	Remote_key_words_filter []string
	Charset_transfer        map[string]string
	config_file_path        string
	file_modify_time        int64
	mutex                   sync.Mutex
	update_chan             chan int
}

const (
	CONFIG_XML_SUCCESS = iota + 1
	CONFIG_XML_START
	CONFIG_XML_ERROR
	SYSTEM_ALL_OLD_SERVICES_STOP
	SYSTEM_SHUT_DOWN
)

var config *Config = nil

func GetConfigInfo() *Config {
	return config
}

type Configure struct {
	XmlName          xml.Name `xml:"configure"`
	GlobalSection    Global   `xml:"global"`
	LocalServerInfo  Local    `xml:"local"`
	RemoteServerInfo Remote   `xml:"remote"`
	CharsetInfo      Charset  `xml:"charset"`
}

type Global struct {
	XmlName   xml.Name `xml:"global"`
	Log_name  string   `xml:"log_name"`
	Log_level int      `xml:"log_level"`
}

type Local struct {
	XmlName        xml.Name `xml:"local"`
	Local_ip       string   `xml:"local_ip"`
	Local_port     string   `xml:"local_port"`
	Local_protocol string   `xml:"local_protocol"`
}

type Remote struct {
	XmlName                 xml.Name `xml:"remote"`
	Remote_ip               string   `xml:"remote_ip"`
	Remote_port             string   `xml:"remote_port"`
	Remote_protocol         string   `xml:"remote_protocol"`
	Remote_charset          string   `xml:"remote_charset"`
	Remote_thread_num       int      `xml:"remote_thread_num"`
	Remote_key_words_filter string   `xml:"remote_key_words_filter"`
}

type Charset struct {
	XmlName          xml.Name `xml:"charset"`
	Charset_transfer string   `xml:"charset_transfer"`
}

func ParsingXml(filename string, ch chan int) error {
	var err error = nil
	if filename == "" || 0 == len(filename) {
		err = errors.New("filename is empty, please check it.")
		return err
	}
	var errinfo string = ""
	file, err := os.Open(filename)
	if err != nil && os.IsNotExist(err) {
		errinfo = fmt.Sprintf("filename:%s is not exist, error info:%s, please check it.", filename, err.Error())
		err = errors.New(errinfo)
		return err
	}
	defer file.Close()
	first_ini_flag := false
	if nil == config {
		config = &Config{config_file_path: filename, Log_level: 5}
		first_ini_flag = true
	}
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	system_config := Configure{}
	err = xml.Unmarshal(data, &system_config)
	if err != nil {
		return err
	}
	config.mutex.Lock()
	global_config := system_config.GlobalSection
	log_name := global_config.Log_name
	log_name = strings.Replace(log_name, " ", "", -1)
	if log_name == "" {
		errinfo = fmt.Sprintf("the config section:log_name is empty, please check it")
		err = errors.New(errinfo)
		config.mutex.Unlock()
		return err
	}
	if log_name != config.Log_file_name {
		config.Log_file_name = log_name
	}
	if config.Log_level != global_config.Log_level {
		config.Log_level = global_config.Log_level
	}

	localserverxml := system_config.LocalServerInfo
	local_ip := localserverxml.Local_ip
	local_ip = strings.Replace(local_ip, " ", "", -1)
	if local_ip == "" {
		errinfo = fmt.Sprintf("the config section:local_ip is empty, please check it")
		err = errors.New(errinfo)
		config.mutex.Unlock()
		return err
	}
	if local_ip != config.Local_ip {
		config.Local_ip = local_ip
	}
	local_port := localserverxml.Local_port
	local_port = strings.Replace(local_port, " ", "", -1)
	if local_port == "" {
		errinfo = fmt.Sprintf("the config section:local_port is empty, please check it")
		err = errors.New(errinfo)
		config.mutex.Unlock()
		return err
	}
	if local_port != config.Local_port {
		config.Local_port = local_port
	}

	local_protocol := localserverxml.Local_protocol
	local_protocol = strings.Replace(local_protocol, " ", "", -1)
	if local_protocol == "" {
		errinfo = fmt.Sprintf("the config section:local_protocol is empty, please check it")
		err = errors.New(errinfo)
		config.mutex.Unlock()
		return err
	}
	if local_protocol != config.Local_protocol {
		config.Local_protocol = local_protocol
	}

	remoteserverxml := system_config.RemoteServerInfo
	remote_ip := remoteserverxml.Remote_ip
	remote_ip = strings.Replace(remote_ip, " ", "", -1)
	if remote_ip == "" {
		errinfo = fmt.Sprintf("the config section:remote_ip is empty, please check it")
		err = errors.New(errinfo)
		config.mutex.Unlock()
		return err
	}
	if remote_ip != config.Remote_ip {
		config.Remote_ip = remote_ip
	}

	remote_port := remoteserverxml.Remote_port
	remote_port = strings.Replace(remote_port, " ", "", -1)
	if remote_port == "" {
		errinfo = fmt.Sprintf("the config section:remote_port is empty, please check it")
		err = errors.New(errinfo)
		config.mutex.Unlock()
		return err
	}
	config.Remote_port = make([]string, 0)
	if strings.Contains(remote_port, ",") {
		config.Remote_port = strings.Split(remote_port, ",")
	} else {
		config.Remote_port = append(config.Remote_port, remote_port)
	}

	remote_protocol := remoteserverxml.Remote_protocol
	remote_protocol = strings.Replace(remote_protocol, " ", "", -1)
	if remote_protocol == "" {
		errinfo = fmt.Sprintf("the config section:remote_protocol is empty, please check it")
		err = errors.New(errinfo)
		config.mutex.Unlock()
		return err
	}
	if remote_protocol != config.Remote_protocol {
		config.Remote_protocol = remote_protocol
	}
	remote_charset := remoteserverxml.Remote_charset
	remote_charset = strings.Replace(remote_charset, " ", "", -1)
	if remote_charset == " " {
		errinfo = fmt.Sprintf("the config section:remote_charset is empty, please check it")
		err = errors.New(errinfo)
		config.mutex.Unlock()
		return err
	}
	if remote_charset != config.Remote_charset {
		config.Remote_charset = remote_charset
	}
	remote_thread_num := remoteserverxml.Remote_thread_num
	if remote_thread_num <= 0 {
		remote_thread_num = 10
	}
	if remote_thread_num != config.Remote_thread_num {
		config.Remote_thread_num = remote_thread_num
	}
	config.Remote_key_words_filter = make([]string, 0)
	remote_key_words_filter := remoteserverxml.Remote_key_words_filter
	if "" != remote_key_words_filter {
		if strings.Contains(remote_key_words_filter, ",") {
			config.Remote_key_words_filter = strings.Split(remote_key_words_filter, ",")
		} else {
			config.Remote_key_words_filter = append(config.Remote_key_words_filter, remote_key_words_filter)
		}
	}
	charset_info := system_config.CharsetInfo
	charset_transfer := charset_info.Charset_transfer
	charset_transfer = strings.Replace(charset_transfer, " ", "", -1)
	if charset_transfer == "" {
		errinfo = fmt.Sprintf("the config section:charset_transfer is empty, please check it")
		err = errors.New(errinfo)
		config.mutex.Unlock()
		return err
	}
	config.Charset_transfer = make(map[string]string)
	if strings.Contains(charset_transfer, ";") {
		sli := strings.Split(charset_transfer, ";")
		for _, value := range sli {
			if strings.Contains(value, ",") {
				if strings.Count(value, ",") > 1 {
					err = errors.New("charset_transfer config format error, please check it")
					config.mutex.Unlock()
					return err
				}
				sli2 := strings.Split(value, ",")
				config.Charset_transfer[sli2[0]] = sli2[1]
			} else {
				err = errors.New("charset_transfer config error, please check it")
				config.mutex.Unlock()
				return err
			}
		}
	} else {
		if strings.Contains(charset_transfer, ",") {
			if strings.Count(charset_transfer, ",") > 1 {
				err = errors.New("charset_transfer config format error, please check it")
				config.mutex.Unlock()
				return err
			}
			sli1 := strings.Split(charset_transfer, ",")
			config.Charset_transfer[sli1[0]] = sli1[1]
		} else {
			err = errors.New("charset_transfer config error, please check it")
			config.mutex.Unlock()
			return err
		}
	}

	if nil == config.update_chan {
		config.update_chan = ch
		config.update_chan <- CONFIG_XML_SUCCESS
	}
	if fileInfo, err := os.Stat(config.config_file_path); err == nil {
		config.file_modify_time = reflect.ValueOf(fileInfo.Sys()).Elem().FieldByName("Mtim").Field(0).Int()
	}
	config.mutex.Unlock()
	if first_ini_flag {
		go ConfigFileAutoUpdateStart()
	}
	// dump
	fmt.Println("-------------------------------------")
	fmt.Println("-------------------------------------")
	fmt.Println("parsing xml file is completed, the config info:")
	fmt.Println("-------------------------------------")
	fmt.Println("-------------------------------------")
	fmt.Println("log_path:", log_name)
	fmt.Println("log_level:", global_config.Log_level)
	fmt.Println("local_ip:", local_ip)
	fmt.Println("local_port:", local_port)
	fmt.Println("local_protcocol:", local_protocol)
	fmt.Println("remote_ip:", remote_ip)
	fmt.Println("remote_port:", remote_port)
	fmt.Println("remote_protcocol:", remote_protocol)
	fmt.Println("remote_charset:", remote_charset)
	fmt.Println("remote_thread_num:", remote_thread_num)
	fmt.Println("remote_key_words_filter:", remote_key_words_filter)
	fmt.Println("charset_transfer", charset_transfer)
	fmt.Println("-------------------------------------")
	fmt.Println("-------------------------------------")
	return nil
}

func ConfigFileAutoUpdateStop() {
	config.mutex.Lock()
	config.config_file_path = ""
	if nil != config.update_chan {
		close(config.update_chan)
		config.update_chan = nil
	}
	config.mutex.Unlock()
	time.Sleep(3 * time.Second)
}

func ConfigFileAutoUpdateStart() {
	var system_status int = 0
	for {
		if "" != config.config_file_path {
			if fileInfo, err := os.Stat(config.config_file_path); err == nil {
				mtim := reflect.ValueOf(fileInfo.Sys()).Elem().FieldByName("Mtim").Field(0).Int()
				if config.file_modify_time != mtim {
					fmt.Println("\n...................................................")
					fmt.Println("...................................................")
					fmt.Println("the system config file update begin.............")
					config.update_chan <- int(CONFIG_XML_START)
					fmt.Println("the system is waiting all old services stoped.............")
					for {
						time.Sleep(1e8)
						system_status = <-config.update_chan
						if int(SYSTEM_ALL_OLD_SERVICES_STOP) == system_status {
							fmt.Println("the system is updating config file start.............")
							err = ParsingXml(config.config_file_path, nil)
							if nil == err {
								config.update_chan <- int(CONFIG_XML_SUCCESS)
								fmt.Println("the system is updating config file end.............")
								break
							} else {
								config.update_chan <- int(CONFIG_XML_ERROR)
								fmt.Println("the system is updating config file error.............")
							}
						}
					}
					fmt.Println("...................................................")
					fmt.Println("...................................................\n")
				}
			}
		} else {
			break
		}
		time.Sleep(3 * 1e9)
	}
	return
}
