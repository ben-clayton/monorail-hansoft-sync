// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package monorail

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	monorailv3 "chromium.googlesource.com/infra/infra.git/go/src/infra/monorailv2/api/v3/api_proto"

	"go.chromium.org/luci/auth"
	"go.chromium.org/luci/grpc/prpc"
)

const (
	fieldEstimatedTime = "EstimateTime"
)

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {

	return nil
}

// New constructs and returns a new Monorail
func New(authJSONPath string) (Monorail, error) {
	ctx := context.Background()
	authenticator := auth.NewAuthenticator(ctx, auth.InteractiveLogin, auth.Options{
		ServiceAccountJSONPath: authJSONPath,
		Audience:               "https://monorail-prod.appspot.com",
		UseIDTokens:            true,
	})

	httpClient, err := authenticator.Client()
	if err != nil {
		return nil, err
	}

	prpcClient := &prpc.Client{C: httpClient, Host: "api-dot-monorail-prod.appspot.com"}

	issuesClient := monorailv3.NewIssuesPRPCClient(prpcClient)
	usersClient := monorailv3.NewUsersPRPCClient(prpcClient)
	frontendClient := monorailv3.NewFrontendPRPCClient(prpcClient)

	return &mr{issuesClient, usersClient, frontendClient}, nil
}

// Monorail is the interface to the monorail API
type Monorail interface {
	Project(name string) (Project, error)
}

// Project is the interface to a monorail project
type Project interface {
	Name() string
	Issues() ([]Issue, error)
}

// Issue is the interface to a single issue
type Issue interface {
	ID() int
	Summary() string
	Assignee() string
	Status() Status
	EstimatedDuration() time.Duration
	Priority() Priority
	Milestone() string
	Sprint() string
}

type mr struct {
	issuesClient   monorailv3.IssuesClient
	usersClient    monorailv3.UsersClient
	frontendClient monorailv3.FrontendClient
}

func (m *mr) Project(name string) (Project, error) {
	ctx := context.Background()
	monorailName := "projects/" + name

	fieldDefs := map[string]*monorailv3.FieldDef{}
	{
		request := &monorailv3.GatherProjectEnvironmentRequest{Parent: monorailName}
		response, err := m.frontendClient.GatherProjectEnvironment(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("GatherProjectEnvironment returned %w", err)
		}
		for _, field := range response.Fields {
			fieldDefs[field.GetName()] = field
		}
	}

	return &project{m, name, monorailName, fieldDefs}, nil
}

type project struct {
	m            *mr
	name         string
	monorailName string
	fieldDefs    map[string]*monorailv3.FieldDef
}

func (p *project) Name() string { return p.name }

func (p *project) Issues() ([]Issue, error) {
	ctx := context.Background()

	issuesRequest := monorailv3.SearchIssuesRequest{
		Projects: []string{p.monorailName},
	}

	userIDsToEmail := map[string]string{}

	issues := []*issue{}
	namePrefix := p.monorailName + "/issues/"

	for {
		issuesResponse, err := p.m.issuesClient.SearchIssues(ctx, &issuesRequest)
		if err != nil {
			return nil, err
		}
		fmt.Println("issues returned: ", len(issuesResponse.Issues))

		for _, item := range issuesResponse.Issues {
			name := item.GetName()
			if !strings.HasPrefix(name, namePrefix) {
				return nil, fmt.Errorf("Expected issue '%v' to have '%v' prefix", name, namePrefix)
			}
			idStr := name[len(namePrefix):]
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return nil, fmt.Errorf("Failed to parse issue ID from '%v'", idStr)
			}
			estimatedHours := 0
			for _, field := range item.FieldValues {
				if def, ok := p.fieldDefs[field.GetField()]; ok {
					if def.GetDisplayName() == fieldEstimatedTime {
						estimatedHours, _ = strconv.Atoi(field.GetValue())
						break
					}
				}
			}
			priority := PriorityMedium
			milestone := ""
			sprint := ""
			for _, label := range item.GetLabels() {
				if parts := strings.Split(label.GetLabel(), "-"); len(parts) == 2 {
					key, val := parts[0], parts[1]
					switch key {
					case "Priority":
						priority = Priority(val)
						continue
					case "Milestone":
						milestone = val
						continue
					case "Sprint":
						sprint = val
						continue
					}
				}
			}
			userID := item.GetOwner().GetUser()
			issues = append(issues, &issue{
				id:                id,
				summary:           item.GetSummary(),
				assignee:          userID, // We'll transform this to email address in a second pass
				status:            Status(item.GetStatus().GetStatus()),
				estimatedDuration: time.Hour * time.Duration(estimatedHours),
				priority:          priority,
				milestone:         milestone,
				sprint:            sprint,
			})
			if userID != "" {
				userIDsToEmail[userID] = ""
			}
		}
		if issuesResponse.GetNextPageToken() == "" {
			break
		}
		issuesRequest.PageToken = issuesResponse.GetNextPageToken()
	}
	usersRequest := &monorailv3.BatchGetUsersRequest{}
	for id := range userIDsToEmail {
		if id != "" {
			usersRequest.Names = append(usersRequest.Names, id)
		}
	}
	usersResponse, err := p.m.usersClient.BatchGetUsers(ctx, usersRequest)
	if err != nil {
		return nil, fmt.Errorf("BatchGetUsers() returned %w", err)
	}
	for _, user := range usersResponse.Users {
		userIDsToEmail[user.GetName()] = user.GetEmail()
	}

	out := make([]Issue, len(issues))
	for i, issue := range issues {
		if issue.assignee != "" {
			// Remap assignee ID to email address
			email, ok := userIDsToEmail[issue.assignee]
			if !ok {
				return nil, fmt.Errorf("Couldn't resolve email address of '%v'", issue.assignee)
			}
			issue.assignee = email
		}
		out[i] = issue
	}

	return out, nil
}

type issue struct {
	id                int
	summary           string
	assignee          string
	status            Status
	estimatedDuration time.Duration
	priority          Priority
	milestone         string
	sprint            string
}

func (i issue) ID() int                          { return i.id }
func (i issue) Summary() string                  { return i.summary }
func (i issue) Assignee() string                 { return i.assignee }
func (i issue) Status() Status                   { return i.status }
func (i issue) EstimatedDuration() time.Duration { return i.estimatedDuration }
func (i issue) Priority() Priority               { return i.priority }
func (i issue) Milestone() string                { return i.milestone }
func (i issue) Sprint() string                   { return i.sprint }

// Status is an enumerator of issue status
type Status string

// Enumerator values of Status
const (
	StatusAccepted  Status = "Accepted"
	StatusDone      Status = "Done"
	StatusDuplicate Status = "Duplicate"
	StatusFixed     Status = "Fixed"
	StatusInvalid   Status = "Invalid"
	StatusNew       Status = "New"
	StatusStarted   Status = "Started"
	StatusVerified  Status = "Verified"
	StatusWontFix   Status = "WontFix"
)

// IsClosed returns true if the status is of a closed state
func (s Status) IsClosed() bool {
	switch s {
	case StatusDone, StatusDuplicate, StatusFixed,
		StatusInvalid, StatusVerified, StatusWontFix:
		return true
	}
	return false
}

// Priority is an enumerator of issue priority
type Priority string

// Enumerator values of Priority
const (
	PriorityLow      Priority = "Low"
	PriorityMedium   Priority = "Medium"
	PriorityHigh     Priority = "High"
	PriorityCritical Priority = "Critical"
)
