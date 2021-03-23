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

// #cgo LDFLAGS: -ldl
// #include "sdk.h"
import "C"

import (
	"fmt"
	"unsafe"
)

type uniqueID int32
type taskRef int32

type sdk struct{ funcs C.HPMSdkFunctions }

func (s *sdk) Init(libraryDirectory string) error {
	libDir := C.CString(libraryDirectory)
	defer C.free(unsafe.Pointer(libDir))
	if err := toError(C.HPMInit(&s.funcs, nil, libDir)); err != nil {
		return err
	}
	return nil
}

func (s *sdk) Destroy() {
	C.HPMDestroy(&s.funcs)
}

type processCallbackHandler interface {
	onProcessCallback()
}

func (s *sdk) SessionOpen(
	address string,
	port int,
	database,
	user,
	password string,
	onProcessCallback processCallbackHandler) (unsafe.Pointer, error) {

	addr := C.CString(address)
	defer C.free(unsafe.Pointer(addr))
	db := C.CString(database)
	defer C.free(unsafe.Pointer(db))
	usr := C.CString(user)
	defer C.free(unsafe.Pointer(usr))
	pw := C.CString(password)
	defer C.free(unsafe.Pointer(pw))
	e := C.HPMError(0)
	callbackHandle := registerProcessCallbackHandler(onProcessCallback)
	callbackInfo := C.HPMNeedSessionProcessCallbackInfo{
		m_pContext:  unsafe.Pointer(callbackHandle),
		m_pCallback: C.HPMNeedSessionProcessCallback(C.onProcessCallback),
	}
	emptyString := C.CString("")
	defer C.free(unsafe.Pointer(emptyString))

	session := C.session_open(&s.funcs,
		/* pError */ &e,
		/* pAddress */ addr,
		/* Port */ C.int(port),
		/* pDatabase */ db,
		/* pResourceName */ usr,
		/* pPassword */ pw,
		/* bBlockOnOperations */ C.int(1),
		/* pNeedProcessCallback */ &callbackInfo,
		/* SDKVersion */ C.EHPMSDK_Version,
		/* SDKDebug */ C.EHPMSdkDebugMode_Off,
		/* nSessions */ 0,
		/* pWorkingDirectory */ emptyString,
		/* pCertificateSettings */ nil,
		/* pExtendedErrorMessage */ nil)
	if err := toError(e); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *sdk) SessionStop(session unsafe.Pointer) error {
	return toError(C.session_stop(&s.funcs, session))
}

func (s *sdk) SessionClose(session unsafe.Pointer, onProcessCallback processCallbackHandler) error {
	unregisterProcessCallbackHandler(onProcessCallback)
	return toError(C.session_close(&s.funcs, session))
}

func (s *sdk) SessionProcess(session unsafe.Pointer) error {
	return toError(C.session_process(&s.funcs, session))
}

func (s *sdk) ProjectEnum(session unsafe.Pointer) ([]uniqueID, error) {
	var e *C.HPMProjectEnum
	if err := toError(C.project_enum(&s.funcs, session, &e)); err != nil {
		return nil, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(e), nil)

	out := make([]uniqueID, e.m_nProjects)
	ptr := uintptr(unsafe.Pointer(e.m_pProjects))
	for i := range out {
		out[i] = *(*uniqueID)(unsafe.Pointer(ptr + uintptr(i*4)))
	}
	return out, nil
}

func (s *sdk) ProjectUtilGetBacklog(session unsafe.Pointer, project uniqueID) (uniqueID, error) {
	var backlog C.HPMUniqueID
	if err := toError(C.project_util_get_backlog(&s.funcs, session, C.HPMUniqueID(project), &backlog)); err != nil {
		return 0, err
	}
	return uniqueID(backlog), nil
}

type projectProperties struct {
	// The name of the project.
	name string
	// The nice name of the project. A name that is safe to use in the file system. Converted from m_pName internally and is read-only (changes will be ignored by server).
	niceName string
	// If set this name is used for sorting instead of the project name.
	sortName string
	// [default=0] Archived status. Set to 1 to indicate that the project is archived. An archived project is kept in the database but is not synchronized to resources connecting to the database saving network bandwidth and memory on the client.
	archivedStatus int32
	// [type=EHPMProjectMethod,default=EHPMProjectMethod_FixedDuration] The project method used in the project. Can be one of @{EHPMProjectMethod}.
	projectMethod int32
	// [type=EHPMProjectTaskCompletionStyle,default=EHPMProjectTaskCompletionStyle_Auto] The completion style used in the project. Can be one of @{EHPMProjectTaskCompletionStyle}.
	completionStyle int32
	// [type=EHPMProjectDefaultEditorMode,default=EHPMProjectDefaultEditorMode_Agile] The default editor mode for the project. Can be one of @{EHPMProjectDefaultEditorMode}.
	defaultEditorMode int32
	// [type=EHPMProjectAgileTemplate,default=EHPMProjectAgileTemplate_SCRUM] The agile template used for the project. Can be one of @{EHPMProjectAgileTemplate}.
	agileTemplate int32
}

func (s *sdk) ProjectGetProperties(session unsafe.Pointer, project uniqueID) (projectProperties, error) {
	var properties *C.HPMProjectProperties
	if err := toError(C.project_get_properties(&s.funcs, session, C.HPMUniqueID(project), &properties)); err != nil {
		return projectProperties{}, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(properties), nil)

	return projectProperties{
		name:              C.GoString(properties.m_pName),
		niceName:          C.GoString(properties.m_pNiceName),
		sortName:          C.GoString(properties.m_pSortName),
		archivedStatus:    int32(properties.m_bArchivedStatus),
		projectMethod:     int32(properties.m_ProjectMethod),
		completionStyle:   int32(properties.m_CompletionStyle),
		defaultEditorMode: int32(properties.m_DefaultEditorMode),
		agileTemplate:     int32(properties.m_AgileTemplate),
	}, nil
}

func (s *sdk) ProjectGetMilestones(session unsafe.Pointer, project uniqueID) ([]taskRef, error) {
	var m *C.HPMProjectMilestones
	if err := toError(C.project_get_milestones(&s.funcs, session, C.HPMUniqueID(project), &m)); err != nil {
		return nil, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(m), nil)

	out := make([]taskRef, m.m_nMilestones)
	ptr := uintptr(unsafe.Pointer(m.m_pMilestones))
	for i := range out {
		out[i] = taskRef(*(*C.HPMUniqueID)(unsafe.Pointer(ptr + uintptr(i*4))))
	}
	return out, nil
}

func (s *sdk) ProjectGetSprints(session unsafe.Pointer, project uniqueID) ([]uniqueID, error) {
	var sprints *C.HPMProjectSprints
	if err := toError(C.project_get_sprints(&s.funcs, session, C.HPMUniqueID(project), &sprints)); err != nil {
		return nil, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(sprints), nil)

	out := make([]uniqueID, sprints.m_nSprints)
	ptr := uintptr(unsafe.Pointer(sprints.m_pSprints))
	for i := range out {
		out[i] = uniqueID(*(*C.HPMUniqueID)(unsafe.Pointer(ptr + uintptr(i*4))))
	}
	return out, nil
}

type projectCustomColumn struct {
	hash int32
	name string
	unit string
}

func (s *sdk) ProjectCustomColumnsGet(session unsafe.Pointer, project uniqueID) ([]projectCustomColumn, error) {
	var columns *C.HPMProjectCustomColumns
	if err := toError(C.project_custom_columns_get(&s.funcs, session, C.HPMUniqueID(project), &columns)); err != nil {
		return nil, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(columns), nil)

	out := make([]projectCustomColumn, 0, columns.m_nHiddenColumns+columns.m_nShowingColumns)
	add := func(l *C.HPMProjectCustomColumnsColumn, n C.HPMUInt32) {
		ptr := uintptr(unsafe.Pointer(l))
		for i := 0; i < int(n); i++ {
			column := (*C.HPMProjectCustomColumnsColumn)(unsafe.Pointer(ptr))
			out = append(out, projectCustomColumn{
				hash: int32(column.m_Hash),
			})
			ptr += unsafe.Sizeof(C.HPMProjectCustomColumnsColumn{})
		}
	}
	add(columns.m_pHiddenColumns, columns.m_nHiddenColumns)
	add(columns.m_pHiddenColumns, columns.m_nShowingColumns)
	return out, nil
}

func (s *sdk) ProjectWorkflowEnum(session unsafe.Pointer, project uniqueID) ([]int, error) {
	var e *C.HPMProjectWorkflowEnum
	if err := toError(C.project_workflow_enum(&s.funcs, session, C.HPMUniqueID(project), C.HPMUInt32(1), &e)); err != nil {
		return nil, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(e), nil)

	out := make([]int, e.m_nWorkflows)
	ptr := uintptr(unsafe.Pointer(e.m_pWorkflows))
	for i := range out {
		out[i] = int(*(*C.HPMUInt32)(unsafe.Pointer(ptr + uintptr(i*4))))
	}
	return out, nil
}

func (s *sdk) ProjectWorkflowGetStatuses(session unsafe.Pointer, project uniqueID, workflow int) (map[int]string, error) {
	var settings *C.HPMProjectWorkflowSettings
	if err := toError(C.project_workflow_get_settings(&s.funcs, session, C.HPMUniqueID(project), C.HPMUInt32(workflow), &settings)); err != nil {
		return nil, err
	}
	out := map[int]string{}
	ptr := uintptr(unsafe.Pointer(settings.m_pWorkflowObjects))
	for i := 0; i < int(settings.m_nWorkflowObjects); i++ {
		object := (*C.HPMProjectWorkflowObject)(unsafe.Pointer(ptr))
		if object.m_ObjectType == C.EHPMProjectWorkflowObjectType_WorkflowStatus {
			name, err := s.translateString(session, object.m_WorkflowStatus_pName)
			if err != nil {
				return nil, err
			}
			out[int(object.m_ObjectID)] = name
		}
		ptr += unsafe.Sizeof(C.HPMProjectWorkflowObject{})
	}
	return out, nil
}

func (s *sdk) ProjectResourceEnum(session unsafe.Pointer, project uniqueID) ([]uniqueID, error) {
	var e *C.HPMProjectResourceEnum
	if err := toError(C.project_resource_enum(&s.funcs, session, C.HPMUniqueID(project), &e)); err != nil {
		return nil, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(e), nil)

	out := make([]uniqueID, e.m_nResources)
	ptr := uintptr(unsafe.Pointer(e.m_pResources))
	for i := range out {
		out[i] = uniqueID(*(*C.HPMUniqueID)(unsafe.Pointer(ptr + uintptr(i*4))))
	}
	return out, nil
}

func (s *sdk) TaskEnum(session unsafe.Pointer, container uniqueID) ([]uniqueID, error) {
	var e *C.HPMTaskEnum
	if err := toError(C.task_enum(&s.funcs, session, C.HPMUniqueID(container), &e)); err != nil {
		return nil, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(e), nil)

	out := make([]uniqueID, e.m_nTasks)
	ptr := uintptr(unsafe.Pointer(e.m_pTasks))
	for i := range out {
		out[i] = *(*uniqueID)(unsafe.Pointer(ptr + uintptr(i*4)))
	}
	return out, nil
}

func (s *sdk) TaskRefEnum(session unsafe.Pointer, container uniqueID) ([]taskRef, error) {
	var e *C.HPMTaskEnum
	if err := toError(C.task_ref_enum(&s.funcs, session, C.HPMUniqueID(container), &e)); err != nil {
		return nil, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(e), nil)

	out := make([]taskRef, e.m_nTasks)
	ptr := uintptr(unsafe.Pointer(e.m_pTasks))
	for i := range out {
		out[i] = *(*taskRef)(unsafe.Pointer(ptr + uintptr(i*4)))
	}
	return out, nil
}

func (s *sdk) TaskGetDescription(session unsafe.Pointer, task uniqueID) (string, error) {
	var e *C.HPMString
	if err := toError(C.task_get_description(&s.funcs, session, C.HPMUniqueID(task), &e)); err != nil {
		return "", err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(e), nil)

	return C.GoString(e.m_pString), nil
}

func (s *sdk) TaskSetDescription(session unsafe.Pointer, task uniqueID, description string) error {
	str := C.CString(description)
	defer C.free(unsafe.Pointer(str))
	return toError(C.task_set_description(&s.funcs, session, C.HPMUniqueID(task), str))
}

func (s *sdk) TaskGetWorkflow(session unsafe.Pointer, task uniqueID) (int, error) {
	var id C.HPMUInt32
	if err := toError(C.task_get_workflow(&s.funcs, session, C.HPMUniqueID(task), &id)); err != nil {
		return 0, err
	}
	return int(id), nil
}

func (s *sdk) TaskGetWorkflowStatus(session unsafe.Pointer, task uniqueID) (int, error) {
	var status C.HPMInt32
	if err := toError(C.task_get_workflow_status(&s.funcs, session, C.HPMUniqueID(task), &status)); err != nil {
		return 0, err
	}
	return int(status), nil
}

func (s *sdk) TaskSetWorkflowStatus(session unsafe.Pointer, task uniqueID, status int) error {
	return toError(C.task_set_workflow_status(&s.funcs, session, C.HPMUniqueID(task), C.HPMInt32(status), C.EHPMTaskSetStatusFlag_DoAutoAssignments|C.EHPMTaskSetStatusFlag_DoAutoCompletion))
}

func (s *sdk) TaskSetEstimatedIdealDays(session unsafe.Pointer, task uniqueID, days float64) error {
	return toError(C.task_set_estimated_ideal_days(&s.funcs, session, C.HPMUniqueID(task), C.HPMFP64(days)))
}

func (s *sdk) TaskGetEstimatedIdealDays(session unsafe.Pointer, task uniqueID) (float64, error) {
	days := C.HPMFP64(0)
	if err := toError(C.task_get_estimated_ideal_days(&s.funcs, session, C.HPMUniqueID(task), &days)); err != nil {
		return 0, err
	}
	return float64(days), nil
}

type allocation struct {
	resource uniqueID
	percent  int
}

func (s *sdk) TaskGetResourceAllocation(session unsafe.Pointer, task uniqueID) ([]allocation, error) {
	var a *C.HPMTaskResourceAllocation
	if err := toError(C.task_get_resource_allocation(&s.funcs, session, C.HPMUniqueID(task), &a)); err != nil {
		return nil, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(a), nil)

	out := make([]allocation, a.m_nResources)
	ptr := uintptr(unsafe.Pointer(a.m_pResources))
	for i := range out {
		alloc := (*C.HPMTaskResourceAllocationResource)(unsafe.Pointer(ptr))
		out[i] = allocation{
			resource: uniqueID(alloc.m_ResourceID),
			percent:  int(alloc.m_PercentAllocated),
		}
		ptr += unsafe.Sizeof(C.HPMTaskResourceAllocationResource{})
	}
	return out, nil
}

func (s *sdk) TaskSetResourceAllocation(session unsafe.Pointer, task uniqueID, allocations []allocation) error {
	n := len(allocations)
	allocs := (*C.HPMTaskResourceAllocationResource)(C.malloc((C.ulong)(n) * (C.ulong)(unsafe.Sizeof(C.HPMTaskResourceAllocationResource{}))))
	ptr := uintptr(unsafe.Pointer(allocs))
	for i := 0; i < n; i++ {
		alloc := (*C.HPMTaskResourceAllocationResource)(unsafe.Pointer(ptr))
		alloc.m_ResourceID = C.HPMUniqueID(allocations[i].resource)
		alloc.m_PercentAllocated = C.HPMInt32(allocations[i].percent)
		ptr += unsafe.Sizeof(C.HPMTaskResourceAllocationResource{})
	}
	defer C.free(unsafe.Pointer(allocs))

	a := C.HPMTaskResourceAllocation{
		m_nResources: C.HPMUInt32(n),
		m_pResources: allocs,
	}
	return toError(C.task_set_resource_allocation(&s.funcs, session, C.HPMUniqueID(task), &a, C.HPMInt32(0), C.HPMInt32(0)))
}

func (s *sdk) TaskGetHyperlink(session unsafe.Pointer, task uniqueID) (string, error) {
	var link *C.HPMString
	if err := toError(C.task_get_hyperlink(&s.funcs, session, C.HPMUniqueID(task), &link)); err != nil {
		return "", err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(link), nil)
	return C.GoString(link.m_pString), nil
}

func (s *sdk) TaskSetHyperlink(session unsafe.Pointer, task uniqueID, hyperlink string) error {
	str := C.CString(hyperlink)
	defer C.free(unsafe.Pointer(str))
	return toError(C.task_set_hyperlink(&s.funcs, session, C.HPMUniqueID(task), (*C.HPMChar)(str)))
}

func (s *sdk) UtilGetNoMilestoneID(session unsafe.Pointer) (taskRef, error) {
	var id C.HPMInt32
	if err := toError(C.util_get_no_milestone_id(&s.funcs, session, &id)); err != nil {
		return 0, err
	}
	return taskRef(id), nil
}

func (s *sdk) TaskGetLinkedToMilestones(session unsafe.Pointer, task uniqueID) ([]taskRef, error) {
	var l *C.HPMTaskLinkedToMilestones
	if err := toError(C.task_get_linked_to_milestones(&s.funcs, session, C.HPMUniqueID(task), &l)); err != nil {
		return nil, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(l), nil)

	out := make([]taskRef, l.m_nMilestones)
	ptr := uintptr(unsafe.Pointer(l.m_pMilestones))
	for i := range out {
		out[i] = taskRef(*(*C.HPMUInt32)(unsafe.Pointer(ptr + uintptr(i*4))))
	}
	return out, nil
}

func (s *sdk) TaskSetLinkedToMilestones(session unsafe.Pointer, task uniqueID, milestones []taskRef) error {
	m := (*C.HPMUniqueID)(C.malloc(C.ulong(4 * len(milestones))))
	defer C.free(unsafe.Pointer(m))
	for i, milestone := range milestones {
		p := (*C.HPMUniqueID)(unsafe.Pointer(uintptr(unsafe.Pointer(m)) + uintptr(i*4)))
		*p = C.HPMUniqueID(milestone)
	}
	l := C.HPMTaskLinkedToMilestones{
		m_nMilestones: C.HPMUInt32(len(milestones)),
		m_pMilestones: m,
	}
	return toError(C.task_set_linked_to_milestones(&s.funcs, session, C.HPMUniqueID(task), &l))
}

func (s *sdk) TaskGetLinkedToSprint(session unsafe.Pointer, task uniqueID) (taskRef, error) {
	var id C.HPMUniqueID
	if err := toError(C.task_get_linked_to_sprint(&s.funcs, session, C.HPMUniqueID(task), &id)); err != nil {
		return 0, err
	}
	return taskRef(id), nil
}

func (s *sdk) TaskSetLinkedToSprint(session unsafe.Pointer, project uniqueID, task, sprint taskRef) error {
	// https://stackoverflow.com/a/33297896
	parent := (*C.HPMTaskCreateUnifiedReference)(C.malloc(C.ulong(unsafe.Sizeof(C.HPMTaskCreateUnifiedReference{}))))
	defer C.free(unsafe.Pointer(parent))
	*parent = C.HPMTaskCreateUnifiedReference{
		m_RefID: C.HPMUniqueID(sprint),
	}

	entry := (*C.HPMTaskCreateUnifiedEntry)(C.malloc(C.ulong(unsafe.Sizeof(C.HPMTaskCreateUnifiedEntry{}))))
	defer C.free(unsafe.Pointer(entry))
	*entry = C.HPMTaskCreateUnifiedEntry{
		m_bIsProxy:       1,
		m_TaskType:       C.EHPMTaskType_Planned,
		m_TaskLockedType: C.EHPMTaskLockedType_BacklogItem,
		m_nParentRefIDs:  1,
		m_pParentRefIDs:  parent,
		m_PreviousRefID: C.HPMTaskCreateUnifiedReference{
			m_RefID: C.HPMUniqueID(sprint),
		},
		m_Proxy_ReferToRefTaskID: C.HPMUniqueID(task),
	}

	data := C.HPMTaskCreateUnified{
		m_nTasks:      1,
		m_pTasks:      entry,
		m_OptionFlags: C.EHPMTaskCreateOptionFlag_UpdateCustomDateColumns | C.EHPMTaskCreateOptionFlag_SetDefaultValues,
	}
	var result *C.HPMChangeCallbackData_TaskCreateUnified
	if err := toError(C.task_create_unified(&s.funcs, session, C.HPMUniqueID(project), &data, &result)); err != nil {
		return err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(result), nil)

	return nil
}

func (s *sdk) TaskGetBacklogPriority(session unsafe.Pointer, task uniqueID) (Priority, error) {
	var priority C.HPMInt32
	if err := toError(C.task_get_backlog_priority(&s.funcs, session, C.HPMUniqueID(task), &priority)); err != nil {
		return PriorityMedium, err
	}
	return Priority(priority), nil
}

func (s *sdk) TaskSetBacklogPriority(session unsafe.Pointer, task uniqueID, priority Priority) error {
	return toError(C.task_set_backlog_priority(&s.funcs, session, C.HPMUniqueID(task), (C.HPMInt32)(priority)))
}

func (s *sdk) TaskGetMainReference(session unsafe.Pointer, task uniqueID) (taskRef, error) {
	var ref C.HPMUniqueID
	if err := toError(C.task_get_main_reference(&s.funcs, session, C.HPMUniqueID(task), &ref)); err != nil {
		return 0, err
	}
	return taskRef(ref), nil
}

type unifiedTaskType = int

const (
	// A regular task
	unifiedTaskPlanned unifiedTaskType = iota
	// A milestone
	unifiedTaskMilestone
)

func (s *sdk) TaskCreateUnified(session unsafe.Pointer, container uniqueID, ty unifiedTaskType) (taskRef, error) {
	task := (*C.HPMTaskCreateUnifiedEntry)(C.malloc(C.ulong(unsafe.Sizeof(C.HPMTaskCreateUnifiedEntry{}))))
	defer C.free(unsafe.Pointer(task))

	*task = C.HPMTaskCreateUnifiedEntry{}
	switch ty {
	case unifiedTaskPlanned:
		task.m_TaskType = C.EHPMTaskType_Planned
	case unifiedTaskMilestone:
		task.m_TaskType = C.EHPMTaskType_Milestone
	}

	data := C.HPMTaskCreateUnified{
		m_nTasks:      1,
		m_pTasks:      task,
		m_OptionFlags: C.EHPMTaskCreateOptionFlag_UpdateCustomDateColumns | C.EHPMTaskCreateOptionFlag_SetDefaultValues,
	}
	var result *C.HPMChangeCallbackData_TaskCreateUnified
	if err := toError(C.task_create_unified(&s.funcs, session, C.HPMUniqueID(container), &data, &result)); err != nil {
		return 0, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(result), nil)

	return taskRef(result.m_pTasks.m_TaskRefID), nil
}

func (s *sdk) TaskRefGetTask(session unsafe.Pointer, ref taskRef) (uniqueID, error) {
	var realID C.HPMUniqueID
	if err := toError(C.task_ref_get_task(&s.funcs, session, C.HPMUniqueID(ref), &realID)); err != nil {
		return 0, err
	}
	return uniqueID(realID), nil
}

func (s *sdk) TaskRefGetContainer(session unsafe.Pointer, ref taskRef) (uniqueID, error) {
	var realID C.HPMUniqueID
	if err := toError(C.task_ref_get_container(&s.funcs, session, C.HPMUniqueID(ref), &realID)); err != nil {
		return 0, err
	}
	return uniqueID(realID), nil
}

func (s *sdk) ResourceGetProperties(session unsafe.Pointer, id uniqueID) (resource, error) {
	var e *C.HPMResourceProperties
	if err := toError(C.resource_get_properties(&s.funcs, session, C.HPMUniqueID(id), &e)); err != nil {
		return resource{}, err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(e), nil)

	return resource{
		id:    id,
		name:  C.GoString(e.m_pName),
		email: C.GoString(e.m_pEmailAddress),
	}, nil
}

func (s *sdk) translateString(session unsafe.Pointer, untranslated *C.HPMUntranslatedString) (string, error) {
	language := C.HPMLanguage{
		m_LanguageID: 0x0809, // English - United Kingdom
	}

	var translated *C.HPMString
	if err := toError(C.localization_translate_string(&s.funcs, session, &language, untranslated, &translated)); err != nil {
		return "", err
	}
	defer C.object_free(&s.funcs, session, unsafe.Pointer(translated), nil)

	return C.GoString(translated.m_pString), nil
}

func toError(e C.HPMError) error {
	if e == C.EHPMError_NoError {
		return nil
	}
	msg := C.GoString(C.HPMErrorToStr(e))
	return fmt.Errorf("%v", msg)
}
