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

import "go.uber.org/zap"

// ServerOption modifies properties of a Server. Can be used to set
// optional provider params, such as server name or logger
type ServerOption func(provider *Server)

// WithServerName if set to true, then it will start server with TLS encryption
func WithServerName(n string) ServerOption {
	return func(s *Server) {
		s.serverName = n
	}
}

// WithLogger creates an option that assigns a logger to an instance of Server
func WithLogger(l *zap.Logger) ServerOption {
	return func(s *Server) {
		s.logger = l
	}
}

// applyOpts applies a set of options to Server.
func applyOpts(s *Server, opts []ServerOption) {
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
}
