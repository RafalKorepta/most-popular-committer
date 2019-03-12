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
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockListener struct {
	mock.Mock
}

func (l *mockListener) Accept() (net.Conn, error) {
	return nil, nil
}

func (l *mockListener) Close() error {
	return nil
}

func (l *mockListener) Addr() net.Addr {
	return nil
}

func TestNewServer(t *testing.T) {
	t.Run("Fails without a listener", func(t *testing.T) {
		// Given no network listener

		// When creating new server
		_, err := NewServer(nil)

		// Then an error is returned
		assert.Error(t, err)
	})

	t.Run("Valid new server", func(t *testing.T) {
		// Given network listener
		mockListener := &mockListener{}

		// When creating new server
		srv, err := NewServer(mockListener)

		// Then an error is returned
		assert.NoError(t, err)
		assert.Equal(t, mockListener, srv.listener)
		assert.Equal(t, "most-popular-committer", srv.serverName)
	})

	t.Run("Valid new server with all functional options", func(t *testing.T) {
		// Given network listener
		mockListener := &mockListener{}

		// When creating new server
		srv, err := NewServer(mockListener,
			WithServerName("backend"),
			WithLogger(zap.L()),
			WithCertFile("../certs/local_certs/server.pem"),
			WithKeyFile("../certs/local_certs/server.key"),
			WithSecure(true),
			WithCapacity(10),
			WithRate(25),
		)

		// Then an error is returned
		assert.NoError(t, err)
		assert.Equal(t, mockListener, srv.listener)
		assert.Equal(t, "backend", srv.serverName)
	})
}
