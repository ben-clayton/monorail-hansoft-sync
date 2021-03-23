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

#include "../../third_party/hansoft_sdk/HPMSdk.c"

extern void onProcessCallback(void *);

void *session_open(
    HPMSdkFunctions *funcs,
    HPMError *_pError,
    const HPMChar *_pAddress,
    HPMInt32 _Port,
    const HPMChar *_pDatabase,
    const HPMChar *_pResourceName,
    const HPMChar *_pPassword,
    HPMInt32 _bBlockOnOperations,
    HPMNeedSessionProcessCallbackInfo const *_pNeedProcessCallback,
    HPMUInt32 _SDKVersion,
    HPMInt32 _SDKDebug,
    HPMUInt32 _nSessions,
    const HPMChar *_pWorkingDirectory,
    const HPMCertificateSettings *_pCertificateSettings,
    const HPMChar **_pExtendedErrorMessage)
{
    return funcs->SessionOpen(
        _pError,
        _pAddress,
        _Port,
        _pDatabase,
        _pResourceName,
        _pPassword,
        _bBlockOnOperations,
        _pNeedProcessCallback,
        _SDKVersion,
        _SDKDebug,
        _nSessions,
        _pWorkingDirectory,
        _pCertificateSettings,
        _pExtendedErrorMessage);
}

HPMError session_stop(
    HPMSdkFunctions *funcs,
    void *_pSession)
{
    return funcs->SessionStop(_pSession);
}

HPMError session_close(
    HPMSdkFunctions *funcs,
    void *_pSession)
{
    return funcs->SessionClose(_pSession);
}

HPMError session_process(
    HPMSdkFunctions *funcs,
    void *_pSession)
{
    return funcs->SessionProcess(_pSession);
}

HPMError project_enum(
    HPMSdkFunctions *funcs,
    void *_pSession,
    const HPMProjectEnum **_pEnum)
{
    return funcs->ProjectEnum(_pSession, _pEnum);
}

HPMError project_util_get_backlog(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ProjectID,
    HPMUniqueID *_pReturn)
{
    return funcs->ProjectUtilGetBacklog(_pSession, _ProjectID, _pReturn);
}

HPMError project_get_milestones(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ProjectID,
    const HPMProjectMilestones **_pData)
{
    return funcs->ProjectGetMilestones(_pSession, _ProjectID, _pData);
}

HPMError project_get_sprints(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ProjectID,
    const HPMProjectSprints **_pData)
{
    return funcs->ProjectGetSprints(_pSession, _ProjectID, _pData);
}

HPMError project_get_properties(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ProjectID,
    const HPMProjectProperties **_pProperties)
{
    return funcs->ProjectGetProperties(_pSession, _ProjectID, _pProperties);
}

HPMError project_workflow_enum(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ProjectID,
    HPMUInt32 _bOnlyNewestVersions,
    const HPMProjectWorkflowEnum **_pEnum

)
{
    return funcs->ProjectWorkflowEnum(_pSession, _ProjectID, _bOnlyNewestVersions, _pEnum);
}

HPMError project_workflow_get_settings(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ProjectID,
    HPMUInt32 _WorkflowID,
    const HPMProjectWorkflowSettings **_pSettings)
{
    return funcs->ProjectWorkflowGetSettings(_pSession, _ProjectID, _WorkflowID, _pSettings);
}

HPMError project_custom_columns_get(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ProjectID,
    const HPMProjectCustomColumns **_pColumns)
{
    return funcs->ProjectCustomColumnsGet(_pSession, _ProjectID, _pColumns);
}

HPMError project_resource_enum(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ProjectID,
    const HPMProjectResourceEnum **_pEnum)
{
    return funcs->ProjectResourceEnum(_pSession, _ProjectID, _pEnum);
}

HPMError task_enum(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ContainerID,
    const HPMTaskEnum **_pEnum)
{
    return funcs->TaskEnum(_pSession, _ContainerID, _pEnum);
}

HPMError task_ref_enum(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ContainerID,
    const HPMTaskEnum **_pEnum)
{
    return funcs->TaskRefEnum(_pSession, _ContainerID, _pEnum);
}

HPMError task_get_description(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    const HPMString **_pData)
{
    return funcs->TaskGetDescription(_pSession, _TaskID, _pData);
}

HPMError task_set_description(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    const HPMChar *_pData)
{
    return funcs->TaskSetDescription(_pSession, _TaskID, _pData);
}

HPMError task_get_linked_to_milestones(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    const HPMTaskLinkedToMilestones **_pData)
{
    return funcs->TaskGetLinkedToMilestones(_pSession, _TaskID, _pData);
}

HPMError task_set_linked_to_milestones(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    const HPMTaskLinkedToMilestones *_pData)
{
    return funcs->TaskSetLinkedToMilestones(_pSession, _TaskID, _pData);
}

HPMError task_get_linked_to_sprint(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    HPMUniqueID *_pData)
{
    return funcs->TaskGetLinkedToSprint(_pSession, _TaskID, _pData);
}

HPMError task_get_status(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    HPMInt32 *_pData)
{
    return funcs->TaskGetStatus(_pSession, _TaskID, _pData);
}

HPMError task_set_status(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    HPMInt32 _Data,
    HPMInt32 _bGotoWorkflowStatus,
    HPMInt32 _SetStatusFlags)
{
    return funcs->TaskSetStatus(_pSession, _TaskID, _Data, _bGotoWorkflowStatus, _SetStatusFlags);
}

HPMError task_get_workflow(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    HPMUInt32 *_pData)
{
    return funcs->TaskGetWorkflow(_pSession, _TaskID, _pData);
}

HPMError task_get_workflow_status(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    HPMInt32 *_pData)
{
    return funcs->TaskGetWorkflowStatus(_pSession, _TaskID, _pData);
}

HPMError task_set_workflow_status(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    HPMInt32 _Data,
    HPMInt32 _Flags)
{
    return funcs->TaskSetWorkflowStatus(_pSession, _TaskID, _Data, _Flags);
}

HPMError task_create_unified(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ContainerID,
    const HPMTaskCreateUnified *_pCreateData,
    const HPMChangeCallbackData_TaskCreateUnified **_pReturn)
{
    return funcs->TaskCreateUnified(_pSession, _ContainerID, _pCreateData, _pReturn);
}

HPMError task_get_estimated_ideal_days(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    HPMFP64 *_pData)
{
    return funcs->TaskGetEstimatedIdealDays(_pSession, _TaskID, _pData);
}

HPMError task_set_estimated_ideal_days(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    HPMFP64 _pData)
{
    return funcs->TaskSetEstimatedIdealDays(_pSession, _TaskID, _pData);
}

HPMError task_get_resource_allocation(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    const HPMTaskResourceAllocation **_pData)
{
    return funcs->TaskGetResourceAllocation(_pSession, _TaskID, _pData);
}

HPMError task_set_resource_allocation(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    const HPMTaskResourceAllocation *_pData,
    HPMInt32 _bGotoWorkflowStatusWhenAssigned,
    HPMInt32 _SetStatusFlags)
{
    return funcs->TaskSetResourceAllocation(_pSession, _TaskID, _pData, _bGotoWorkflowStatusWhenAssigned, _SetStatusFlags);
}

HPMError task_get_hyperlink(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    const HPMString **_pData)
{
    return funcs->TaskGetHyperlink(_pSession, _TaskID, _pData);
}

HPMError task_set_hyperlink(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    const HPMChar *_pData)
{
    return funcs->TaskSetHyperlink(_pSession, _TaskID, _pData);
}

HPMError task_get_backlog_priority(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    HPMInt32 *_pData)
{
    return funcs->TaskGetBacklogPriority(_pSession, _TaskID, _pData);
}

HPMError task_set_backlog_priority(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    HPMInt32 _Data)
{
    return funcs->TaskSetBacklogPriority(_pSession, _TaskID, _Data);
}

HPMError task_get_main_reference(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskID,
    HPMUniqueID *_pMainRefID)
{
    return funcs->TaskGetMainReference(_pSession, _TaskID, _pMainRefID);
}

HPMError task_ref_get_task(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskRefID,
    HPMUniqueID *_pTaskID)
{
    return funcs->TaskRefGetTask(_pSession, _TaskRefID, _pTaskID);
}

HPMError task_ref_get_container(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _TaskRefID,
    HPMUniqueID *_pTaskID)
{
    return funcs->TaskRefGetContainer(_pSession, _TaskRefID, _pTaskID);
}

HPMError resource_get_properties(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMUniqueID _ResourceID,
    const HPMResourceProperties **_pResourceProperties)
{
    return funcs->ResourceGetProperties(_pSession, _ResourceID, _pResourceProperties);
}

HPMError util_get_no_milestone_id(
    HPMSdkFunctions *funcs,
    void *_pSession,
    HPMInt32 *_pID)
{
    return funcs->UtilGetNoMilestoneID(_pSession, _pID);
}

HPMError localization_translate_string(
    HPMSdkFunctions *funcs,
    void *_pSession,
    const HPMLanguage *_pLanguage,
    const HPMUntranslatedString *_pUntranslatedString,
    const HPMString **_pTranslatedString)
{
    return funcs->LocalizationTranslateString(_pSession, _pLanguage, _pUntranslatedString, _pTranslatedString);
}

HPMError object_free(
    HPMSdkFunctions *funcs,
    void *_pSession,
    const void *_pObject,
    HPMInt32 *_pDeleted)
{
    return funcs->ObjectFree(_pSession, _pObject, _pDeleted);
}
