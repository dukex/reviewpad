// Copyright 2022 Explore.dev Unipessoal Lda. All Rights Reserved.
// Use of this source code is governed by a license that can be
// found in the LICENSE file

package utils

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/go-github/v42/github"
	"github.com/tomnomnom/linkheader"
)

const maxPerPage = int32(100)

func GetPullRequestOwnerName(pullRequest *github.PullRequest) string {
	return pullRequest.Base.Repo.Owner.GetLogin()
}

func GetPullRequestRepoName(pullRequest *github.PullRequest) string {
	return pullRequest.Base.Repo.GetName()
}

func GetPullRequestNumber(pullRequest *github.PullRequest) int {
	return pullRequest.GetNumber()
}

func PaginatedRequest(
	initFn func() interface{},
	reqFn func(interface{}, int) (interface{}, *github.Response, error),
) (interface{}, error) {
	page := 1
	results, resp, err := reqFn(initFn(), page)
	if err != nil {
		return nil, err
	}

	numPages := ParseNumPages(resp)
	page++
	for page <= numPages && resp.NextPage > page {
		results, _, err = reqFn(results, page)
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

func ParseNumPagesFromLink(link string) int {
	urlInfo := linkheader.Parse(link).FilterByRel("last")
	if len(urlInfo) < 1 {
		return 0
	}

	urlData, err := url.Parse(urlInfo[0].URL)
	if err != nil {
		return 0
	}

	numPagesStr := urlData.Query().Get("page")
	if numPagesStr == "" {
		return 0
	}

	numPages, err := strconv.ParseInt(numPagesStr, 10, 32)
	if err != nil {
		return 0
	}

	return int(numPages)
}

//ParseNumPages Given a link header string representing pagination info, returns total number of pages.
func ParseNumPages(resp *github.Response) int {
	link := resp.Header.Get("Link")
	if strings.Trim(link, " ") == "" {
		return 0
	}

	return ParseNumPagesFromLink(link)
}

func GetPullRequestComments(ctx context.Context, client *github.Client, owner string, repo string, number int, opts *github.IssueListCommentsOptions) ([]*github.IssueComment, error) {
	fs, err := PaginatedRequest(
		func() interface{} {
			return []*github.IssueComment{}
		},
		func(i interface{}, page int) (interface{}, *github.Response, error) {
			fls := i.([]*github.IssueComment)
			fs, resp, err := client.Issues.ListComments(ctx, owner, repo, number, &github.IssueListCommentsOptions{
				Sort: opts.Sort,
				Direction: opts.Direction,
				Since: opts.Since,
				ListOptions: github.ListOptions{
					Page:    page,
					PerPage: int(maxPerPage),
				},
			})
			if err != nil {
				return nil, nil, err
			}
			fls = append(fls, fs...)
			return fls, resp, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return fs.([]*github.IssueComment), nil
}

func GetPullRequestFiles(ctx context.Context, client *github.Client, owner string, repo string, number int) ([]*github.CommitFile, error) {
	fs, err := PaginatedRequest(
		func() interface{} {
			return []*github.CommitFile{}
		},
		func(i interface{}, page int) (interface{}, *github.Response, error) {
			fls := i.([]*github.CommitFile)
			fs, resp, err := client.PullRequests.ListFiles(ctx, owner, repo, number, &github.ListOptions{
				Page:    page,
				PerPage: int(maxPerPage),
			})
			if err != nil {
				return nil, nil, err
			}
			fls = append(fls, fs...)
			return fls, resp, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return fs.([]*github.CommitFile), nil
}

func GetPullRequestReviewers(ctx context.Context, client *github.Client, owner string, repo string, number int, opts *github.ListOptions) (*github.Reviewers, error) {
	fs, err := PaginatedRequest(
		func() interface{} {
			return &github.Reviewers{}
		},
		func(i interface{}, page int) (interface{}, *github.Response, error) {
			fls := i.(*github.Reviewers)
			fs, resp, err := client.PullRequests.ListReviewers(ctx, owner, repo, number, &github.ListOptions{
				Page:    page,
				PerPage: int(maxPerPage),
			})
			if err != nil {
				return nil, nil, err
			}
			fls.Users = append(fls.Users, fs.Users...)
			fls.Teams = append(fls.Teams, fs.Teams...)
			return fls, resp, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return fs.(*github.Reviewers), nil
}