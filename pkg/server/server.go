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
	"net"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	serverDefaultName = "most-popular-committer"
)

type Server struct {
	listener   net.Listener
	serverName string
	logger     *zap.Logger
	server     *http.Server
}

// NewServer constructor of Server
func NewServer(l net.Listener, opts ...ServerOption) (*Server, error) {
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

	s.server = s.createHTTPServer(s.listener.Addr().String())

	defer s.listener.Close()
	return s.server.Serve(s.listener)
}

func (s *Server) createHTTPServer(addr string) *http.Server {
	rootHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO(tamird): point to merged gRPC code rather than a PR.
		// This is a partial recreation of gRPC's internal checks
		// https://github.com/grpc/grpc-go/pull/514/files#diff-95e9a25b738459a2d3030e1e6fa2a718R61
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			s.logger.Debug("Handle gRPC of " + s.serverName + " server")
		} else {
			s.logger.Debug("Handle none gRPC of " + s.serverName + " server")
		}
	})

	return &http.Server{
		Addr:    addr,
		Handler: rootHandler,
	}
}
