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
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	DebugFlag      = "debug"
	configPathFlag = "cfg_path"
	configFlag     = "config"
)

var (
	// Version will be populated with binary semver by the linker
	// during the build process.
	// See https://blog.cloudflare.com/setting-go-variables-at-compile-time/
	// and https://golang.org/cmd/link/ in section Flags `-X importpath.name=value`.
	Version string

	// Commit will be populated with correct git commit id by the linker
	// during the build process.
	// See https://blog.cloudflare.com/setting-go-variables-at-compile-time/
	// and https://golang.org/cmd/link/ in section Flags `-X importpath.name=value`.
	Commit string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "most-popular-projects",
	Short: "Most popular github project per language",
	Long: `Server for finding most popular github
projects per programmatic language`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolP(DebugFlag, "d", false, "turn on debug logging")
	rootCmd.PersistentFlags().String(configPathFlag, ".", "Relative path where config resides")
	rootCmd.PersistentFlags().String(configFlag, ".most-popular-committer",
		"config file (default is $HOME/.most-popular-committer.yml)")
	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		zap.L().Error("Can not bind persistent flags")
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigName(viper.GetString(configFlag)) // name of config file (without extension)
	viper.AddConfigPath(viper.GetString(configPathFlag))
	viper.AddConfigPath("$HOME")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		zap.S().Errorw("Failed to read from config file",
			"configFile", viper.ConfigFileUsed(),
			"error", err)
	}

	// Update global logger in debug configuration
	cfg := zap.NewProductionConfig()
	if viper.GetBool("debug") {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}

	newLogger, err := cfg.Build(zap.AddStacktrace(zap.ErrorLevel),
		zap.Fields(
			zap.Field{
				Key:    "commit",
				Type:   zapcore.StringType,
				String: Commit,
			},
			zap.Field{
				Key:    "version",
				Type:   zapcore.StringType,
				String: Version,
			},
		))
	if err != nil {
		log.Fatalf("Unable to create logger. Error: %v", err)
	}
	zap.ReplaceGlobals(newLogger)
}
