package conf

import (
	"os"

	"android/tvbox/keyd/lib/uinput"

	"gopkg.in/yaml.v3"
)

// KeyHook 配置结构
type KeyHook struct {
	Action  Action      `yaml:"action"`
	Keycode interface{} `yaml:"keycode"`
	Command string      `yaml:"command"`
}

type Device struct {
	Name  string       `yaml:"name"`
	Ident uinput.Ident `yaml:"ident"`
	Input string       `yaml:"input"`
	Hooks []KeyHook    `yaml:"hooks"`
}

var Config struct {
	Debug  bool     `yaml:"debug"`
	Shell  string   `yaml:"shell"`
	Codes  []int    `yaml:"codes"`
	Device []Device `yaml:"device"`
}

func Init(file string) (err error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}

	if err = yaml.Unmarshal(data, &Config); err != nil {
		return
	}

	return
}
