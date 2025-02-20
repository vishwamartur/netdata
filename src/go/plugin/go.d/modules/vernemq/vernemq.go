// SPDX-License-Identifier: GPL-3.0-or-later

package vernemq

import (
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/netdata/netdata/go/plugins/plugin/go.d/agent/module"
	"github.com/netdata/netdata/go/plugins/plugin/go.d/pkg/confopt"
	"github.com/netdata/netdata/go/plugins/plugin/go.d/pkg/prometheus"
	"github.com/netdata/netdata/go/plugins/plugin/go.d/pkg/web"
)

//go:embed "config_schema.json"
var configSchema string

func init() {
	module.Register("vernemq", module.Creator{
		JobConfigSchema: configSchema,
		Create:          func() module.Module { return New() },
		Config:          func() any { return &Config{} },
	})
}

func New() *VerneMQ {
	return &VerneMQ{
		Config: Config{
			HTTPConfig: web.HTTPConfig{
				RequestConfig: web.RequestConfig{
					URL: "http://127.0.0.1:8888/metrics",
				},
				ClientConfig: web.ClientConfig{
					Timeout: confopt.Duration(time.Second),
				},
			},
		},
		charts:    &module.Charts{},
		seenNodes: make(map[string]bool),
	}
}

type Config struct {
	UpdateEvery    int `yaml:"update_every,omitempty" json:"update_every"`
	web.HTTPConfig `yaml:",inline" json:""`
}

type VerneMQ struct {
	module.Base
	Config `yaml:",inline" json:""`

	charts *module.Charts

	prom prometheus.Prometheus

	namespace struct {
		found bool
		name  string
	} // added in v2.0 (default is 'vernemq')

	seenNodes map[string]bool
}

func (v *VerneMQ) Configuration() any {
	return v.Config
}

func (v *VerneMQ) Init() error {
	if err := v.validateConfig(); err != nil {
		return fmt.Errorf("config validation: %v", err)
	}

	prom, err := v.initPrometheusClient()
	if err != nil {
		return fmt.Errorf("init prometheus client: %v", err)
	}
	v.prom = prom

	return nil
}

func (v *VerneMQ) Check() error {
	mx, err := v.collect()
	if err != nil {
		return err
	}

	if len(mx) == 0 {
		return errors.New("no metrics collected")
	}

	return nil
}

func (v *VerneMQ) Charts() *module.Charts {
	return v.charts
}

func (v *VerneMQ) Collect() map[string]int64 {
	mx, err := v.collect()
	if err != nil {
		v.Error(err)
	}

	if len(mx) == 0 {
		return nil
	}

	return mx
}

func (v *VerneMQ) Cleanup() {
	if v.prom != nil && v.prom.HTTPClient() != nil {
		v.prom.HTTPClient().CloseIdleConnections()
	}
}
