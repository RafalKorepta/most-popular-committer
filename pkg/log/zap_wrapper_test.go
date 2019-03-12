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
package log

import (
	"testing"

	"encoding/json"

	"github.com/magiconair/properties/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Msg struct {
	Level string  `json:"level"`
	Ts    float64 `json:"ts"`
	Msg   string  `json:"msg"`
}

var LogMsg Msg

type mockWriteSyncer struct {
	zapcore.WriteSyncer
	Name string
}

func (m mockWriteSyncer) Write(p []byte) (int, error) {
	err := json.Unmarshal(p, &LogMsg)
	return 0, err
}
func (mockWriteSyncer) Sync() error {
	return nil
}

func TestZapWrapper(t *testing.T) {
	// Arrange
	outSink := mockWriteSyncer{
		Name: "OutSink",
	}
	errSink := mockWriteSyncer{
		Name: "ErrSink",
	}

	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionConfig().EncoderConfig),
			zapcore.Lock(zapcore.AddSync(outSink)),
			zap.InfoLevel,
		),
		zap.ErrorOutput(errSink),
	)
	zw := ZapWrapper{
		Logger: logger,
		Sugar:  logger.Sugar(),
	}

	const errMsg = "test msg"
	t.Run("Error writes to stderr", func(t *testing.T) {
		// Act
		zw.Error(errMsg)

		// Assert
		assert.Equal(t, LogMsg.Msg, errMsg, "The message must be the same")
		assert.Equal(t, LogMsg.Level, "error", "The level must be error")
	})
	t.Run("Infof writes to stdout", func(t *testing.T) {
		// Act
		const key = "fieldKey"
		const value = "fieldValue"
		zw.Infof(errMsg, key, value)

		// Assert
		assert.Equal(t, LogMsg.Msg, errMsg+"["+key+" "+value+"]",
			"The message must be the same and args must be in square bracket")
		assert.Equal(t, LogMsg.Level, "info", "The level must be error")
	})
}
