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

package oceanbase

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	logger "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apitypes "github.com/oceanbase/ob-operator/api/types"
	"github.com/oceanbase/ob-operator/api/v1alpha1"
	oceanbaseconst "github.com/oceanbase/ob-operator/internal/const/oceanbase"
	clusterstatus "github.com/oceanbase/ob-operator/internal/const/status/obcluster"
	"github.com/oceanbase/ob-operator/internal/dashboard/business/common"
	"github.com/oceanbase/ob-operator/internal/dashboard/business/constant"
	modelcommon "github.com/oceanbase/ob-operator/internal/dashboard/model/common"
	"github.com/oceanbase/ob-operator/internal/dashboard/model/param"
	"github.com/oceanbase/ob-operator/internal/dashboard/model/response"
	"github.com/oceanbase/ob-operator/internal/oceanbase"
)

const (
	StatusDeleting  = "deleting"
	StatusRunning   = "running"
	StatusOperating = "operating"
	StatusFailed    = "failed"
)

func convertStatus(detailedStatus string) string {
	switch detailedStatus {
	case StatusRunning, StatusDeleting:
		return detailedStatus
	default:
		return StatusOperating
	}
}

func getStatisticStatus(obcluster *v1alpha1.OBCluster) string {
	if !obcluster.ObjectMeta.DeletionTimestamp.IsZero() {
		return StatusDeleting
	} else if obcluster.Status.Status == StatusRunning {
		return StatusRunning
	} else if obcluster.Status.Status == clusterstatus.Failed {
		return StatusFailed
	} else {
		return StatusOperating
	}
}

func buildOBClusterOverview(ctx context.Context, obcluster *v1alpha1.OBCluster) (*response.OBClusterOverview, error) {
	topology, err := buildOBClusterTopologyResp(ctx, obcluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build obcluster topology")
	}
	return &response.OBClusterOverview{
		Namespace:    obcluster.Namespace,
		Name:         obcluster.Name,
		ClusterName:  obcluster.Spec.ClusterName,
		ClusterId:    obcluster.Spec.ClusterId,
		Status:       getStatisticStatus(obcluster),
		StatusDetail: obcluster.Status.Status,
		CreateTime:   obcluster.ObjectMeta.CreationTimestamp.Unix(),
		Image:        obcluster.Status.Image,
		Topology:     topology,
	}, nil
}

func buildOBClusterResponse(ctx context.Context, obcluster *v1alpha1.OBCluster) (*response.OBCluster, error) {
	overview, err := buildOBClusterOverview(ctx, obcluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build obcluster overview")
	}
	respCluster := &response.OBCluster{
		OBClusterOverview: *overview,
		OBClusterExtra: response.OBClusterExtra{
			RootPasswordSecret: obcluster.Spec.UserSecrets.Root,
			Parameters:         nil,
		},
		// TODO: add metrics
		Metrics: nil,
	}
	var parameters []modelcommon.KVPair
	for _, param := range obcluster.Spec.Parameters {
		parameters = append(parameters, modelcommon.KVPair{
			Key:   param.Name,
			Value: param.Value,
		})
	}
	respCluster.Parameters = parameters

	if obcluster.Spec.MonitorTemplate != nil {
		respCluster.Monitor = &response.MonitorSpec{}
		respCluster.Monitor.Image = obcluster.Spec.MonitorTemplate.Image
		respCluster.Monitor.Resource = response.ResourceSpecRender{
			Cpu:      obcluster.Spec.MonitorTemplate.Resource.Cpu.Value(),
			MemoryGB: obcluster.Spec.MonitorTemplate.Resource.Memory.String(),
		}
	}
	if obcluster.Spec.BackupVolume != nil {
		respCluster.BackupVolume = &response.NFSVolumeSpec{}
		respCluster.BackupVolume.Address = obcluster.Spec.BackupVolume.Volume.NFS.Server
		respCluster.BackupVolume.Path = obcluster.Spec.BackupVolume.Volume.NFS.Path
	}
	labels := obcluster.GetLabels()
	if labels != nil {
		if mode, ok := labels[oceanbaseconst.AnnotationsMode]; ok {
			switch mode {
			case oceanbaseconst.ModeStandalone:
				respCluster.Mode = modelcommon.ClusterModeStandalone
			case oceanbaseconst.ModeService:
				respCluster.Mode = modelcommon.ClusterModeService
			default:
				respCluster.Mode = modelcommon.ClusterModeNormal
			}
		} else {
			respCluster.Mode = modelcommon.ClusterModeNormal
		}
	}
	if obcluster.Spec.OBServerTemplate != nil {
		respCluster.OBClusterExtra.Resource = response.ResourceSpecRender{
			Cpu:      obcluster.Spec.OBServerTemplate.Resource.Cpu.Value(),
			MemoryGB: obcluster.Spec.OBServerTemplate.Resource.Memory.String(),
		}
		respCluster.OBClusterExtra.Storage = response.OBServerStorage{
			DataStorage: response.StorageSpec{
				StorageClass: obcluster.Spec.OBServerTemplate.Storage.DataStorage.StorageClass,
				SizeGB:       obcluster.Spec.OBServerTemplate.Storage.DataStorage.Size.String(),
			},
			RedoLogStorage: response.StorageSpec{
				StorageClass: obcluster.Spec.OBServerTemplate.Storage.RedoLogStorage.StorageClass,
				SizeGB:       obcluster.Spec.OBServerTemplate.Storage.RedoLogStorage.Size.String(),
			},
			SysLogStorage: response.StorageSpec{
				StorageClass: obcluster.Spec.OBServerTemplate.Storage.LogStorage.StorageClass,
				SizeGB:       obcluster.Spec.OBServerTemplate.Storage.LogStorage.Size.String(),
			},
		}
	}

	return respCluster, nil
}

func buildOBClusterTopologyResp(ctx context.Context, obcluster *v1alpha1.OBCluster) ([]response.OBZone, error) {
	obzoneList, err := oceanbase.ListOBZonesOfOBCluster(ctx, obcluster)
	if err != nil {
		return nil, errors.Wrapf(err, "List obzone of obcluster %s %s", obcluster.Namespace, obcluster.Name)
	}
	topology := make([]response.OBZone, 0, len(obzoneList.Items))
	for _, obzone := range obzoneList.Items {
		observers := make([]response.OBServer, 0)
		observerList, err := oceanbase.ListOBServersOfOBZone(ctx, &obzone)
		if err != nil {
			return nil, errors.Wrapf(err, "List observers of obzone %s %s", obzone.Namespace, obzone.Name)
		}
		logger.Infof("found %d observer", len(observerList.Items))
		for _, observer := range observerList.Items {
			logger.Infof("add observer %s to result", observer.Name)
			observers = append(observers, response.OBServer{
				Namespace:    observer.Namespace,
				Name:         observer.Name,
				Status:       convertStatus(observer.Status.Status),
				StatusDetail: observer.Status.Status,
				Address:      observer.Status.PodIp,
				// TODO: add metrics
				Metrics: nil,
			})
		}

		nodeSelector := make([]modelcommon.KVPair, 0)
		for k, v := range obzone.Spec.Topology.NodeSelector {
			nodeSelector = append(nodeSelector, modelcommon.KVPair{
				Key:   k,
				Value: v,
			})
		}

		affinities := make([]modelcommon.AffinitySpec, 0)
		if obzone.Spec.Topology.Affinity != nil {
			zoneAffinity := obzone.Spec.Topology.Affinity
			switch {
			case zoneAffinity.NodeAffinity != nil:
				for _, term := range zoneAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
					for _, req := range term.MatchExpressions {
						affinities = append(affinities, modelcommon.AffinitySpec{
							Type: modelcommon.NodeAffinityType,
							KVPair: modelcommon.KVPair{
								Key:   req.Key,
								Value: req.Values[0],
							},
						})
					}
				}
			case zoneAffinity.PodAffinity != nil:
				for _, term := range zoneAffinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
					for _, req := range term.LabelSelector.MatchExpressions {
						affinities = append(affinities, modelcommon.AffinitySpec{
							Type: modelcommon.PodAffinityType,
							KVPair: modelcommon.KVPair{
								Key:   req.Key,
								Value: req.Values[0],
							},
						})
					}
				}
			case zoneAffinity.PodAntiAffinity != nil:
				for _, term := range zoneAffinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
					for _, req := range term.LabelSelector.MatchExpressions {
						affinities = append(affinities, modelcommon.AffinitySpec{
							Type: modelcommon.PodAntiAffinityType,
							KVPair: modelcommon.KVPair{
								Key:   req.Key,
								Value: req.Values[0],
							},
						})
					}
				}
			}
		}

		tolerations := make([]modelcommon.KVPair, 0)
		for _, toleration := range obzone.Spec.Topology.Tolerations {
			tolerations = append(tolerations, modelcommon.KVPair{
				Key:   toleration.Key,
				Value: toleration.Value,
			})
		}
		respZone := response.OBZone{
			Namespace:    obzone.Namespace,
			Name:         obzone.Name,
			Zone:         obzone.Spec.Topology.Zone,
			Replicas:     obzone.Spec.Topology.Replica,
			Status:       convertStatus(obzone.Status.Status),
			StatusDetail: obzone.Status.Status,
			RootService:  "",
			// TODO: query real rs
			OBServers:    observers,
			NodeSelector: nodeSelector,
			Affinities:   affinities,
			Tolerations:  tolerations,
		}
		if len(obzone.Status.OBServerStatus) > 0 {
			respZone.RootService = obzone.Status.OBServerStatus[0].Server
		}
		topology = append(topology, respZone)
	}

	return topology, nil
}

func ListOBClusters(ctx context.Context) ([]response.OBClusterOverview, error) {
	obclusters := make([]response.OBClusterOverview, 0)
	obclusterList, err := oceanbase.ListAllOBClusters(ctx)
	if err != nil {
		return obclusters, errors.Wrap(err, "failed to list obclusters")
	}
	for _, obcluster := range obclusterList.Items {
		resp, err := buildOBClusterOverview(ctx, &obcluster)
		if err != nil {
			logger.Errorf("failed to build obcluster response: %v", err)
		}
		obclusters = append(obclusters, *resp)
	}
	return obclusters, nil
}

func buildOBServerTemplate(observerSpec *param.OBServerSpec) *apitypes.OBServerTemplate {
	if observerSpec == nil {
		return nil
	}
	observerTemplate := &apitypes.OBServerTemplate{
		Image: observerSpec.Image,
		Resource: &apitypes.ResourceSpec{
			Cpu:    *apiresource.NewQuantity(observerSpec.Resource.Cpu, apiresource.DecimalSI),
			Memory: *apiresource.NewQuantity(observerSpec.Resource.MemoryGB*constant.GB, apiresource.BinarySI),
		},
		Storage: &apitypes.OceanbaseStorageSpec{
			DataStorage: &apitypes.StorageSpec{
				StorageClass: observerSpec.Storage.Data.StorageClass,
				Size:         *apiresource.NewQuantity(observerSpec.Storage.Data.SizeGB*constant.GB, apiresource.BinarySI),
			},
			RedoLogStorage: &apitypes.StorageSpec{
				StorageClass: observerSpec.Storage.RedoLog.StorageClass,
				Size:         *apiresource.NewQuantity(observerSpec.Storage.RedoLog.SizeGB*constant.GB, apiresource.BinarySI),
			},
			LogStorage: &apitypes.StorageSpec{
				StorageClass: observerSpec.Storage.Log.StorageClass,
				Size:         *apiresource.NewQuantity(observerSpec.Storage.Log.SizeGB*constant.GB, apiresource.BinarySI),
			},
		},
	}
	return observerTemplate
}

func buildMonitorTemplate(monitorSpec *param.MonitorSpec) *apitypes.MonitorTemplate {
	if monitorSpec == nil {
		return nil
	}
	monitorTemplate := &apitypes.MonitorTemplate{
		Image: monitorSpec.Image,
		Resource: &apitypes.ResourceSpec{
			Cpu:    *apiresource.NewQuantity(monitorSpec.Resource.Cpu, apiresource.DecimalSI),
			Memory: *apiresource.NewQuantity(monitorSpec.Resource.MemoryGB*constant.GB, apiresource.BinarySI),
		},
	}
	return monitorTemplate
}

func buildBackupVolume(nfsVolumeSpec *param.NFSVolumeSpec) *apitypes.BackupVolumeSpec {
	if nfsVolumeSpec == nil {
		return nil
	}
	backupVolume := &apitypes.BackupVolumeSpec{
		Volume: &corev1.Volume{
			Name: "ob-backup",
			VolumeSource: corev1.VolumeSource{
				NFS: &corev1.NFSVolumeSource{
					Server:   nfsVolumeSpec.Address,
					Path:     nfsVolumeSpec.Path,
					ReadOnly: false,
				},
			},
		},
	}
	return backupVolume
}

func buildOBClusterTopology(topology []param.ZoneTopology) []apitypes.OBZoneTopology {
	obzoneTopology := make([]apitypes.OBZoneTopology, 0)
	for _, zone := range topology {
		topo := apitypes.OBZoneTopology{
			Zone:         zone.Zone,
			NodeSelector: common.KVsToMap(zone.NodeSelector),
			Replica:      zone.Replicas,
		}
		if len(zone.Affinities) > 0 {
			topo.Affinity = &corev1.Affinity{}
			for _, kv := range zone.Affinities {
				switch kv.Type {
				case modelcommon.NodeAffinityType:
					if topo.Affinity.NodeAffinity == nil {
						topo.Affinity.NodeAffinity = &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{},
							},
						}
					}
					nodeSelectorTerm := corev1.NodeSelectorTerm{
						MatchExpressions: []corev1.NodeSelectorRequirement{{
							Key:      kv.Key,
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{kv.Value},
						}},
					}
					topo.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(topo.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, nodeSelectorTerm)
				case modelcommon.PodAffinityType:
					if topo.Affinity.PodAffinity == nil {
						topo.Affinity.PodAffinity = &corev1.PodAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{},
						}
					}
					podAffinityTerm := corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{{
								Key:      kv.Key,
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{kv.Value},
							}},
						},
					}
					topo.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(topo.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution, podAffinityTerm)
				case modelcommon.PodAntiAffinityType:
					if topo.Affinity.PodAntiAffinity == nil {
						topo.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{},
						}
					}
					podAntiAffinityTerm := corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{{
								Key:      kv.Key,
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{kv.Value},
							}},
						},
					}
					topo.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(topo.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, podAntiAffinityTerm)
				}
			}
		}
		if len(zone.Tolerations) > 0 {
			topo.Tolerations = make([]corev1.Toleration, 0)
			for _, kv := range zone.Tolerations {
				toleration := corev1.Toleration{
					Key:      kv.Key,
					Operator: corev1.TolerationOpEqual,
					Value:    kv.Value,
					Effect:   corev1.TaintEffectNoSchedule,
				}
				topo.Tolerations = append(topo.Tolerations, toleration)
			}
		}
		obzoneTopology = append(obzoneTopology, topo)
	}
	return obzoneTopology
}

func buildOBClusterParameters(parameters []modelcommon.KVPair) []apitypes.Parameter {
	obparameters := make([]apitypes.Parameter, 0)
	for _, parameter := range parameters {
		obparameters = append(obparameters, apitypes.Parameter{
			Name:  parameter.Key,
			Value: parameter.Value,
		})
	}
	return obparameters
}

func generateUUID() string {
	parts := strings.Split(uuid.New().String(), "-")
	return parts[len(parts)-1]
}

func generateUserSecrets(clusterName string, clusterId int64) *apitypes.OBUserSecrets {
	return &apitypes.OBUserSecrets{
		Root:     fmt.Sprintf("%s-%d-root-%s", clusterName, clusterId, generateUUID()),
		ProxyRO:  fmt.Sprintf("%s-%d-proxyro-%s", clusterName, clusterId, generateUUID()),
		Monitor:  fmt.Sprintf("%s-%d-monitor-%s", clusterName, clusterId, generateUUID()),
		Operator: fmt.Sprintf("%s-%d-operator-%s", clusterName, clusterId, generateUUID()),
	}
}

func generateOBClusterInstance(param *param.CreateOBClusterParam) *v1alpha1.OBCluster {
	observerTemplate := buildOBServerTemplate(param.OBServer)
	monitorTemplate := buildMonitorTemplate(param.Monitor)
	backupVolume := buildBackupVolume(param.BackupVolume)
	parameters := buildOBClusterParameters(param.Parameters)
	topology := buildOBClusterTopology(param.Topology)
	obcluster := &v1alpha1.OBCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: param.Namespace,
			Name:      param.Name,
			Labels:    map[string]string{},
		},
		Spec: v1alpha1.OBClusterSpec{
			ClusterName:      param.ClusterName,
			ClusterId:        param.ClusterId,
			OBServerTemplate: observerTemplate,
			MonitorTemplate:  monitorTemplate,
			BackupVolume:     backupVolume,
			Parameters:       parameters,
			Topology:         topology,
			UserSecrets:      generateUserSecrets(param.Name, param.ClusterId),
		},
	}
	switch param.Mode {
	case modelcommon.ClusterModeStandalone:
		obcluster.Labels[oceanbaseconst.AnnotationsMode] = oceanbaseconst.ModeStandalone
	case modelcommon.ClusterModeService:
		obcluster.Labels[oceanbaseconst.AnnotationsMode] = oceanbaseconst.ModeService
	default:
	}
	return obcluster
}

func CreateOBCluster(ctx context.Context, param *param.CreateOBClusterParam) error {
	obcluster := generateOBClusterInstance(param)
	err := oceanbase.CreateSecretsForOBCluster(ctx, obcluster, param.RootPassword)
	if err != nil {
		return errors.Wrap(err, "Create secrets for obcluster")
	}
	logger.Infof("Generated obcluster instance:%v", obcluster)
	return oceanbase.CreateOBCluster(ctx, obcluster)
}

func UpgradeObCluster(ctx context.Context, obclusterIdentity *param.K8sObjectIdentity, updateParam *param.UpgradeOBClusterParam) error {
	obcluster, err := oceanbase.GetOBCluster(ctx, obclusterIdentity.Namespace, obclusterIdentity.Name)
	if err != nil {
		return errors.Wrapf(err, "Get obcluster %s %s", obclusterIdentity.Namespace, obclusterIdentity.Name)
	}
	if obcluster.Status.Status != clusterstatus.Running {
		return errors.Errorf("Obcluster status invalid %s", obcluster.Status.Status)
	}
	obcluster.Spec.OBServerTemplate.Image = updateParam.Image
	return oceanbase.UpdateOBCluster(ctx, obcluster)
}

func ScaleOBServer(ctx context.Context, obzoneIdentity *param.OBZoneIdentity, scaleParam *param.ScaleOBServerParam) error {
	obcluster, err := oceanbase.GetOBCluster(ctx, obzoneIdentity.Namespace, obzoneIdentity.Name)
	if err != nil {
		return errors.Wrapf(err, "Get obcluster %s %s", obzoneIdentity.Namespace, obzoneIdentity.Name)
	}
	if obcluster.Status.Status != clusterstatus.Running {
		return errors.Errorf("Obcluster status invalid %s", obcluster.Status.Status)
	}
	found := false
	replicaChanged := false
	for idx, obzone := range obcluster.Spec.Topology {
		if obzone.Zone == obzoneIdentity.OBZoneName {
			found = true
			if obzone.Replica != scaleParam.Replicas {
				replicaChanged = true
				logger.Infof("Scale obzone %s from %d to %d", obzone.Zone, obzone.Replica, scaleParam.Replicas)
				obcluster.Spec.Topology[idx].Replica = scaleParam.Replicas
			}
		}
	}
	if !found {
		return errors.Errorf("obzone %s not found in obcluster %s %s", obzoneIdentity.OBZoneName, obzoneIdentity.Namespace, obzoneIdentity.Name)
	}
	if !replicaChanged {
		return errors.Errorf("obzone %s replica already satisfied in obcluster %s %s", obzoneIdentity.OBZoneName, obzoneIdentity.Namespace, obzoneIdentity.Name)
	}
	return oceanbase.UpdateOBCluster(ctx, obcluster)
}

func DeleteOBZone(ctx context.Context, obzoneIdentity *param.OBZoneIdentity) error {
	obcluster, err := oceanbase.GetOBCluster(ctx, obzoneIdentity.Namespace, obzoneIdentity.Name)
	if err != nil {
		return errors.Wrapf(err, "Get obcluster %s %s", obzoneIdentity.Namespace, obzoneIdentity.Name)
	}
	if obcluster.Status.Status != clusterstatus.Running {
		return errors.Errorf("Obcluster status invalid %s", obcluster.Status.Status)
	}
	newTopology := make([]apitypes.OBZoneTopology, 0)
	found := false
	for _, obzone := range obcluster.Spec.Topology {
		if obzone.Zone != obzoneIdentity.OBZoneName {
			newTopology = append(newTopology, obzone)
		} else {
			found = true
		}
	}
	if !found {
		return errors.Errorf("obzone %s not found in obcluster %s %s", obzoneIdentity.OBZoneName, obzoneIdentity.Namespace, obzoneIdentity.Name)
	}
	obcluster.Spec.Topology = newTopology
	return oceanbase.UpdateOBCluster(ctx, obcluster)
}

func AddOBZone(ctx context.Context, obclusterIdentity *param.K8sObjectIdentity, zone *param.ZoneTopology) error {
	obcluster, err := oceanbase.GetOBCluster(ctx, obclusterIdentity.Namespace, obclusterIdentity.Name)
	if err != nil {
		return errors.Wrapf(err, "Get obcluster %s %s", obclusterIdentity.Namespace, obclusterIdentity.Name)
	}
	if obcluster.Status.Status != clusterstatus.Running {
		return errors.Errorf("Obcluster status invalid %s", obcluster.Status.Status)
	}
	for _, obzone := range obcluster.Spec.Topology {
		if obzone.Zone == zone.Zone {
			return errors.Errorf("obzone %s already exists", zone.Zone)
		}
	}
	obcluster.Spec.Topology = append(obcluster.Spec.Topology, apitypes.OBZoneTopology{
		Zone:         zone.Zone,
		NodeSelector: common.KVsToMap(zone.NodeSelector),
		Replica:      zone.Replicas,
	})
	return oceanbase.UpdateOBCluster(ctx, obcluster)
}

func GetOBCluster(ctx context.Context, obclusterIdentity *param.K8sObjectIdentity) (*response.OBCluster, error) {
	obcluster, err := oceanbase.GetOBCluster(ctx, obclusterIdentity.Namespace, obclusterIdentity.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Get obcluster %s %s", obclusterIdentity.Namespace, obclusterIdentity.Name)
	}
	return buildOBClusterResponse(ctx, obcluster)
}

func DeleteOBCluster(ctx context.Context, obclusterIdentity *param.K8sObjectIdentity) error {
	return oceanbase.DeleteOBCluster(ctx, obclusterIdentity.Namespace, obclusterIdentity.Name)
}

func GetOBClusterStatistic(ctx context.Context) ([]response.OBClusterStastistic, error) {
	statisticResult := make([]response.OBClusterStastistic, 0)
	obclusterList, err := oceanbase.ListAllOBClusters(ctx)
	if err != nil {
		return statisticResult, errors.Wrap(err, "failed to list obclusters")
	}
	var (
		runningCount   int
		deletingCount  int
		operatingCount int
		failedCount    int
	)
	for _, obcluster := range obclusterList.Items {
		switch getStatisticStatus(&obcluster) {
		case StatusRunning:
			runningCount++
		case StatusDeleting:
			deletingCount++
		case StatusOperating:
			operatingCount++
		case StatusFailed:
			failedCount++
		}
	}
	statisticResult = append(statisticResult,
		response.OBClusterStastistic{
			Status: StatusRunning,
			Count:  runningCount,
		}, response.OBClusterStastistic{
			Status: StatusDeleting,
			Count:  deletingCount,
		}, response.OBClusterStastistic{
			Status: StatusOperating,
			Count:  operatingCount,
		}, response.OBClusterStastistic{
			Status: StatusFailed,
			Count:  failedCount,
		})
	return statisticResult, nil
}
