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

	"github.com/RafalKorepta/most-popular-committer/pkg/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	portNumberFlag = "port_number"
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
		srv, err := server.NewServer(listener, server.WithLogger(zap.L()))
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
	if err := viper.BindPFlags(serveCmd.Flags()); err != nil {
		zap.L().Error("Unable to bind flags")
	}
}
