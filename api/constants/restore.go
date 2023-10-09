/*
Copyright (c) 2023 OceanBase
ob-operator is licensed under Mulan PSL v2.
You can use this software according to the terms and conditions of the Mulan PSL v2.
You may obtain a copy of Mulan PSL v2 at:
         http://license.coscl.org.cn/MulanPSL2
THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
See the Mulan PSL v2 for more details.
*/

package constants

import "github.com/oceanbase/ob-operator/api/types"

const (
	RestoreJobStarting   types.RestoreJobStatus = "STARTING"
	RestoreJobRunning    types.RestoreJobStatus = "RUNNING"
	RestoreJobFailed     types.RestoreJobStatus = "FAILED"
	RestoreJobSuccessful types.RestoreJobStatus = "SUCCESSFUL"
	RestoreJobCanceled   types.RestoreJobStatus = "CANCELED"

	RestoreJobStatusActivating types.RestoreJobStatus = "ACTIVATING"
	RestoreJobStatusReplaying  types.RestoreJobStatus = "REPLAYING"
)
