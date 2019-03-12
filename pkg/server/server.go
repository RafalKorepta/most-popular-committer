// Copyright © 2019 Rafal Korepta <rafal.korepta@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	pb "github.com/RafalKorepta/most-popular-committer/pkg/api/committer"
	"github.com/RafalKorepta/most-popular-committer/pkg/certs"
	"github.com/RafalKorepta/most-popular-committer/pkg/log"
	grpc_ratelimit "github.com/RafalKorepta/most-popular-committer/pkg/ratelimit"
	"github.com/RafalKorepta/most-popular-committer/pkg/ratelimit/tokenbucket"
	"github.com/RafalKorepta/most-popular-committer/pkg/ui"
	"github.com/google/go-github/github"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/uber/jaeger-client-go/config"
	"github.com/uber/jaeger-lib/metrics/prometheus"
	"github.com/veqryn/h2c"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	serverDefaultName = "most-popular-committer"
)

type SecureConfig struct {
	secure   bool
	certFile string
	keyFile  string
}

type Server struct {
	listener   net.Listener
	serverName string
	logger     *zap.Logger

	secureCfg SecureConfig
	capacity  int64
	rate      int64
}

// NewServer constructor of Server
func NewServer(l net.Listener, opts ...Option) (*Server, error) {
	if l == nil {
		return nil, errors.New("missing listener")
	}

	srv := &Server{
		listener:   l,
		serverName: serverDefaultName,
		logger:     zap.L(),
	}
	applyOpts(srv, opts)
	return srv, nil
}

// Serve will start gRPC and REST server on the same port with or without TLS
func (s *Server) Serve() error {
	tracerCloser, err := initializeGlobalTracer(s.serverName, zap.L(), zap.S())
	if err != nil {
		return errors.Wrap(err, "initializing global tracer")
	}
	defer tracerCloser.Close()

	var srv *http.Server
	if s.secureCfg.secure {
		srv, err = s.createHTTPSServer()
		if err != nil {
			return errors.Wrap(err, "crating https server")
		}
	} else {
		srv, err = s.createHTTPServer()
		if err != nil {
			return errors.Wrap(err, "crating http server")
		}
	}

	defer s.listener.Close()
	if s.secureCfg.secure {
		return srv.ServeTLS(s.listener, "", "") // The certificates are initialized already
	}

	return srv.Serve(s.listener)
}

// initializeGlobalTracer will set global tracer using jeager tracer
func initializeGlobalTracer(serverName string, logger *zap.Logger, sugar *zap.SugaredLogger) (io.Closer, error) {
	zapWrapper := log.ZapWrapper{
		Logger: logger,
		Sugar:  sugar,
	}

	metricsFactory := prometheus.New()

	tracer, closer, err := config.Configuration{
		ServiceName: serverName,
	}.NewTracer(
		config.Metrics(metricsFactory),
		config.Logger(zapWrapper),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to start tracer: %v", err)
	}
	opentracing.SetGlobalTracer(tracer)
	return closer, nil
}

func (s *Server) createHTTPServer() (*http.Server, error) {
	addr := s.listener.Addr().String()

	// Because of problems with docker running on osx I disable tls verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint:gosec
	}

	client := github.NewClient(&http.Client{Transport: tr})

	service := &committerService{
		logger:             s.logger,
		repoGetter:         client.Search,
		contributorsGetter: client.Repositories,
	}

	grpcServer := registerCommitterService(service, createGRPCOptions(s.rate, s.capacity)...)

	grpc_prometheus.Register(grpcServer)

	dialOpts := []grpc.DialOption{grpc.WithInsecure()}

	mux, err := registerServerMux(addr, dialOpts...)
	if err != nil {
		return nil, err
	}

	rootHandler := &h2c.HandlerH2C{
		Handler:  grpcHandlerFunc(grpcServer, mux),
		H2Server: &http2.Server{},
	}

	return &http.Server{
		Addr:    addr,
		Handler: rootHandler,
	}, nil
}

func (s *Server) createHTTPSServer() (*http.Server, error) {
	addr := s.listener.Addr().String()

	// Because of problems with docker running on osx I disable tls verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint:gosec
	}

	client := github.NewClient(&http.Client{Transport: tr})

	service := &committerService{
		logger:             s.logger,
		repoGetter:         client.Search,
		contributorsGetter: client.Repositories,
	}

	serverOpts := createGRPCOptions(s.rate, s.capacity)

	certPool, err := certs.CreatePool(s.secureCfg.certFile)
	if err != nil {
		return nil, err
	}
	serverOpts = append(serverOpts, grpc.Creds(credentials.NewClientTLSFromCert(certPool, addr)))

	grpcServer := registerCommitterService(service, serverOpts...)

	grpc_prometheus.Register(grpcServer)

	dialOpts, err := createSecureDialOpts(s.serverName, s.secureCfg.certFile)
	if err != nil {
		return nil, err
	}

	mux, err := registerServerMux(addr, dialOpts...)
	if err != nil {
		return nil, err
	}

	rootHandler := grpcHandlerFunc(grpcServer, mux)

	tlsCfg, err := certs.CreateTLSConfig(s.secureCfg.certFile, s.secureCfg.keyFile)
	if err != nil {
		return nil, err
	}

	return &http.Server{
		Addr:      addr,
		Handler:   rootHandler,
		TLSConfig: tlsCfg,
	}, nil
}

func registerCommitterService(s pb.CommitterServiceServer, serverOpts ...grpc.ServerOption) *grpc.Server {
	grpcServer := grpc.NewServer(serverOpts...)

	pb.RegisterCommitterServiceServer(grpcServer, s)

	return grpcServer
}

func createGRPCOptions(ratePerSecond int64, capacity int64) []grpc.ServerOption {
	var opts []grpc.ServerOption

	grpc_zap.ReplaceGrpcLogger(zap.L())

	optZap := []grpc_zap.Option{
		// Add filed to logs that comes from gRPC middleware
		grpc_zap.WithDurationField(func(duration time.Duration) zapcore.Field {
			return zap.Int64("grpc.time_ns", duration.Nanoseconds())
		}),
	}

	zap.L().Debug("The rate limiting configuration",
		zap.Int64("capacity", capacity),
		zap.Int64("Rate per second", ratePerSecond),
	)

	unaryRateLimiter := tokenbucket.NewTokenBucketRateLimiter(
		time.Second/time.Duration(ratePerSecond), capacity, 1)

	opts = append(opts, grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		grpc_opentracing.UnaryServerInterceptor(),
		grpc_ratelimit.UnaryServerInterceptor(
			grpc_ratelimit.WithLimiter(unaryRateLimiter),
			grpc_ratelimit.WithMaxWaitDuration(time.Microsecond), // Almost no wait for bucket to be filled
		),
		grpc_prometheus.UnaryServerInterceptor,
		grpc_zap.UnaryServerInterceptor(zap.L(), optZap...),
		grpc_recovery.UnaryServerInterceptor(),
	)))

	return opts
}

func createSecureDialOpts(serverOverrideName string, certFile string) ([]grpc.DialOption, error) {
	certPool, err := certs.CreatePool(certFile)
	if err != nil {
		return nil, errors.Wrap(err, "crating certificate pool")
	}
	tCreds := credentials.NewTLS(&tls.Config{
		// Only connection from localhost will be accepted until
		// certificate will have Subject Alternative Name init
		ServerName: serverOverrideName,
		RootCAs:    certPool,
	})
	return []grpc.DialOption{grpc.WithTransportCredentials(tCreds)}, nil
}

// registerServerMux is helper function that registers many http1.1 endpoints in mux
func registerServerMux(addr string, dialOpts ...grpc.DialOption) (*http.ServeMux, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, req *http.Request) {
		var n int64
		n, err := io.Copy(w, strings.NewReader(pb.Swagger))
		if err != nil {
			zap.L().Error("Coping operation failed", zap.Int64("wrriten", n), zap.Error(err))
			http.Error(w, "swagger.json is currently unavailable", http.StatusInternalServerError)
		}
	})

	gwmux := runtime.NewServeMux()
	ctx := context.Background()
	err := pb.RegisterCommitterServiceHandlerFromEndpoint(ctx, gwmux, addr, dialOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to register gRPC gateway: %v", err)
	}

	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/", gwmux)
	ui.ServeSwagger(mux)

	return mux, nil
}

// grpcHandlerFunc returns an http.Handler that delegates to grpcServer on incoming gRPC
// connections or otherHandler otherwise. Copied from cockroachdb.
func grpcHandlerFunc(grpcServer, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(tamird): point to merged gRPC code rather than a PR.
		// This is a partial recreation of gRPC's internal checks
		// https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}
