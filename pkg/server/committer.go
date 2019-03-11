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
	"crypto/tls"
	"net/http"
	"sort"

	"go.uber.org/zap"

	pb "github.com/RafalKorepta/most-popular-committer/pkg/api/committer"
	"github.com/google/go-github/github"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	maxTopRatedProjects = 5
	maxContributors     = 10
)

type committerService struct {
	logger *zap.Logger
	pb.CommitterServiceServer
}

// MostActiveCommitter return list of most active committer per project which have the most github start for
// requested language
func (s *committerService) MostActiveCommitter(ctx context.Context,
	req *pb.CommitterRequest) (*pb.CommitterResponse, error) {

	if req.Language == "" {
		return nil, status.Error(codes.InvalidArgument, "Language need to be provided")
	}

	// Because of problems with docker running on osx I disable tls verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // nolint:gosec
	}

	client := github.NewClient(&http.Client{Transport: tr})

	s.logger.Debug("Created github client")

	rsr, _, err := client.Search.Repositories(ctx, "language:"+req.Language, &github.SearchOptions{
		Sort:  "stars",
		Order: "desc",
		ListOptions: github.ListOptions{
			Page:    0,
			PerPage: maxTopRatedProjects,
		},
	})
	if err != nil {
		s.logger.Error("Failed to query to projects", zap.Error(err))
		return nil, status.Error(codes.Internal, "Failed to search for projects")
	}

	s.logger.Debug("Retrieve repositories", zap.Any("repositories list", rsr))

	resp := &pb.CommitterResponse{
		Language: req.Language,
	}

	for _, repo := range rsr.Repositories {
		contributors, _, err := client.Repositories.ListContributors(
			ctx,
			*repo.Owner.Login,
			*repo.Name,
			&github.ListContributorsOptions{
				Anon: "true",
				ListOptions: github.ListOptions{
					Page:    0,
					PerPage: maxContributors,
				},
			})
		if err != nil {
			s.logger.Error("Failed to query to contributors", zap.Error(err))
			return nil, status.Error(codes.Internal, "Failed to adds contributors")
		}

		for _, c := range contributors {
			if c.Login == nil {
				continue
			}

			resp.Contributors = append(resp.Contributors, &pb.Committer{
				Name:    *c.Login,
				Commits: uint64(*c.Contributions),
			})
		}
	}

	sort.Slice(resp.Contributors, func(i, j int) bool {
		return resp.Contributors[i].Commits > resp.Contributors[j].Commits
	})

	resp.Contributors = resp.Contributors[:maxContributors]
	return resp, nil
}
