package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

func ReadFile(name string) ([]byte, error) {
	if "" == name {
		return nil, fmt.Errorf("file name is empty.")
	}

	_, err := os.Stat(name)
	if nil != err {
		return nil, err
	}

	return ioutil.ReadFile(name)
}

func GenerateConfigFile(path string) (string, error) {
	if "" == path {
		return "", fmt.Errorf("path is empty.")
	}
	var newPath string = ""
	pos := strings.LastIndex(path, "/")
	if -1 == pos {
		if '.' != path[0] {
			newPath = "." + path
		}
	} else {
		temp := path[pos+1:]
		if '.' != temp[0] {
			newPath = path[:pos+1] + "." + temp
		} else {
			newPath = path
		}
	}

	file, err := os.Open(path)
	if nil != err {
		return "", err
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if nil != err {
		return "", err
	}

	value := strings.Replace(string(data), " ", "", -1)
	value = strings.Replace(value, "deamon=true", "", -1)
	// fmt.Println("child config path:", newPath)
	err = ioutil.WriteFile(newPath, []byte(value), 0777)
	if nil != err {
		return "", err
	}
	return newPath, nil
}

func WriteFile(path string, data []byte) error {
	var dir string = ""
	pos := strings.LastIndex(path, "/")
	if -1 == pos {
		dir = "./" + path
	} else {
		dir = path[:pos+1]
	}

	_, err := os.Stat(dir)
	if nil != err {
		if os.IsNotExist(err) {
			err = os.Mkdir(dir, os.ModePerm)
			if nil != err {
				return err
			}
		} else {
			return err
		}
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if nil != err {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	return err
}

func CreatePath(path string) error {
	if "" == path {
		return fmt.Errorf("path is empty.")
	}

	pos := strings.LastIndex(path, "/")
	if -1 == pos {
		return nil
	}
	path = path[:pos]
	_, err := os.Stat(path)
	if nil != err {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, os.ModePerm)
		}
	}

	return err
}
