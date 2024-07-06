package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Config struct {
	InputFilePath   string `json:"inputFilePath"`
	SpecificPath    string `json:"specificFilePath"`
	MubuBaseUrl     string `json:"mubuBaseUrl"`
	AutoModelName   string `json:"autoModelName"`
	AnkiConnectHost string `json:"ankiConnectHost"`
}

func (conf *Config) GetConfig() error {
	file, err := os.Open("config.json")
	if err != nil {
		return fmt.Errorf("配置文件打开失败 %v", err)
	}
	defer file.Close()
	bytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("配置文件读取失败 %v", err)
	}
	err = json.Unmarshal(bytes, conf)
	if err != nil {
		return fmt.Errorf("配置文件Josn解析失败 %v", err)
	}
	return nil
}
