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
	"context"
	"time"

	pb "github.com/RafalKorepta/most-popular-committer/pkg/api/committer"
)

type committerService struct {
	pb.CommitterServiceServer
}

// MostActiveCommitter return list of most active committer per project which have the most github start for
// requested language
func (s *committerService) MostActiveCommitter(context.Context, *pb.CommitterRequest) (*pb.CommitterResponse, error) {
	time.Sleep(time.Second)
	return &pb.CommitterResponse{}, nil
}
