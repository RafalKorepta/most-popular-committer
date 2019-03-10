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
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
)

func CreateX509Pool(cert io.Reader) (*x509.CertPool, error) {
	if cert == nil {
		return nil, fmt.Errorf("cert can not be nil")
	}

	b, err := ioutil.ReadAll(cert)
	if err != nil {
		return nil, fmt.Errorf("can not read the certificate")
	}

	demoCertPool := x509.NewCertPool()
	ok := demoCertPool.AppendCertsFromPEM(b)
	if !ok {
		return nil, fmt.Errorf("could not append certificate")
	}
	return demoCertPool, nil
}
