// Copyright (c) 2021-2023 Doc.ai and/or its affiliates.
//
// Copyright (c) 2023-2024 Cisco and/or its affiliates.
//
// Copyright (c) 2024 OpenInfra Foundation Europe. All rights reserved.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package internal

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/networkservicemesh/govpp/binapi/acl_types"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

// Config 保存从环境变量读取的配置参数
type Config struct {
	Name                   string              `default:"firewall-server" desc:"Name of Firewall Server"`
	ListenOn               string              `default:"listen.on.sock" desc:"listen on socket" split_words:"true"`
	ConnectTo              url.URL             `default:"unix:///var/lib/networkservicemesh/nsm.io.sock" desc:"url to connect to" split_words:"true"`
	MaxTokenLifetime       time.Duration       `default:"10m" desc:"maximum lifetime of tokens" split_words:"true"`
	RegistryClientPolicies []string            `default:"etc/nsm/opa/common/.*.rego,etc/nsm/opa/registry/.*.rego,etc/nsm/opa/client/.*.rego" desc:"paths to files and directories that contain registry client policies" split_words:"true"`
	ServiceName            string              `default:"" desc:"Name of providing service" split_words:"true"`
	Labels                 map[string]string   `default:"" desc:"Endpoint labels"`
	ACLConfigPath          string              `default:"/etc/firewall/config.yaml" desc:"Path to ACL config file" split_words:"true"`
	ACLConfig              []acl_types.ACLRule `default:"" desc:"configured acl rules" split_words:"true"`
	LogLevel               string              `default:"INFO" desc:"Log level" split_words:"true"`
	OpenTelemetryEndpoint  string              `default:"otel-collector.observability.svc.cluster.local:4317" desc:"OpenTelemetry Collector Endpoint" split_words:"true"`
	MetricsExportInterval  time.Duration       `default:"10s" desc:"interval between mertics exports" split_words:"true"`
	PprofEnabled           bool                `default:"false" desc:"is pprof enabled" split_words:"true"`
	PprofListenOn          string              `default:"localhost:6060" desc:"pprof URL to ListenAndServe" split_words:"true"`
}

// LoadConfig 从环境变量加载配置并解析ACL规则
// 返回完整初始化的Config对象，如果配置加载失败则返回error
func LoadConfig(ctx context.Context) (*Config, error) {
	config := new(Config)

	// 显示配置用法
	if err := envconfig.Usage("nsm", config); err != nil {
		return nil, errors.Wrap(err, "cannot show usage of envconfig nsm")
	}

	// 从环境变量解析配置
	if err := envconfig.Process("nsm", config); err != nil {
		return nil, errors.Wrap(err, "cannot process envconfig nsm")
	}

	// 加载ACL规则
	retrieveACLRules(ctx, config)

	return config, nil
}

// retrieveACLRules 从配置文件读取ACL规则并添加到Config中
// 如果文件读取或解析失败，记录错误但不中断程序运行
func retrieveACLRules(ctx context.Context, c *Config) {
	logger := log.FromContext(ctx).WithField("acl", "config")

	raw, err := os.ReadFile(filepath.Clean(c.ACLConfigPath))
	if err != nil {
		logger.Errorf("Error reading config file: %v", err)
		return
	}
	logger.Infof("Read config file successfully")

	var rv map[string]acl_types.ACLRule
	err = yaml.Unmarshal(raw, &rv)
	if err != nil {
		logger.Errorf("Error parsing config file: %v", err)
		return
	}
	logger.Infof("Parsed acl rules successfully")

	for _, v := range rv {
		c.ACLConfig = append(c.ACLConfig, v)
	}

	logger.Infof("Result rules:%v", c.ACLConfig)
}
