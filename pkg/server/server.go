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
	"crypto/x509"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	pb "github.com/RafalKorepta/most-popular-committer/pkg/api/committer"
	"github.com/RafalKorepta/most-popular-committer/pkg/certs"
	"github.com/RafalKorepta/most-popular-committer/pkg/log"
	grpc_ratelimit "github.com/RafalKorepta/most-popular-committer/pkg/ratelimit"
	"github.com/RafalKorepta/most-popular-committer/pkg/ratelimit/tokenbucket"
	"github.com/RafalKorepta/most-popular-committer/pkg/ui/data/swagger"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_opentracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/opentracing/opentracing-go"
	assetfs "github.com/philips/go-bindata-assetfs"
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

	srv, grpcServer, err := s.createHTTPServer()
	if err != nil {
		return errors.Wrap(err, "crating global tracer")
	}

	grpc_prometheus.Register(grpcServer)

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

func registerEmailService(s pb.CommitterServiceServer, serverOpts ...grpc.ServerOption) *grpc.Server {
	grpcServer := grpc.NewServer(serverOpts...)

	pb.RegisterCommitterServiceServer(grpcServer, s)

	return grpcServer
}

func createGRPCOptions(addr string, s SecureConfig, ratePerSecond int64, capacity int64) ([]grpc.ServerOption, error) {
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

	if s.secure {
		certPool, err := createPool(s.certFile)
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.Creds(credentials.NewClientTLSFromCert(certPool, addr)))
	}
	return opts, nil
}

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
	serveSwagger(mux)

	return mux, nil
}

func createDialOpts(serverOverrideName string, secure bool, certFile string) ([]grpc.DialOption, error) {
	if secure {
		certPool, err := createPool(certFile)
		if err != nil {
			return nil, err
		}
		dcreds := credentials.NewTLS(&tls.Config{
			// Only connection from localhost will be accepted until
			// certificate will have Subject Alternative Name init
			ServerName: serverOverrideName,
			RootCAs:    certPool,
		})
		return []grpc.DialOption{grpc.WithTransportCredentials(dcreds)}, nil
	}
	return []grpc.DialOption{grpc.WithInsecure()}, nil
}

func createPool(certFile string) (*x509.CertPool, error) {
	f, err := os.Open(certFile)
	if err != nil {
		zap.L().Error("Unable to open cert file", zap.Error(err))
	}
	certPool, err := certs.CreateX509Pool(f)
	if err != nil {
		return nil, fmt.Errorf("unable to create x509 cert pool: %v", err)
	}
	return certPool, nil
}

func createServerMainHandler(secure bool, grpcServer, mux http.Handler) http.Handler {
	if secure {
		return grpcHandlerFunc(grpcServer, mux)
	}
	// Wrap the Router
	return &h2c.HandlerH2C{
		Handler:  grpcHandlerFunc(grpcServer, mux),
		H2Server: &http2.Server{},
	}
}

func (s *Server) createHTTPServer() (*http.Server, *grpc.Server, error) {
	addr := s.listener.Addr().String()

	serverOpts, err := createGRPCOptions(addr, s.secureCfg, s.rate, s.capacity)
	if err != nil {
		return nil, nil, err
	}

	grpcServer := registerEmailService(&committerService{logger: s.logger}, serverOpts...)

	dialOpts, err := createDialOpts(s.serverName, s.secureCfg.secure, s.secureCfg.certFile)
	if err != nil {
		return nil, nil, err
	}

	mux, err := registerServerMux(addr, dialOpts...)
	if err != nil {
		return nil, nil, err
	}

	rootHandler := createServerMainHandler(s.secureCfg.secure, grpcServer, mux)

	tlsCfg, err := createTLSConfig(s.secureCfg.secure, s.secureCfg.certFile, s.secureCfg.keyFile)
	if err != nil {
		return nil, nil, err
	}

	return &http.Server{
		Addr:      addr,
		Handler:   rootHandler,
		TLSConfig: tlsCfg,
	}, grpcServer, nil
}

func createTLSConfig(secure bool, certFile, keyFile string) (*tls.Config, error) {
	if secure {
		keyPair, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("unable to create x509 key pair certificate: %v", err)
		}

		return &tls.Config{
			Certificates: []tls.Certificate{keyPair},
			NextProtos:   []string{"h2"},
		}, nil
	}
	return nil, nil
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

// serveSwagger will register `/swagger-ui` endpoint into root mux.
// This will provide visual representation of gRPC contract
// The swagger-ui is auto generated by script located in `hack/build-ui.sh`
func serveSwagger(mux *http.ServeMux) {
	err := mime.AddExtensionType(".svg", "image/svg+xml")
	if err != nil {
		zap.L().Error("Unable to add extension type", zap.Error(err))
	}

	// Expose files in third_party/swagger-ui/ on <host>/swagger-ui
	fileServer := http.FileServer(&assetfs.AssetFS{
		Asset:    swagger.Asset,
		AssetDir: swagger.AssetDir,
		Prefix:   "third_party/swagger-ui",
	})
	prefix := "/swagger-ui/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
}
