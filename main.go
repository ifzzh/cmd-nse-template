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

package main

import (
	"context"
	"crypto/tls"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	// 标准库和第三方工具
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/edwarnicke/grpcfd"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	// VPP相关
	"github.com/networkservicemesh/vpphelper"

	// NSM API定义
	"github.com/networkservicemesh/api/pkg/api/networkservice"

	// NSM SDK - VPP集成
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/acl"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/mechanisms/memif"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/up"
	"github.com/networkservicemesh/sdk-vpp/pkg/networkservice/xconnect"

	// NSM SDK - 核心功能
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/client"
	"github.com/networkservicemesh/sdk/pkg/networkservice/chains/endpoint"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/authorize"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/clienturl"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/connect"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/recvfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanisms/sendfd"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/mechanismtranslation"
	"github.com/networkservicemesh/sdk/pkg/networkservice/common/passthrough"
	"github.com/networkservicemesh/sdk/pkg/networkservice/core/chain"
	"github.com/networkservicemesh/sdk/pkg/networkservice/utils/metadata"

	// NSM SDK - 工具
	"github.com/networkservicemesh/sdk/pkg/tools/debug"
	"github.com/networkservicemesh/sdk/pkg/tools/grpcutils"
	"github.com/networkservicemesh/sdk/pkg/tools/log"
	"github.com/networkservicemesh/sdk/pkg/tools/log/logruslogger"
	"github.com/networkservicemesh/sdk/pkg/tools/opentelemetry"
	"github.com/networkservicemesh/sdk/pkg/tools/pprofutils"
	"github.com/networkservicemesh/sdk/pkg/tools/spiffejwt"
	"github.com/networkservicemesh/sdk/pkg/tools/token"
	"github.com/networkservicemesh/sdk/pkg/tools/tracing"

	// 本地模块
	"github.com/ifzzh/cmd-nse-template/internal"
)

func main() {
	// ========================================================================
	// 阶段 0: 初始化 - 设置上下文和日志系统
	// ========================================================================

	// 设置信号捕获上下文（用于优雅退出）
	ctx, cancel := notifyContext()
	defer cancel()

	// 配置日志系统
	log.EnableTracing(true)                                                          // 启用追踪
	logrus.SetFormatter(&nested.Formatter{})                                         // 使用嵌套格式化器
	ctx = log.WithLog(ctx, logruslogger.New(ctx, map[string]interface{}{"cmd": os.Args[0]})) // 初始化日志上下文

	// 调试信息输出
	if err := debug.Self(); err != nil {
		log.FromContext(ctx).Infof("%s", err)
	}

	// 打印启动流程说明
	log.FromContext(ctx).Infof("========================================")
	log.FromContext(ctx).Infof("防火墙网络服务端点 (Firewall NSE) 启动")
	log.FromContext(ctx).Infof("========================================")
	log.FromContext(ctx).Infof("即将执行6个启动阶段，然后输出成功消息：")
	log.FromContext(ctx).Infof("  阶段1: 从环境变量加载配置")
	log.FromContext(ctx).Infof("  阶段2: 获取SPIFFE身份凭证 (SVID)")
	log.FromContext(ctx).Infof("  阶段3: 创建gRPC客户端选项")
	log.FromContext(ctx).Infof("  阶段4: 创建防火墙网络服务端点")
	log.FromContext(ctx).Infof("  阶段5: 创建gRPC服务器并挂载端点")
	log.FromContext(ctx).Infof("  阶段6: 向NSM Manager注册端点")
	log.FromContext(ctx).Infof("  最终: 输出启动耗时统计")
	log.FromContext(ctx).Infof("========================================")

	starttime := time.Now()

	// ========================================================================
	// 阶段 1: 配置加载 - 从环境变量和配置文件读取参数
	// ========================================================================
	log.FromContext(ctx).Infof("")
	log.FromContext(ctx).Infof(">>> 阶段1: 从环境变量加载配置")
	// 加载配置（从环境变量和ACL配置文件）
	config, err := internal.LoadConfig(ctx)
	if err != nil {
		logrus.Fatal(err.Error())
	}

	// 设置日志级别
	l, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logrus.Fatalf("无效的日志级别: %s", config.LogLevel)
	}
	logrus.SetLevel(l)

	// 配置信号切换日志级别（SIGUSR1=TRACE, SIGUSR2=恢复）
	logruslogger.SetupLevelChangeOnSignal(ctx, map[os.Signal]logrus.Level{
		syscall.SIGUSR1: logrus.TraceLevel,
		syscall.SIGUSR2: l,
	})

	log.FromContext(ctx).Infof("配置加载完成: %#v", config)

	// ---- 配置可观测性工具 ----

	// 配置OpenTelemetry（分布式追踪和指标）
	if opentelemetry.IsEnabled() {
		collectorAddress := config.OpenTelemetryEndpoint
		spanExporter := opentelemetry.InitSpanExporter(ctx, collectorAddress)
		metricExporter := opentelemetry.InitOPTLMetricExporter(ctx, collectorAddress, config.MetricsExportInterval)
		o := opentelemetry.Init(ctx, spanExporter, metricExporter, config.Name)
		defer func() {
			if err = o.Close(); err != nil {
				log.FromContext(ctx).Error(err.Error())
			}
		}()
		log.FromContext(ctx).Infof("OpenTelemetry已启用，Collector地址: %s", collectorAddress)
	}

	// 配置pprof性能分析工具
	if config.PprofEnabled {
		go pprofutils.ListenAndServe(ctx, config.PprofListenOn)
		log.FromContext(ctx).Infof("pprof性能分析已启用，监听地址: %s", config.PprofListenOn)
	}

	// ========================================================================
	// 阶段 2: 身份认证 - 获取SPIFFE身份凭证（SVID）
	// ========================================================================
	log.FromContext(ctx).Infof("")
	log.FromContext(ctx).Infof(">>> 阶段2: 获取SPIFFE身份凭证")
	log.FromContext(ctx).Infof("提示: 如果程序停在此处，请检查SPIRE Agent日志")
	// 创建X509身份凭证源（连接SPIRE Agent）
	source, err := workloadapi.NewX509Source(ctx)
	if err != nil {
		logrus.Fatalf("获取X509凭证源失败: %+v", err)
	}

	// 获取SPIFFE验证身份文档（SVID）
	svid, err := source.GetX509SVID()
	if err != nil {
		logrus.Fatalf("获取X509 SVID失败: %+v", err)
	}
	log.FromContext(ctx).Infof("SVID身份: %q", svid.ID)

	// 配置mTLS（双向TLS）配置
	// 客户端配置：用于连接NSM Manager
	tlsClientConfig := tlsconfig.MTLSClientConfig(source, source, tlsconfig.AuthorizeAny())
	tlsClientConfig.MinVersion = tls.VersionTLS12

	// 服务器配置：用于接受NSC连接
	tlsServerConfig := tlsconfig.MTLSServerConfig(source, source, tlsconfig.AuthorizeAny())
	tlsServerConfig.MinVersion = tls.VersionTLS12

	// ========================================================================
	// 阶段 3: gRPC客户端配置 - 设置连接NSM Manager的选项
	// ========================================================================
	log.FromContext(ctx).Infof("")
	log.FromContext(ctx).Infof(">>> 阶段3: 创建gRPC客户端选项")
	// 配置gRPC客户端选项（用于连接NSM Manager和其他NSE）
	clientOptions := append(
		tracing.WithTracingDial(),                                                                                      // 启用追踪
		grpc.WithDefaultCallOptions(
			grpc.WaitForReady(true),                                                                                    // 等待服务就绪
			grpc.PerRPCCredentials(token.NewPerRPCCredentials(spiffejwt.TokenGeneratorFunc(source, config.MaxTokenLifetime))), // JWT令牌认证
		),
		grpc.WithTransportCredentials(
			grpcfd.TransportCredentials(
				credentials.NewTLS(tlsClientConfig), // mTLS传输加密
			),
		),
		grpcfd.WithChainStreamInterceptor(),  // 文件描述符传递（流式）
		grpcfd.WithChainUnaryInterceptor(),   // 文件描述符传递（一元）
	)

	// ========================================================================
	// 阶段 4: 创建防火墙端点 - VPP连接和端点链构建
	// ========================================================================
	log.FromContext(ctx).Infof("")
	log.FromContext(ctx).Infof(">>> 阶段4: 创建防火墙网络服务端点")

	// 启动VPP进程并建立连接
	log.FromContext(ctx).Infof("正在启动VPP (Vector Packet Processing)...")
	vppConn, vppErrCh := vpphelper.StartAndDialContext(ctx)
	exitOnErr(ctx, cancel, vppErrCh) // 监控VPP错误通道
	log.FromContext(ctx).Infof("VPP连接建立成功")

	// 构建防火墙端点链（服务器端）
	log.FromContext(ctx).Infof("正在构建防火墙端点链...")
	firewallEndpoint := new(struct{ endpoint.Endpoint })
	firewallEndpoint.Endpoint = endpoint.NewServer(ctx,
		spiffejwt.TokenGeneratorFunc(source, config.MaxTokenLifetime), // Token生成器
		endpoint.WithName(config.Name),                                 // 端点名称
		endpoint.WithAuthorizeServer(authorize.NewServer()),            // 授权服务器
		endpoint.WithAdditionalFunctionality(
			// 基础功能链（接收/发送文件描述符、接口管理、交叉连接等）
			recvfd.NewServer(),                           // 接收文件描述符
			sendfd.NewServer(),                           // 发送文件描述符
			up.NewServer(ctx, vppConn),                   // VPP接口UP状态管理
			clienturl.NewServer(&config.ConnectTo),       // 客户端连接URL
			xconnect.NewServer(vppConn),                  // VPP交叉连接（L2转发）
			acl.NewServer(vppConn, config.ACLConfig),     // ACL防火墙规则应用 ← 核心功能
			mechanisms.NewServer(map[string]networkservice.NetworkServiceServer{
				memif.MECHANISM: chain.NewNetworkServiceServer(memif.NewServer(ctx, vppConn)), // memif共享内存接口
			}),
			// 连接服务器（处理到其他NSE的连接）
			connect.NewServer(
				client.NewClient(
					ctx,
					client.WithoutRefresh(),                // 禁用刷新（静态配置）
					client.WithName(config.Name),           // 客户端名称
					client.WithDialOptions(clientOptions...), // 使用阶段3配置的客户端选项
					client.WithAdditionalFunctionality(
						// 客户端功能链（元数据、机制转换、标签透传等）
						metadata.NewClient(),                        // 元数据管理
						mechanismtranslation.NewClient(),            // 机制转换
						passthrough.NewClient(config.Labels),        // 标签透传
						up.NewClient(ctx, vppConn),                  // VPP接口UP（客户端）
						xconnect.NewClient(vppConn),                 // VPP交叉连接（客户端）
						memif.NewClient(ctx, vppConn),               // memif接口（客户端）
						sendfd.NewClient(),                          // 发送FD（客户端）
						recvfd.NewClient(),                          // 接收FD（客户端）
					)),
			),
		))
	log.FromContext(ctx).Infof("防火墙端点链构建完成（包含ACL规则）")

	// ========================================================================
	// 阶段 5: gRPC服务器创建 - 创建服务器并注册防火墙端点
	// ========================================================================
	log.FromContext(ctx).Infof("")
	log.FromContext(ctx).Infof(">>> 阶段5: 创建gRPC服务器并挂载端点")
	// 创建gRPC服务器（启用追踪和mTLS）
	server := grpc.NewServer(append(
		tracing.WithTracing(),                                      // 启用分布式追踪
		grpc.Creds(
			grpcfd.TransportCredentials(
				credentials.NewTLS(tlsServerConfig),            // mTLS服务器凭证
			),
		),
	)...)

	// 将防火墙端点注册到gRPC服务器
	firewallEndpoint.Register(server)
	log.FromContext(ctx).Infof("防火墙端点已注册到gRPC服务器")

	// 创建临时目录并构造Unix Socket监听地址
	tmpDir, err := os.MkdirTemp("", config.Name)
	if err != nil {
		logrus.Fatalf("创建临时目录失败: %+v", err)
	}
	defer func(tmpDir string) { _ = os.Remove(tmpDir) }(tmpDir) // 退出时清理
	listenOn := &(url.URL{Scheme: "unix", Path: filepath.Join(tmpDir, config.ListenOn)})
	log.FromContext(ctx).Infof("Unix Socket监听地址: %s", listenOn.String())

	// 启动gRPC服务器
	srvErrCh := grpcutils.ListenAndServe(ctx, listenOn, server)
	exitOnErr(ctx, cancel, srvErrCh) // 监控服务器错误通道
	log.FromContext(ctx).Infof("gRPC服务器启动成功")

	// ========================================================================
	// 阶段 6: 端点注册 - 向NSM Manager注册防火墙网络服务端点
	// ========================================================================
	log.FromContext(ctx).Infof("")
	log.FromContext(ctx).Infof(">>> 阶段6: 向NSM Manager注册端点")

	// 创建Registry客户端
	nseRegistryClient := internal.NewRegistryClient(ctx, &config.ConnectTo, clientOptions, config.RegistryClientPolicies)

	// 注册网络服务端点
	nse, err := internal.RegisterEndpoint(ctx, nseRegistryClient, config.Name, config.ServiceName, config.Labels, listenOn.String())
	if err != nil {
		log.FromContext(ctx).Fatalf("端点注册失败: %+v", err)
	}

	log.FromContext(ctx).Infof("端点注册成功!")
	log.FromContext(ctx).Infof("  名称: %s", nse.Name)
	log.FromContext(ctx).Infof("  服务: %v", nse.NetworkServiceNames)
	log.FromContext(ctx).Infof("  地址: %s", nse.Url)
	log.FromContext(ctx).Infof("  标签: %+v", config.Labels)

	// ========================================================================
	// 启动完成 - 输出启动统计并进入运行状态
	// ========================================================================
	log.FromContext(ctx).Infof("")
	log.FromContext(ctx).Infof("========================================")
	log.FromContext(ctx).Infof("✓ 启动成功! 耗时: %v", time.Since(starttime))
	log.FromContext(ctx).Infof("========================================")
	log.FromContext(ctx).Infof("防火墙NSE正在运行，等待连接请求...")

	// ========================================================================
	// 等待退出信号 - 保持运行直到收到中断信号
	// ========================================================================
	<-ctx.Done()   // 等待上下文取消（SIGINT/SIGTERM等）
	<-vppErrCh     // 等待VPP错误通道关闭
	log.FromContext(ctx).Infof("防火墙NSE已停止")
}

// ============================================================================
// 辅助函数
// ============================================================================

// exitOnErr 监控错误通道，如果收到错误则记录并取消上下文
// 用于处理VPP连接错误和gRPC服务器错误
func exitOnErr(ctx context.Context, cancel context.CancelFunc, errCh <-chan error) {
	// 检查是否已有错误（非阻塞）
	select {
	case err := <-errCh:
		log.FromContext(ctx).Fatal(err) // 立即退出
	default:
	}

	// 在后台等待错误（阻塞）
	go func(ctx context.Context, errCh <-chan error) {
		err := <-errCh
		log.FromContext(ctx).Error(err)
		cancel() // 触发上下文取消，优雅退出
	}(ctx, errCh)
}

// notifyContext 创建一个可以响应系统信号的上下文
// 支持的信号：SIGINT（Ctrl+C）、SIGHUP、SIGTERM、SIGQUIT
func notifyContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(
		context.Background(),
		os.Interrupt,       // Ctrl+C
		syscall.SIGHUP,     // 终端挂起
		syscall.SIGTERM,    // 终止信号（graceful shutdown）
		syscall.SIGQUIT,    // 退出信号
	)
}
