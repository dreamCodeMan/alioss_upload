package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const (
	ALIYUN_OAUTH_URL string = "http://oss-demo.aliyuncs.com/oss-h5-upload-js-php-callback/php/get.php"
)

//设置超时
var timeout time.Duration = time.Second * 30
var client = &http.Client{
	Transport: &http.Transport{
		Dial: func(netw, addr string) (net.Conn, error) {
			conn, err := net.DialTimeout(netw, addr, timeout)
			if err != nil {
				return nil, err
			}
			conn.SetDeadline(time.Now().Add(timeout))
			return conn, nil
		},
		ResponseHeaderTimeout: timeout,
	},
}

type AuthInfo struct {
	AccessId  string `json:"accessid"`
	Host      string `json:"host"`
	Policy    string `json:"policy"`
	Signature string `json:"signature"`
	Expire    int    `json:"expire"`
	Callback  string `json:"callback"`
	Dir       string `json:"dir"`
}

func exist_file(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func auth() (AuthInfo, error) {
	info := new(AuthInfo)
	resp, err := client.Get(ALIYUN_OAUTH_URL)
	if err != nil {
		return *info, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return *info, err
	}
	err = json.Unmarshal(body, &info)
	if err != nil {
		return *info, err
	}
	return *info, nil

}

func build_post_body(field_dict map[string]string, boundary string) string {
	var post_body string

	for k, v := range field_dict {
		if k != "content" && k != "content-type" {
			post_body += fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"%s\"\r\n\r\n%s\r\n", boundary, k, v)
		}
	}

	post_body += fmt.Sprintf("--%s\r\nContent-Disposition: form-data; name=\"file\"; filename=\"%s\"\r\nContent-Type: %s\r\n\r\n%s", boundary, field_dict["key"], field_dict["content-type"], field_dict["content"])
	post_body += fmt.Sprintf("\r\n--%s--\r\n", boundary)

	return post_body
}

func read_file(filename string) (content []byte, err error) {
	var fileByte []byte
	if strings.HasPrefix(filename, "http") {
		resp, err := client.Get(filename)
		if err != nil {
			return content, err
		}
		if resp.StatusCode != http.StatusOK {
			return content, fmt.Errorf("error:status code is %d", resp.StatusCode)
		}
		defer resp.Body.Close()
		fileByte, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			return content, err
		}
	} else {
		if !exist_file(filename) {
			return content, fmt.Errorf("%s not find", filename)
		}

		f, err := os.Open(filename)
		if err != nil {
			return content, err
		}
		defer f.Close()
		fileByte, err = ioutil.ReadAll(f)
		if err != nil {
			return content, err
		}
	}
	return fileByte, err
}

func get_file_name_suffix(file string) string {
	filenameWithSuffix := path.Base(file)
	fileSuffix := path.Ext(filenameWithSuffix)
	filenameOnly := strings.TrimSuffix(filenameWithSuffix, fileSuffix)
	return filenameOnly + fileSuffix
}

func main() {

	var filename string = "alioss.go"
	content, err := read_file(filename)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	info, _ := auth()
	field_dict := map[string]string{}
	field_dict["key"] = info.Dir + get_file_name_suffix(filename)
	field_dict["OSSAccessKeyId"] = info.AccessId
	field_dict["policy"] = info.Policy
	field_dict["success_action_status"] = "200"
	field_dict["Signature"] = info.Signature
	field_dict["callback"] = info.Callback
	field_dict["content"] = string(filename)
	//field_dict["content-type"] = "text/plain"
	field_dict["content-type"] = "application/octet-stream"

	boundary := strconv.Itoa(info.Expire)

	body := build_post_body(field_dict, boundary)
	req, err := http.NewRequest("POST", info.Host, bytes.NewReader([]byte(body)))
	if err != nil {

	}
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	req.Header.Set("Content-Type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary))

	resp, err := client.Do(req)
	if err != nil {
	}
	defer resp.Body.Close()

	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
	}

	if resp.StatusCode == 200 {
		fmt.Println("upload alioss sucess")
	} else {
		fmt.Println(string(content))
	}

}
