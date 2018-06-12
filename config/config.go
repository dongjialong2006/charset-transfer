// Edit by Dongcf at 2016-10-13

// Package parsing_config is used to deal config file, the file is xml format.
package config

// parsing xml file
import (
	"charset-transfer-tool/log"
	"context"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/fsnotify.v1"
)

var logger = log.New("config")
var rw sync.RWMutex
var config *Configure = nil

type Configure struct {
	XmlName xml.Name `xml:"configure"`
	Log     *LogInfo `xml:"log"`
	Local   *Local   `xml:"local"`
	Remote  *Remote  `xml:"remote"`
	Servers *Servers `xml:"servers"`
}

type LogInfo struct {
	Name  string `xml:"name"`
	Level int    `xml:"level"`
}

type Local struct {
	IP       string `xml:"ip"`
	Port     string `xml:"port"`
	Protocol string `xml:"protocol"`
}

type Remote struct {
	IP             string `xml:"ip"`
	Port           string `xml:"port"`
	Protocol       string `xml:"protocol"`
	Charset        string `xml:"charset"`
	KeyWordsFilter string `xml:"key_words_filter"`
}

type Servers struct {
	Servers []string `xml:"server"`
}

func GetConfig() *Configure {
	rw.RLock()
	defer rw.RUnlock()
	return config
}

func GetSourceAddrCharset(addr string) (string, bool) {
	rw.RLock()
	defer rw.RUnlock()
	if nil == config.Servers {
		return "", false
	}

	for _, server := range config.Servers.Servers {
		if strings.HasPrefix(server, addr) {
			return strings.Split(server, ",")[1], true
		}
	}

	return "", false
}

func Watch(ctx context.Context, path string) error {
	if "" == path {
		return fmt.Errorf("path is empty.")
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	logger.Infof("config file name:%s.", path)
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	rw.Lock()
	config, err = loadFileConfig(file.Name())
	rw.Unlock()
	if nil != err {
		return err
	}

	go func() {
		defer watcher.Close()
		defer file.Close()
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-watcher.Events:
				if event.Name == file.Name() && (event.Op == fsnotify.Create || event.Op == fsnotify.Write) {
					rw.Lock()
					config, err = loadFileConfig(file.Name())
					rw.Unlock()
					if nil != err {
						logger.Error(err)
						return
					}
				}
			case err = <-watcher.Errors:
				logger.Errorf("watcher event err:%v.", err)
				return
			}
		}
	}()

	return watcher.Add(filepath.Dir(path))
}

func loadFileConfig(path string) (*Configure, error) {
	cfg := &Configure{}
	file, err := os.Open(path)
	if err != nil && os.IsNotExist(err) {
		return nil, err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	if err = xml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
