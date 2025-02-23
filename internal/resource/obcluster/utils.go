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

package obcluster

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/oceanbase/ob-operator/api/v1alpha1"
	oceanbaseconst "github.com/oceanbase/ob-operator/internal/const/oceanbase"
	resourceutils "github.com/oceanbase/ob-operator/internal/resource/utils"
	"github.com/oceanbase/ob-operator/pkg/oceanbase-sdk/operation"
	tasktypes "github.com/oceanbase/ob-operator/pkg/task/types"
)

func (m *OBClusterManager) checkIfStorageSizeExpand(obzone *v1alpha1.OBZone) bool {
	return obzone.Spec.OBServerTemplate.Storage.DataStorage.Size.Cmp(m.OBCluster.Spec.OBServerTemplate.Storage.DataStorage.Size) < 0 ||
		obzone.Spec.OBServerTemplate.Storage.LogStorage.Size.Cmp(m.OBCluster.Spec.OBServerTemplate.Storage.LogStorage.Size) < 0 ||
		obzone.Spec.OBServerTemplate.Storage.RedoLogStorage.Size.Cmp(m.OBCluster.Spec.OBServerTemplate.Storage.RedoLogStorage.Size) < 0
}

func (m *OBClusterManager) checkIfCalcResourceChange(obzone *v1alpha1.OBZone) bool {
	return obzone.Spec.OBServerTemplate.Resource.Cpu.Cmp(m.OBCluster.Spec.OBServerTemplate.Resource.Cpu) != 0 ||
		obzone.Spec.OBServerTemplate.Resource.Memory.Cmp(m.OBCluster.Spec.OBServerTemplate.Resource.Memory) != 0
}

func (m *OBClusterManager) checkIfBackupVolumeAdded(obzone *v1alpha1.OBZone) bool {
	return obzone.Spec.BackupVolume == nil && m.OBCluster.Spec.BackupVolume != nil
}

func (m *OBClusterManager) retryUpdateStatus() error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		obcluster, err := m.getOBCluster()
		if err != nil {
			return client.IgnoreNotFound(err)
		}
		obcluster.Status = *m.OBCluster.Status.DeepCopy()
		return m.Client.Status().Update(m.Ctx, obcluster)
	})
}

func (m *OBClusterManager) listOBZones() (*v1alpha1.OBZoneList, error) {
	// this label always exists
	obzoneList := &v1alpha1.OBZoneList{}
	err := m.Client.List(m.Ctx, obzoneList, client.MatchingLabels{
		oceanbaseconst.LabelRefOBCluster: m.OBCluster.Name,
	}, client.InNamespace(m.OBCluster.Namespace))
	if err != nil {
		return nil, errors.Wrap(err, "get obzone list")
	}
	return obzoneList, nil
}
func (m *OBClusterManager) listOBParameters() (*v1alpha1.OBParameterList, error) {
	// this label always exists
	obparameterList := &v1alpha1.OBParameterList{}
	err := m.Client.List(m.Ctx, obparameterList, client.MatchingLabels{
		oceanbaseconst.LabelRefOBCluster: m.OBCluster.Name,
	}, client.InNamespace(m.OBCluster.Namespace))
	if err != nil {
		return nil, errors.Wrap(err, "get obzone list")
	}
	return obparameterList, nil
}

func (m *OBClusterManager) getOBCluster() (*v1alpha1.OBCluster, error) {
	obcluster := &v1alpha1.OBCluster{}
	err := m.Client.Get(m.Ctx, types.NamespacedName{
		Namespace: m.OBCluster.Namespace,
		Name:      m.OBCluster.Name,
	}, obcluster)
	if err != nil {
		return nil, errors.Wrap(err, "get obcluster")
	}
	return obcluster, nil
}

func (m *OBClusterManager) generateZoneName(zone string) string {
	return fmt.Sprintf("%s-%d-%s", m.OBCluster.Spec.ClusterName, m.OBCluster.Spec.ClusterId, zone)
}

func (m *OBClusterManager) generateParameterName(name string) string {
	return fmt.Sprintf("%s-%d-%s", m.OBCluster.Spec.ClusterName, m.OBCluster.Spec.ClusterId, strings.ReplaceAll(name, "_", "-"))
}

func (m *OBClusterManager) getZonesToDelete() ([]v1alpha1.OBZone, error) {
	deletedZones := make([]v1alpha1.OBZone, 0)
	obzoneList, err := m.listOBZones()
	if err != nil {
		m.Logger.Error(err, "List obzone failed")
		return deletedZones, errors.Wrapf(err, "List obzone of obcluster %s failed", m.OBCluster.Name)
	}
	for _, obzone := range obzoneList.Items {
		reserve := false
		for _, zone := range m.OBCluster.Spec.Topology {
			if zone.Zone == obzone.Spec.Topology.Zone {
				reserve = true
				break
			}
		}
		if !reserve {
			m.Logger.V(oceanbaseconst.LogLevelTrace).Info("Need to delete obzone", "obzone", obzone.Name)
			deletedZones = append(deletedZones, obzone)
		}
	}
	return deletedZones, nil
}

func (m *OBClusterManager) getOceanbaseOperationManager() (*operation.OceanbaseOperationManager, error) {
	return resourceutils.GetSysOperationClient(m.Client, m.Logger, m.OBCluster)
}

func (m *OBClusterManager) createUser(userName, secretName, privilege string) error {
	m.Logger.V(oceanbaseconst.LogLevelDebug).Info("begin create user", "username", userName)
	password, err := resourceutils.ReadPassword(m.Client, m.OBCluster.Namespace, secretName)
	if err != nil {
		return errors.Wrapf(err, "Get password from secret %s failed", secretName)
	}
	m.Logger.V(oceanbaseconst.LogLevelDebug).Info("finish get password", "username", userName, "password", password)
	oceanbaseOperationManager, err := m.getOceanbaseOperationManager()
	if err != nil {
		m.Logger.Error(err, "Get oceanbase operation manager")
		return errors.Wrap(err, "Get oceanbase operation manager")
	}
	m.Logger.V(oceanbaseconst.LogLevelDebug).Info("finish get operationmanager", "username", userName)
	err = oceanbaseOperationManager.CreateUser(userName)
	if err != nil {
		m.Logger.Error(err, "Create user")
		return errors.Wrapf(err, "Create user %s", userName)
	}
	m.Logger.V(oceanbaseconst.LogLevelDebug).Info("finish create user", "username", userName)
	err = oceanbaseOperationManager.SetUserPassword(userName, password)
	if err != nil {
		m.Logger.Error(err, "Set user password")
		return errors.Wrapf(err, "Set password for user %s", userName)
	}
	m.Logger.V(oceanbaseconst.LogLevelDebug).Info("finish set user password", "username", userName)
	object := "*.*"
	err = oceanbaseOperationManager.GrantPrivilege(privilege, object, userName)
	if err != nil {
		m.Logger.Error(err, "Grant privilege")
		return errors.Wrapf(err, "Grant privilege for user %s", userName)
	}
	m.Logger.V(oceanbaseconst.LogLevelDebug).Info("finish grant user privilege", "username", userName)
	return nil
}

type obzoneChanger func(*v1alpha1.OBZone)

func (m *OBClusterManager) changeZonesWhenScaling(obzone *v1alpha1.OBZone) {
	obzone.Spec.OBServerTemplate.Resource.Cpu = m.OBCluster.Spec.OBServerTemplate.Resource.Cpu
	obzone.Spec.OBServerTemplate.Resource.Memory = m.OBCluster.Spec.OBServerTemplate.Resource.Memory
}

func (m *OBClusterManager) changeZonesWhenExpandingPVC(obzone *v1alpha1.OBZone) {
	obzone.Spec.OBServerTemplate.Storage.DataStorage.Size = m.OBCluster.Spec.OBServerTemplate.Storage.DataStorage.Size
	obzone.Spec.OBServerTemplate.Storage.LogStorage.Size = m.OBCluster.Spec.OBServerTemplate.Storage.LogStorage.Size
	obzone.Spec.OBServerTemplate.Storage.RedoLogStorage.Size = m.OBCluster.Spec.OBServerTemplate.Storage.RedoLogStorage.Size
}

func (m *OBClusterManager) changeZonesWhenMountingBackupVolume(obzone *v1alpha1.OBZone) {
	obzone.Spec.BackupVolume = m.OBCluster.Spec.BackupVolume
}

func (m *OBClusterManager) modifyOBZonesAndCheckStatus(changer obzoneChanger, status string, timeoutSeconds int) tasktypes.TaskFunc {
	return func() tasktypes.TaskError {
		obzoneList, err := m.listOBZones()
		if err != nil {
			return errors.Wrap(err, "list obzones")
		}
		for _, obzone := range obzoneList.Items {
			changer(&obzone)
			err = m.Client.Update(m.Ctx, &obzone)
			if err != nil {
				return errors.Wrap(err, "update obzone")
			}
		}

		// check status of obzones
		matched := true
	outer:
		for i := 0; i < timeoutSeconds; i++ {
			time.Sleep(time.Second)
			obzoneList, err = m.listOBZones()
			if err != nil {
				return errors.Wrap(err, "list obzones")
			}
			for _, obzone := range obzoneList.Items {
				if obzone.Status.Status != status {
					matched = false
					continue outer
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			return errors.New("failed to wait for status of obzone to be " + status)
		}
		return nil
	}
}

func (m *OBClusterManager) rollingUpdateZones(changer obzoneChanger, workingStatus, targetStatus string, timeoutSeconds int) tasktypes.TaskFunc {
	return func() tasktypes.TaskError {
		tk := time.NewTicker(time.Duration(timeoutSeconds*2) * time.Second)
		defer tk.Stop()
		obzoneList, err := m.listOBZones()
		if err != nil {
			return errors.Wrap(err, "list obzones")
		}
		for _, obzone := range obzoneList.Items {
			m.Recorder.Event(m.OBCluster, "Normal", "RollingUpdateOBZone", "Rolling update OBZone "+obzone.Name)
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				changer(&obzone)
				return m.Client.Update(m.Ctx, &obzone)
			})
			if err != nil {
				return errors.Wrap(err, "update obzone")
			}
			for i := 0; i < timeoutSeconds; i++ {
				select {
				case <-tk.C:
					return errors.New("task timeout")
				default:
				}
				time.Sleep(time.Second)
				updatedOBZone := &v1alpha1.OBZone{}
				err := m.Client.Get(m.Ctx, types.NamespacedName{
					Namespace: obzone.Namespace,
					Name:      obzone.Name,
				}, updatedOBZone)
				if err != nil {
					return errors.Wrap(err, "get obzone")
				}
				if updatedOBZone.Status.Status == workingStatus {
					break
				}
			}
			for i := 0; i < timeoutSeconds; i++ {
				select {
				case <-tk.C:
					return errors.New("task timeout")
				default:
				}
				time.Sleep(time.Second)
				updatedOBZone := &v1alpha1.OBZone{}
				err := m.Client.Get(m.Ctx, types.NamespacedName{
					Namespace: obzone.Namespace,
					Name:      obzone.Name,
				}, updatedOBZone)
				if err != nil {
					return errors.Wrap(err, "get obzone")
				}
				if updatedOBZone.Status.Status == targetStatus {
					break
				}
			}
		}
		return nil
	}
}
