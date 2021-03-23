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

package hansoft

import (
	"fmt"
	"sort"
	"time"
	"unsafe"
)

// Hansoft is the interface to the Hansoft SDK
type Hansoft interface {
	Connect(address string, port int, database, user, password string) (Session, error)
	Destroy() error
}

// Session is the interface to a Hansoft connection
type Session interface {
	Destroy() error
	Projects() ([]Project, error)
}

// Project is the interface to a Hansoft project
type Project interface {
	Name() string
	Backlog() Backlog
	Statuses() []Status
	Resources() []Resource
	Milestones() []Milestone
	Sprints() []Sprint
}

// Backlog is the interface to a Hansoft project backlog
type Backlog interface {
	Tasks() ([]Task, error)
	NewTask() (Task, error)
}

// Milestone is the interface to a Hansoft project milestone
type Milestone interface {
	Name() (string, error)
}

// Sprint is the interface to a Hansoft project sprint
type Sprint interface {
	Name() (string, error)
}

// Task is the interface to a Hansoft project task
type Task interface {
	Assignee() (Resource, error)
	Description() (string, error)
	EstimatedDuration() (time.Duration, error)
	Hyperlink() (string, error)
	Milestone() (Milestone, error)
	Priority() (Priority, error)
	SetAssignee(Resource) error
	SetDescription(string) error
	SetEstimatedDuration(time.Duration) error
	SetHyperlink(string) error
	SetMilestone(Milestone) error
	SetPriority(Priority) error
	SetSprint(Sprint) error
	SetStatus(Status) error
	Sprint() (Sprint, error)
	Status() (Status, error)
}

// Resource is the interface to a project developer
type Resource interface {
	Name() string
	Email() string
}

// Status of a task
type Status = string

// Priority is an enumerator of Task priority
type Priority int

// Enumerator values of Priority
const (
	PriorityNone     Priority = 1 // TODO: Map to EHPMTaskAgilePriorityCategory_*
	PriorityVeryLow  Priority = 2 // instead of hardcoding
	PriorityLow      Priority = 3
	PriorityMedium   Priority = 4
	PriorityHigh     Priority = 5
	PriorityVeryHigh Priority = 6
)

type hansoft struct {
	sdk *sdk
}

func (h *hansoft) Connect(address string, port int, database, user, password string) (Session, error) {
	s := &session{sdk: h.sdk}
	s.sessionProcess.events = make(chan struct{}, 256)
	s.sessionProcess.done = make(chan struct{})
	handle, err := h.sdk.SessionOpen(address, port, database, user, password, s)
	if err != nil {
		return nil, err
	}
	s.handle = handle

	go func() {
		defer close(s.sessionProcess.done)
		for range s.sessionProcess.events {
			if err := s.sdk.SessionProcess(s.handle); err != nil {
				fmt.Println("SessionProcess() returned", err)
			}
		}
	}()

	noMilestoneID, err := s.sdk.UtilGetNoMilestoneID(s.handle)
	if err != nil {
		s.Destroy()
		return nil, fmt.Errorf("UtilGetNoMilestoneID() returned: %w", err)
	}
	s.noMilestoneID = noMilestoneID

	return s, nil
}

func (h *hansoft) Destroy() error {
	h.sdk.Destroy()
	return nil
}

type session struct {
	sdk            *sdk
	handle         unsafe.Pointer
	noMilestoneID  taskRef
	sessionProcess struct {
		events chan struct{}
		done   chan struct{}
	}
}

func (s *session) Destroy() error {
	close(s.sessionProcess.events)
	<-s.sessionProcess.done
	if err := s.sdk.SessionStop(s.handle); err != nil {
		return err
	}
	return s.sdk.SessionClose(s.handle, s)
}

func (s *session) Projects() ([]Project, error) {
	ids, err := s.sdk.ProjectEnum(s.handle)
	if err != nil {
		return nil, err
	}
	out := make([]Project, len(ids))
	for i, id := range ids {
		backlogID, err := s.sdk.ProjectUtilGetBacklog(s.handle, id)
		if err != nil {
			return nil, err
		}
		props, err := s.sdk.ProjectGetProperties(s.handle, id)
		if err != nil {
			return nil, err
		}
		workflowIDs, err := s.sdk.ProjectWorkflowEnum(s.handle, id)
		if err != nil {
			return nil, err
		}
		statusToIDs := map[Status]int{}
		idToStatus := map[int]Status{}
		for workflowID := range workflowIDs {
			statusByString, err := s.sdk.ProjectWorkflowGetStatuses(s.handle, id, workflowID)
			if err != nil {
				return nil, err
			}
			for i, s := range statusByString {
				statusToIDs[Status(s)] = i
				idToStatus[i] = Status(s)
			}
		}
		resourceIDs, err := s.sdk.ProjectResourceEnum(s.handle, id)
		if err != nil {
			return nil, err
		}
		resources := map[uniqueID]Resource{}
		for _, id := range resourceIDs {
			r, err := s.sdk.ResourceGetProperties(s.handle, id)
			if err != nil {
				return nil, err
			}
			resources[r.id] = r
		}
		milestoneRefs, err := s.sdk.ProjectGetMilestones(s.handle, id)
		if err != nil {
			return nil, err
		}
		sprintIDs, err := s.sdk.ProjectGetSprints(s.handle, id)
		if err != nil {
			return nil, err
		}
		project := &project{s, id, nil, props, statusToIDs, idToStatus, resources, map[uniqueID]Milestone{}, map[uniqueID]Sprint{}}
		project.backlog = &backlog{project, backlogID}
		for _, ref := range milestoneRefs {
			id, err := s.sdk.TaskRefGetTask(s.handle, ref)
			if err != nil {
				return nil, fmt.Errorf("Failed to get milestone task ID: %w", err)
			}
			project.milestones[id] = &milestone{project, ref, id}
		}
		for _, id := range sprintIDs {
			ref, err := s.sdk.TaskGetMainReference(s.handle, id)
			if err != nil {
				return nil, fmt.Errorf("Failed to get sprint ref: %w", err)
			}
			project.sprints[id] = &sprint{project, ref, id}
		}
		out[i] = project
	}

	return out, nil
}

func (s *session) onProcessCallback() {
	s.sessionProcess.events <- struct{}{}
}

type project struct {
	session     *session
	id          uniqueID
	backlog     *backlog
	properties  projectProperties
	statusToIDs map[Status]int
	idToStatus  map[int]Status
	resources   map[uniqueID]Resource
	milestones  map[uniqueID]Milestone
	sprints     map[uniqueID]Sprint
}

func (p *project) Name() string {
	return p.properties.name
}

func (p *project) Backlog() Backlog {
	return p.backlog
}

func (p *project) Statuses() []Status {
	out := make([]Status, 0, len(p.statusToIDs))
	for s := range p.statusToIDs {
		out = append(out, s)
	}
	sort.Strings([]string(out))
	return out
}

func (p *project) Resources() []Resource {
	out := make([]Resource, 0, len(p.resources))
	for _, r := range p.resources {
		out = append(out, r)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Name() < out[b].Name() })
	return out
}

func (p *project) Milestones() []Milestone {
	out := make([]Milestone, 0, len(p.milestones))
	for _, r := range p.milestones {
		out = append(out, r)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].(*milestone).id < out[b].(*milestone).id })
	return out
}

func (p *project) Sprints() []Sprint {
	out := make([]Sprint, 0, len(p.sprints))
	for _, r := range p.sprints {
		out = append(out, r)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].(*sprint).ref < out[b].(*sprint).ref })
	return out
}

type milestone struct {
	project *project
	ref     taskRef
	id      uniqueID
}

func (m *milestone) Name() (string, error) {
	description, err := m.project.session.sdk.TaskGetDescription(m.project.session.handle, m.id)
	if err != nil {
		return "", fmt.Errorf("Failed to get milestone name: %w", err)
	}
	return description, nil
}

type sprint struct {
	project *project
	ref     taskRef
	id      uniqueID
}

func (m *sprint) Name() (string, error) {
	description, err := m.project.session.sdk.TaskGetDescription(m.project.session.handle, m.id)
	if err != nil {
		return "", fmt.Errorf("Failed to get sprint name: %w", err)
	}
	return description, nil
}

type backlog struct {
	project *project
	id      uniqueID
}

func (b *backlog) Tasks() ([]Task, error) {
	refs, err := b.project.session.sdk.TaskRefEnum(b.project.session.handle, b.id)
	if err != nil {
		return nil, err
	}
	out := make([]Task, len(refs))
	for i, ref := range refs {
		id, err := b.project.session.sdk.TaskRefGetTask(b.project.session.handle, ref)
		if err != nil {
			return nil, fmt.Errorf("Failed to get task ID: %w", err)
		}
		out[i] = &task{b.project, id, ref}
	}
	return out, nil
}

func (b *backlog) NewTask() (Task, error) {
	ref, err := b.project.session.sdk.TaskCreateUnified(b.project.session.handle, b.id, unifiedTaskPlanned)
	if err != nil {
		return nil, err
	}
	id, err := b.project.session.sdk.TaskRefGetTask(b.project.session.handle, ref)
	if err != nil {
		return nil, fmt.Errorf("Failed to get task ID: %w", err)
	}
	return &task{b.project, id, ref}, nil
}

type task struct {
	project *project
	id      uniqueID // Database id
	ref     taskRef
}

func (t *task) Description() (string, error) {
	description, err := t.project.session.sdk.TaskGetDescription(t.project.session.handle, t.id)
	if err != nil {
		return "", err
	}
	return description, nil
}

func (t *task) SetDescription(description string) error {
	return t.project.session.sdk.TaskSetDescription(t.project.session.handle, t.id, description)
}

func (t *task) Status() (Status, error) {
	s, err := t.project.session.sdk.TaskGetWorkflowStatus(t.project.session.handle, t.id)
	if err != nil {
		return "", nil
	}
	return t.project.idToStatus[s], nil
}

func (t *task) SetStatus(status Status) error {
	if id, ok := t.project.statusToIDs[status]; ok {
		return t.project.session.sdk.TaskSetWorkflowStatus(t.project.session.handle, t.id, id)
	}
	return fmt.Errorf("Unrecognized status '%v'", status)
}

func (t *task) Milestone() (Milestone, error) {
	refs, err := t.project.session.sdk.TaskGetLinkedToMilestones(t.project.session.handle, t.id)
	if err != nil {
		return nil, fmt.Errorf("TaskGetLinkedToMilestones() returned %w", err)
	}
	if len(refs) > 0 {
		ref := refs[0]
		if ref == t.project.session.noMilestoneID {
			return nil, nil
		}
		id, err := t.project.session.sdk.TaskRefGetTask(t.project.session.handle, refs[0])
		if err != nil {
			return nil, fmt.Errorf("Failed to get milestone task ID: %w", err)
		}
		return t.project.milestones[id], nil
	}
	return nil, nil
}

func (t *task) SetMilestone(m Milestone) error {
	var ids []taskRef
	if m != nil {
		ids = []taskRef{m.(*milestone).ref}
	} else {
		ids = []taskRef{t.project.session.noMilestoneID}
	}
	return t.project.session.sdk.TaskSetLinkedToMilestones(t.project.session.handle, t.id, ids)
}

func (t *task) Sprint() (Sprint, error) {
	ref, err := t.project.session.sdk.TaskGetLinkedToSprint(t.project.session.handle, t.id)
	if err != nil {
		return nil, fmt.Errorf("TaskGetLinkedToSprints() returned %w", err)
	}
	if ref == -1 {
		return nil, nil
	}
	id, err := t.project.session.sdk.TaskRefGetTask(t.project.session.handle, ref)
	if err != nil {
		return nil, fmt.Errorf("TaskGetMainReference() returned %w", err)
	}
	return t.project.sprints[id], nil
}

func (t *task) SetSprint(s Sprint) error {
	if s != nil {
		ref := s.(*sprint).ref
		return t.project.session.sdk.TaskSetLinkedToSprint(t.project.session.handle, t.project.id, t.ref, ref)
	}
	return nil
}

func (t *task) Priority() (Priority, error) {
	return t.project.session.sdk.TaskGetBacklogPriority(t.project.session.handle, t.id)
}

func (t *task) SetPriority(p Priority) error {
	return t.project.session.sdk.TaskSetBacklogPriority(t.project.session.handle, t.id, p)
}

const hoursInWorkingDay = 8

func (t *task) EstimatedDuration() (time.Duration, error) {
	days, err := t.project.session.sdk.TaskGetEstimatedIdealDays(t.project.session.handle, t.id)
	if err != nil {
		return 0, err
	}
	return time.Duration(days * hoursInWorkingDay * float64(time.Hour)), nil
}

func (t *task) SetEstimatedDuration(duration time.Duration) error {
	days := 0.0
	if duration != 0 {
		days = float64(duration) / float64(time.Hour*hoursInWorkingDay)
	}
	return t.project.session.sdk.TaskSetEstimatedIdealDays(t.project.session.handle, t.id, days)
}

func (t *task) Assignee() (Resource, error) {
	allocations, err := t.project.session.sdk.TaskGetResourceAllocation(t.project.session.handle, t.id)
	if err != nil {
		return nil, err
	}
	if len(allocations) == 0 {
		return nil, nil
	}
	sort.Slice(allocations, func(a, b int) bool { return allocations[a].percent > allocations[b].percent })
	return t.project.resources[allocations[0].resource], nil
}

func (t *task) SetAssignee(r Resource) error {
	var allocations []allocation
	if r != nil {
		allocations = []allocation{
			{
				resource: r.(resource).id,
				percent:  100,
			},
		}
	}
	return t.project.session.sdk.TaskSetResourceAllocation(t.project.session.handle, t.id, allocations)
}

func (t *task) Hyperlink() (string, error) {
	return t.project.session.sdk.TaskGetHyperlink(t.project.session.handle, t.id)
}

func (t *task) SetHyperlink(link string) error {
	return t.project.session.sdk.TaskSetHyperlink(t.project.session.handle, t.id, link)
}

type resource struct {
	id    uniqueID
	name  string
	email string
}

func (r resource) Name() string  { return r.name }
func (r resource) Email() string { return r.email }

// New constructs and returns a new Hansoft
func New() (Hansoft, error) {
	s := &sdk{}
	if err := s.Init("third_party/hansoft_sdk/Linux2.6"); err != nil {
		return nil, err
	}
	return &hansoft{s}, nil
}
