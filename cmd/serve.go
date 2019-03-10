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

package cmd

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/RafalKorepta/most-popular-committer/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	portNumberFlag   = "port_number"
	certPathFlag     = "certs_path"
	certFileNameFlag = "cert_file_name"
	keyFileNameFlag  = "key_file_name"
	secureFlag       = "secure"
	serverCapacity   = "capacity"
	serverRate       = "rate"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the server",
	Long: `The command for starting the service that
search for most popular projects on github with
the given programmatic language`,
	Run: func(cmd *cobra.Command, args []string) {
		listener, err := net.ListenTCP("tcp",
			&net.TCPAddr{
				IP:   net.ParseIP("127.0.0.1"),
				Port: viper.GetInt(portNumberFlag),
			})
		if err != nil {
			zap.L().Fatal(fmt.Sprintf("Can not listen on localhost:%d", viper.GetInt(portNumberFlag)), zap.Error(err))
		}
		srv, err := server.NewServer(listener,
			server.WithLogger(zap.L()),
			server.WithCapacity(viper.GetInt64(serverCapacity)),
			server.WithRate(viper.GetInt64(serverRate)),
			server.WithSecure(viper.GetBool(secureFlag)),
			server.WithCertFile(filepath.Join(viper.GetString(certPathFlag), viper.GetString(certFileNameFlag))),
			server.WithKeyFile(filepath.Join(viper.GetString(certPathFlag), viper.GetString(keyFileNameFlag))))
		if err != nil {
			zap.L().Fatal("Unable to create server", zap.Error(err))
		}
		err = srv.Serve()
		if err != nil {
			zap.L().Fatal("Server failed", zap.Error(err))
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().IntP(portNumberFlag, "p", 9091,
		"the port on which the server will be listen on incoming requests")
	serveCmd.Flags().Int64P(serverCapacity, "c", 10,
		"server request maximum capacity")
	serveCmd.Flags().Int64P(serverRate, "r", 25,
		"server per second request rate")
	serveCmd.Flags().String(certPathFlag, "pkg/certs/local_certs",
		"the path where key and certificate are located")
	serveCmd.Flags().String(certFileNameFlag, "server.pem",
		"the path where key and certificate are located")
	serveCmd.Flags().String(keyFileNameFlag, "server.key",
		"the path where key and certificate are located")
	serveCmd.Flags().BoolP(secureFlag, "s", false,
		"flag which change if email service will be serving tls connection or not")

	if err := viper.BindPFlags(serveCmd.Flags()); err != nil {
		zap.L().Error("Unable to bind flags")
	}
}
