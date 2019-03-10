// Copyright [2018] [Rafał Korepta]
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package certs

import (
	"path/filepath"
	"testing"

	"os"

	"fmt"
	"io"

	"github.com/stretchr/testify/assert"
)

const testData = "local_certs"

type MockReader struct {
	io.Reader
}

func (MockReader) Read(p []byte) (n int, err error) { return 0, fmt.Errorf("test error") }

func Test_CreateX509Pool(t *testing.T) {
	// Arrange
	cert, emptyFile := helperLoadFiles(t)
	defer cleanup()

	t.Run("Correct creation of x509 cert pool", func(t *testing.T) {
		// Act
		certPool, err := CreateX509Pool(cert)

		// Assert
		assert.NoError(t, err, "Error should not occur")
		assert.NotNil(t, certPool, "certPool must exist")
	})

	t.Run("The cert argument is nil", func(t *testing.T) {
		// Act
		certPool, err := CreateX509Pool(nil)

		// Assert
		assert.Error(t, err, "Error must occur")
		assert.Nil(t, certPool, "certPool must not exist")
	})

	t.Run("The cert argument is empty file descriptor that cause PANIC", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("The code did not panic")
			}
		}()

		// Act
		certPool, err := CreateX509Pool(&os.File{})

		// Assert
		assert.Nil(t, err, "Error must not exist")
		assert.Nil(t, certPool, "certPool must not exist")
	})

	t.Run("The cert argument is io.Read which has Read implement to return an error", func(t *testing.T) {
		// Arrange
		mock := MockReader{}

		// Act
		certPool, err := CreateX509Pool(mock)

		// Assert
		assert.Error(t, err, "Error must occur")
		assert.Nil(t, certPool, "certPool must not exist")
	})

	t.Run("The cert argument is file descriptor which points to empty file", func(t *testing.T) {
		// Act
		certPool, err := CreateX509Pool(emptyFile)

		// Assert
		assert.Error(t, err, "Error must occur")
		assert.Nil(t, certPool, "certPool must not exist")
	})
}

func cleanup() {
	os.Remove(filepath.Join(testData, "temp.pem"))
}

func helperLoadFiles(t *testing.T) (*os.File, *os.File) {
	certPath := filepath.Join(testData, "server.pem")
	cert, err := os.Open(certPath)
	if err != nil {
		t.Fatal(err)
	}

	emptyFile, err := os.Create(filepath.Join(testData, "temp.pem"))
	if err != nil {
		t.Fatal(err)
	}
	return cert, emptyFile
}
