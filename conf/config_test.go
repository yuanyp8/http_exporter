package conf_test

import (
	"fmt"
	"github.com/yuanyp8/http_exporter/conf"
	"testing"
)

func TestLoadConfig(t *testing.T) {

	if err := conf.C().ReloadConfig("testdata/config-demo.yaml"); err != nil {
		t.Errorf("Error loading config: %v\n", err)
	}
	fmt.Println(conf.C().C.Modules["http_get_2xx"].HTTP.Method)
}
