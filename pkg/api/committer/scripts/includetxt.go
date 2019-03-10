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
package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// Reads all .json files in the current folder
// and encodes them as strings literals in textfiles.go
func main() {
	var source *os.File
	fs, err := ioutil.ReadDir(".")
	check(err, "Failed at reading the directory")
	out, err := os.Create("swagger.pb.go")
	check(err, "Failed at creating a file")
	_, err = out.Write([]byte("package committer \n\nconst (\n"))
	check(err, "Failed at writing to file")
	for _, f := range fs {
		if strings.HasSuffix(f.Name(), ".json") {
			name := strings.TrimPrefix(f.Name(), "committer.")
			_, err = out.Write([]byte(strings.TrimSuffix(name, ".json") + " = `"))
			check(err, "Failed at writing to file")
			source, err = os.Open(f.Name())
			check(err, "Failed at open a file")
			_, err = io.Copy(out, source)
			check(err, "Failed at coping from source file to destination")
			_, err = out.Write([]byte("`\n"))
			check(err, "Failed at writing to file")
		}
	}
	_, err = out.Write([]byte(")\n"))
	check(err, "Failed at writing to file")
}

func check(err error, msg string) {
	if err != nil {
		log.Fatal(msg, " with error: ", err)
	}
}
