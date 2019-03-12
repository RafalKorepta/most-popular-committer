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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/google/go-github/github"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/stretchr/testify/require"

	pb "github.com/RafalKorepta/most-popular-committer/pkg/api/committer"
)

type mockRepoGetter struct {
	mock.Mock
}

func (m *mockRepoGetter) Repositories(ctx context.Context, query string,
	opt *github.SearchOptions) (*github.RepositoriesSearchResult, *github.Response, error) {

	args := m.Called(ctx, query, opt)
	result, ok := args.Get(0).(*github.RepositoriesSearchResult)
	if !ok {
		return nil, nil, args.Error(0)
	}
	return result, nil, nil
}

type mockContGetter struct {
	mock.Mock
}

func (m *mockContGetter) ListContributors(ctx context.Context, owner string, repoName string,
	opt *github.ListContributorsOptions) ([]*github.Contributor, *github.Response, error) {

	args := m.Called(ctx, owner, repoName, opt)
	result, ok := args.Get(0).([]*github.Contributor)
	if !ok {
		return nil, nil, args.Error(0)
	}
	return result, nil, nil
}

func TestCommitterService_MostActiveCommitter(t *testing.T) {
	testUser := "test user"
	testRepo := "test repo"

	t.Run("Failed when language is not provided", func(t *testing.T) {
		// Given empty service
		srv := &committerService{}

		// When most active committer is called
		resp, err := srv.MostActiveCommitter(context.TODO(), &pb.CommitterRequest{
			Language: "",
		})

		// Then an error is returned
		require.Error(t, err)
		require.Nil(t, resp)
	})

	t.Run("Failed when repositories query filed", func(t *testing.T) {
		// Given empty context
		ctx := context.TODO()

		// And test request
		req := &pb.CommitterRequest{
			Language: "testlanguage",
		}

		// And mocked repository getter
		rg := &mockRepoGetter{}
		rg.On("Repositories", ctx, "language:"+req.Language, &github.SearchOptions{
			Sort:  "stars",
			Order: "desc",
			ListOptions: github.ListOptions{
				Page:    0,
				PerPage: 5,
			},
		}).Return(errors.New("test error"))

		// And service with repoGetter
		srv := &committerService{
			logger:     zap.L(),
			repoGetter: rg,
		}

		// When most active committer is called
		_, err := srv.MostActiveCommitter(ctx, req)

		// Then an error is returned
		require.Error(t, err)

		assert.Contains(t, err.Error(), "Failed at finding projects")
	})

	t.Run("Failed when committer query filed", func(t *testing.T) {
		// Given empty context
		ctx := context.TODO()

		// And test request
		req := &pb.CommitterRequest{
			Language: "testlanguage",
		}

		// And mocked contributor getter
		cg := &mockContGetter{}
		cg.On("ListContributors", ctx, testUser, testRepo, &github.ListContributorsOptions{
			Anon: "true",
			ListOptions: github.ListOptions{
				Page:    0,
				PerPage: 10,
			},
		}).Return(errors.New("test error"))

		// And service with repoGetter
		srv := &committerService{
			logger:             zap.L(),
			repoGetter:         repositoryGetterSetup(ctx, req.Language, testUser, testRepo),
			contributorsGetter: cg,
		}

		// When most active committer is called
		_, err := srv.MostActiveCommitter(ctx, req)

		// Then an error is returned
		require.Error(t, err)

		assert.Contains(t, err.Error(), "Failed at finding contributors")
	})

	t.Run("Return empty response when no repository found", func(t *testing.T) {
		// Given empty context
		ctx := context.TODO()

		// And test request
		req := &pb.CommitterRequest{
			Language: "testlanguage",
		}

		// And mocked repository getter
		rg := &mockRepoGetter{}
		rg.On("Repositories", ctx, "language:"+req.Language, &github.SearchOptions{
			Sort:  "stars",
			Order: "desc",
			ListOptions: github.ListOptions{
				Page:    0,
				PerPage: 5,
			},
		}).Return(&github.RepositoriesSearchResult{
			Repositories: nil,
		})

		// And service with repoGetter
		srv := &committerService{
			logger:     zap.L(),
			repoGetter: rg,
		}

		// When most active committer is called
		resp, err := srv.MostActiveCommitter(ctx, req)

		// Then no error is returned
		require.NoError(t, err)

		assert.Len(t, resp.Contributors, 0)
		assert.Equal(t, req.Language, resp.Language)
	})

	t.Run("Return empty response when no contributors found", func(t *testing.T) {
		// Given empty context
		ctx := context.TODO()

		// And test request
		req := &pb.CommitterRequest{
			Language: "testlanguage",
		}

		// And mocked repository getter
		cg := &mockContGetter{}
		cg.On("ListContributors", ctx, testUser, testRepo, &github.ListContributorsOptions{
			Anon: "true",
			ListOptions: github.ListOptions{
				Page:    0,
				PerPage: 10,
			},
		}).Return(nil)

		// And service with repoGetter and contributorGetter
		srv := &committerService{
			logger:             zap.L(),
			repoGetter:         repositoryGetterSetup(ctx, req.Language, testUser, testRepo),
			contributorsGetter: cg,
		}

		// When most active committer is called
		resp, err := srv.MostActiveCommitter(ctx, req)

		// Then no error is returned
		require.NoError(t, err)

		assert.Len(t, resp.Contributors, 0)
		assert.Equal(t, req.Language, resp.Language)
	})

	t.Run("Return valid response", func(t *testing.T) {
		// Given empty context
		ctx := context.TODO()

		// And test request
		req := &pb.CommitterRequest{
			Language: "testlanguage",
		}

		// And service with repoGetter and contributorGetter
		srv := &committerService{
			logger:             zap.L(),
			repoGetter:         repositoryGetterSetup(ctx, req.Language, testUser, testRepo),
			contributorsGetter: contributorGetterSetup(ctx, testUser, testRepo, 1),
		}

		// When most active committer is called
		resp, err := srv.MostActiveCommitter(ctx, req)

		// Then no error is returned
		require.NoError(t, err)

		assert.Equal(t, &pb.CommitterResponse{
			Language: "testlanguage",
			Contributors: []*pb.Committer{
				{
					Name:    "test user",
					Commits: 1,
				},
			},
		}, resp)
	})
}

func repositoryGetterSetup(ctx context.Context, language, user, repo string) RepositoryGetter {
	rg := &mockRepoGetter{}

	rg.On("Repositories", ctx, "language:"+language, &github.SearchOptions{
		Sort:  "stars",
		Order: "desc",
		ListOptions: github.ListOptions{
			Page:    0,
			PerPage: 5,
		},
	}).Return(&github.RepositoriesSearchResult{
		Repositories: []github.Repository{
			{
				Owner: &github.User{
					Login: &user,
				},
				Name: &repo,
			},
		},
	})

	return rg
}

func contributorGetterSetup(ctx context.Context, user, repo string, contributions int) ContributorsGetter {
	// And mocked contributor getter
	cg := &mockContGetter{}
	cg.On("ListContributors", ctx, user, repo, &github.ListContributorsOptions{
		Anon: "true",
		ListOptions: github.ListOptions{
			Page:    0,
			PerPage: 10,
		},
	}).Return([]*github.Contributor{
		{
			Login:         &user,
			Contributions: &contributions,
		},
	})
	return cg
}
