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

package projectsync

import (
	"fmt"
	"log"
	"mhs/src/hansoft"
	"mhs/src/monorail"
	"strconv"
	"strings"
	"time"
)

type sync struct {
	h                hansoft.Project
	m                monorail.Project
	crbugPrefix      string
	statusMtoH       map[monorail.Status]hansoft.Status
	statusHtoM       map[hansoft.Status]monorail.Status
	priorityMtoH     map[monorail.Priority]hansoft.Priority
	priorityHtoM     map[hansoft.Priority]monorail.Priority
	milestones       map[string]hansoft.Milestone
	sprints          map[string]hansoft.Sprint
	resourcesByEmail map[string]hansoft.Resource
}

func alternativeEmail(email string) string {
	if idx := strings.IndexRune(email, '@'); idx > 0 {
		name, domain := email[:idx], email[idx:]
		if domain == "@google.com" {
			return name + "@chromium.org"
		} else if domain == "@chromium.com" {
			return name + "@google.com"
		}
	}
	return email
}

// Sync performs a two-way synchronization of the monorail and hansoft projects
func Sync(m monorail.Project, h hansoft.Project) error {
	s := sync{
		m:           m,
		h:           h,
		crbugPrefix: "crbug.com/" + m.Name() + "/",
		statusMtoH: map[monorail.Status]hansoft.Status{
			monorail.StatusAccepted:  "Assigned",
			monorail.StatusDone:      "Resolved",
			monorail.StatusDuplicate: "Closed",
			monorail.StatusFixed:     "Resolved",
			monorail.StatusInvalid:   "Closed",
			monorail.StatusNew:       "New",
			monorail.StatusStarted:   "Assigned",
			monorail.StatusVerified:  "Verified",
			monorail.StatusWontFix:   "Closed",
		},
		statusHtoM: map[hansoft.Status]monorail.Status{
			"Assigned": monorail.StatusAccepted,
			"Closed":   monorail.StatusFixed,
			"New":      monorail.StatusNew,
			"Resolved": monorail.StatusFixed,
			"Verified": monorail.StatusVerified,
		},
		priorityMtoH: map[monorail.Priority]hansoft.Priority{
			"":                        hansoft.PriorityNone,
			monorail.PriorityLow:      hansoft.PriorityLow,
			monorail.PriorityMedium:   hansoft.PriorityMedium,
			monorail.PriorityHigh:     hansoft.PriorityHigh,
			monorail.PriorityCritical: hansoft.PriorityVeryHigh,
		},
		priorityHtoM: map[hansoft.Priority]monorail.Priority{
			hansoft.PriorityNone:     "",
			hansoft.PriorityVeryLow:  monorail.PriorityLow,
			hansoft.PriorityLow:      monorail.PriorityLow,
			hansoft.PriorityMedium:   monorail.PriorityMedium,
			hansoft.PriorityHigh:     monorail.PriorityHigh,
			hansoft.PriorityVeryHigh: monorail.PriorityCritical,
		},
		milestones:       map[string]hansoft.Milestone{},
		sprints:          map[string]hansoft.Sprint{},
		resourcesByEmail: map[string]hansoft.Resource{},
	}
	for _, r := range h.Resources() {
		if email := r.Email(); email != "" {
			s.resourcesByEmail[email] = r
			s.resourcesByEmail[alternativeEmail(email)] = r
		}
	}
	for _, m := range h.Milestones() {
		name, err := m.Name()
		if err != nil {
			return err
		}
		s.milestones[name] = m
	}
	for _, m := range h.Sprints() {
		name, err := m.Name()
		if err != nil {
			return err
		}
		s.sprints[name] = m
	}

	hIssues, err := s.gatherHansoftIssues(h)
	if err != nil {
		return err
	}
	mIssues, err := s.gatherMonorailIssues(m)
	if err != nil {
		return err
	}

	for id, m := range mIssues {
		h, exists := hIssues[id]
		if exists {
			diffs := s.diff(h, m)
			if len(diffs) == 0 {
				continue // in sync
			}
			log.Printf("Updating hansoft task %s%v. Diffs: %v\n", s.crbugPrefix, m.id, diffs)
		} else {
			h = &hIssue{}
			hIssues[id] = h
			log.Printf("Creating hansoft task %s%v: %v\n", s.crbugPrefix, m.id, m.summary)
		}

		if status, ok := s.statusMtoH[m.status]; ok {
			h.status = status
		} else {
			warn("Don't know how to translate monorail status '%v' to hansoft", m.status)
		}

		if priority, ok := s.priorityMtoH[m.priority]; ok {
			h.priority = priority
		} else {
			warn("Don't know how to translate monorail priority '%v' to hansoft", m.priority)
			h.priority = hansoft.PriorityMedium
		}

		if m.milestone != "" {
			h.milestone = s.milestones[m.milestone]
			if h.milestone == nil {
				warn("Hansoft does not contain sprint '%v'", m.sprint)
			}
		}

		if m.sprint != "" {
			h.sprint = s.sprints[m.sprint]
			if h.sprint == nil {
				warn("Hansoft does not contain sprint '%v'", m.sprint)
			}
		}

		h.id = m.id
		h.summary = m.summary
		h.assignee = s.resourcesByEmail[m.assignee]
		h.estimatedDuration = m.estimatedDuration

		if h.assignee == nil && m.assignee != "" && !m.status.IsClosed() {
			warn("Hansoft project does not have a user with address '%v'", m.assignee)
		}

		if err := s.updateHansoftIssue(h); err != nil {
			warn("%v", err)
		}
	}

	return nil
}

func (s *sync) diff(h *hIssue, m *mIssue) []issueDiff {
	diffs := []issueDiff{}
	if h.id != m.id {
		diffs = append(diffs, diffID)
	}
	if h.summary != m.summary {
		diffs = append(diffs, diffSummary)
	}
	if h.status != s.statusMtoH[m.status] {
		diffs = append(diffs, diffStatus)
	}
	if h.assignee != s.resourcesByEmail[m.assignee] && !m.status.IsClosed() {
		diffs = append(diffs, diffAssignee)
	}
	if (h.estimatedDuration / time.Minute) != (m.estimatedDuration / time.Minute) {
		diffs = append(diffs, diffDuration)
	}
	if h.priority != s.priorityMtoH[m.priority] {
		diffs = append(diffs, diffPriority)
	}
	if h.milestone != s.milestones[m.milestone] && !m.status.IsClosed() {
		diffs = append(diffs, diffMilestone)
	}
	if h.sprint != s.sprints[m.sprint] && !m.status.IsClosed() {
		diffs = append(diffs, diffSprint)
	}
	return diffs
}

func (s *sync) updateHansoftIssue(i *hIssue) error {
	if i.Task == nil {
		task, err := s.h.Backlog().NewTask()
		if err != nil {
			return fmt.Errorf("Failed to create new hansoft task: %w", err)
		}
		i.Task = task
	}
	if err := i.Task.SetDescription(i.summary); err != nil {
		return fmt.Errorf("Failed to set new hansoft task description: %w", err)
	}
	hyperlink := fmt.Sprintf("%s%v", s.crbugPrefix, i.id)
	if err := i.Task.SetHyperlink(hyperlink); err != nil {
		return fmt.Errorf("Failed to set new hansoft task hyperlink: %w", err)
	}
	if err := i.Task.SetAssignee(i.assignee); err != nil {
		return fmt.Errorf("Failed to set new hansoft task assignee: %w", err)
	}
	if err := i.Task.SetStatus(i.status); err != nil {
		return fmt.Errorf("Failed to set new hansoft task status: %w", err)
	}
	if err := i.Task.SetEstimatedDuration(i.estimatedDuration); err != nil {
		return fmt.Errorf("Failed to set new hansoft task status: %w", err)
	}
	if err := i.Task.SetPriority(i.priority); err != nil {
		return fmt.Errorf("Failed to set new hansoft task status: %w", err)
	}
	if err := i.Task.SetMilestone(i.milestone); err != nil {
		return fmt.Errorf("Failed to set new hansoft task milestone: %w", err)
	}
	if err := i.Task.SetSprint(i.sprint); err != nil {
		return fmt.Errorf("Failed to set new hansoft task sprint: %w", err)
	}
	return nil
}

type issueDiff string

const (
	diffID        issueDiff = "id"
	diffSummary   issueDiff = "summary"
	diffAssignee  issueDiff = "assignee"
	diffStatus    issueDiff = "status"
	diffDuration  issueDiff = "duration"
	diffPriority  issueDiff = "priority"
	diffMilestone issueDiff = "milestone"
	diffSprint    issueDiff = "sprint"
)

type hIssue struct {
	hansoft.Task
	id                int
	summary           string
	assignee          hansoft.Resource
	status            hansoft.Status
	estimatedDuration time.Duration
	priority          hansoft.Priority
	milestone         hansoft.Milestone
	sprint            hansoft.Sprint
}

type mIssue struct {
	monorail.Issue
	id                int
	summary           string
	assignee          string
	status            monorail.Status
	estimatedDuration time.Duration
	priority          monorail.Priority
	milestone         string
	sprint            string
}

func (s *sync) gatherHansoftIssues(h hansoft.Project) (map[int]*hIssue, error) {
	tasks, err := h.Backlog().Tasks()
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch hansoft tasks: %w", err)
	}

	out := map[int]*hIssue{}
	for _, task := range tasks {
		hyperlink, err := task.Hyperlink()
		if err != nil {
			warn("Failed to get hansoft task url: %w", err)
			continue
		}
		if !strings.HasPrefix(hyperlink, s.crbugPrefix) {
			continue
		}
		id, err := strconv.Atoi(hyperlink[len(s.crbugPrefix):])
		if err != nil {
			warn("%v: Failed to parse bug ID from hyperlink '%v'", hyperlink, s.crbugPrefix)
			continue
		}
		description, err := task.Description()
		if err != nil {
			warn("%v: Failed to get hansoft task description: %w", hyperlink, err)
			continue
		}
		assignee, err := task.Assignee()
		if err != nil {
			warn("%v: Failed to get hansoft task assignee: %w", hyperlink, err)
			continue
		}
		status, err := task.Status()
		if err != nil {
			warn("%v: Failed to get hansoft task assignee: %w", hyperlink, err)
			continue
		}
		estimatedDuration, err := task.EstimatedDuration()
		if err != nil {
			warn("%v: Failed to get hansoft task estimated duration: %w", hyperlink, err)
			continue
		}
		priority, err := task.Priority()
		if err != nil {
			warn("%v: Failed to get hansoft task priority: %w", hyperlink, err)
			continue
		}
		milestone, err := task.Milestone()
		if err != nil {
			warn("%v: Failed to get hansoft task milestone: %w", hyperlink, err)
			continue
		}
		sprint, err := task.Sprint()
		if err != nil {
			warn("%v: Failed to get hansoft task sprint: %w", hyperlink, err)
			continue
		}
		out[id] = &hIssue{
			Task:              task,
			id:                id,
			summary:           description,
			assignee:          assignee,
			status:            status,
			estimatedDuration: estimatedDuration,
			priority:          priority,
			milestone:         milestone,
			sprint:            sprint,
		}
	}
	return out, nil
}

func (s *sync) gatherMonorailIssues(m monorail.Project) (map[int]*mIssue, error) {
	issues, err := m.Issues()
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch monorail issues: %w", err)
	}

	out := map[int]*mIssue{}
	for _, i := range issues {
		id := i.ID()
		out[id] = &mIssue{
			Issue:             i,
			id:                id,
			summary:           i.Summary(),
			assignee:          i.Assignee(),
			status:            i.Status(),
			estimatedDuration: i.EstimatedDuration(),
			priority:          i.Priority(),
			milestone:         i.Milestone(),
			sprint:            i.Sprint(),
		}
	}
	return out, nil
}

func warn(msg string, args ...interface{}) {
	err := fmt.Errorf(msg, args...)
	log.Printf("warning: %v\n", err)
}
