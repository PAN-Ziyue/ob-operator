import { request } from '@umijs/max';

const tenantPrefix = '/api/v1/obtenants';

export async function getAllTenants(
  obcluster?: string,
): Promise<API.TenantsListResponse> {
  let query = '';
  if (obcluster) query = `?obcluster=${obcluster}`;
  return request(`${tenantPrefix}${query}`, {
    method: 'GET',
  });
}

export async function getTenant({
  ns,
  name,
}: API.NamespaceAndName): Promise<API.TenantBasicInfoResponse> {
  let r = await request(`${tenantPrefix}/${ns}/${name}`, {
    method: 'GET',
  });
  let res: API.TenantBasicInfo = {
    info: {
      charset: '',
      clusterName: '',
      tenantName: '',
      tenantRole: '',
      unitNumber: 0,
      status: '',
      name: '',
      namespace: '',
      locality: '',
    },
    source: {},
    replicas: [],
  };
  if (r.successful) {
    Object.keys(res.info).forEach((key) => {
      res.info[key] = r.data[key];
    });
    if (r.data.primaryTenant) res.source!.primaryTenant = r.data.primaryTenant;
    if (r.data.restoreSource?.archiveSource)
      res.source!.archiveSource = r.data.restoreSource.archiveSource;
    if (r.data.restoreSource?.bakDataSource)
      res.source!.bakDataSource = r.data.restoreSource.bakDataSource;
    if (r.data.restoreSource?.until)
      res.source!.until = r.data.restoreSource.until;

    res.replicas = r.data.topology;
    return {
      ...r,
      data: res,
    };
  }
  return r;
}

export async function createTenant({
  ns,
  name,
  ...body
}: API.NamespaceAndName & API.TenantBody): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}`, {
    method: 'PUT',
    data: body,
  });
}

export async function deleteTenent({
  ns,
  name,
}: API.NamespaceAndName): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}`, {
    method: 'DELETE',
  });
}
// 创建特定租户的备份策略，密码应采用AES加密 没有body??
export async function createPolicyOfTenant({
  ns,
  name,
}: API.NamespaceAndName): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}/backupPolicy`, {
    method: 'PUT',
  });
}

export async function updatePolicyOfTenant({
  ns,
  name,
  ...body
}: API.NamespaceAndName & API.TenantPolicy): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}/backupPolicy`, {
    method: 'POST',
    data: body,
  });
}

export async function deletePolicyOfTenant({
  ns,
  name,
}: API.NamespaceAndName): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}/backupPolicy`, {
    method: 'DELETE',
  });
}

export async function getBackupPolicy({
  ns,
  name,
}: API.NamespaceAndName): Promise<API.BackupPolicyResponse> {
  let r = await request(`${tenantPrefix}/${ns}/${name}/backupPolicy`);
  const keys = [
    'destType',
    'archivePath',
    'bakDataPath',
    'scheduleType',
    'scheduleTime',
    'scheduleDates',
  ];

  if (r.successful) {
    return {
      ...r,
      data: keys.reduce((pre, cur) => {
        pre[cur] = r.data[cur];
        return pre;
      }, {}),
    };
  }
  return r;
}

export async function getBackupJobs({
  ns,
  name,
  type,
  limit = 10
}: API.NamespaceAndName & {
  type: API.JobType;
  limit?: number;
}): Promise<API.BackupJobsResponse> {
  let r = await request(
    `${tenantPrefix}/${ns}/${name}/backup/${type}/jobs?limit=${limit}`,
  );
  let res: API.BackupJob[] = [];
  if (r.successful) {
    res = r.data.map((job: API.BackupJob) => ({
      encryptionSecret: job.encryptionSecret,
      endTime: job.endTime,
      name: job.name,
      path: job.path,
      startTime: job.startTime,
      status: job.status,
      statusInDatabase: job.statusInDatabase,
      type: job.type,
    }));
    return {
      ...r,
      data: res,
    };
  }
  return r;
}

// 备租户回放日志
export async function replayLogOfTenant({
  ns,
  name,
  ...body
}: API.NamespaceAndName & API.ReplayLogType): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}/logreplay`, {
    method: 'POST',
    data: body,
  });
}

export async function changeTenantRole({
  ns,
  name,
}: API.NamespaceAndName): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}/role`, {
    method: 'POST',
  });
}
export async function changeTenantPassword({
  ns,
  name,
  ...body
}: API.NamespaceAndName & API.UserCredentials): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}/userCredentials`, {
    method: 'POST',
    data: body,
  });
}

export async function modifyUnitNumber({
  ns,
  name,
  ...body
}: API.NamespaceAndName & API.UnitNumber): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}/unitNumber`, {
    method: 'PUT',
    data: body,
  });
}

// 升级特定租户的租户兼容版本以匹配集群版本
export async function upgradeTenantCompatibilityVersion({
  ns,
  name,
}: API.NamespaceAndName): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}/version`, {
    method: 'POST',
  });
}

export async function modifyUnitConfig({
  ns,
  name,
  zone,
  ...body
}: API.NamespaceAndName & {
  zone: string;
} & API.UnitConfig): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}/${zone}`, {
    method: 'PUT',
    data: body,
  });
}

export async function patchTenantConfiguration({
  ns,
  name,
  ...body
}: API.NamespaceAndName &
  API.PatchTenantConfiguration): Promise<API.CommonResponse> {
  return request(`${tenantPrefix}/${ns}/${name}`, {
    method: 'PATCH',
    data: body,
  });
}
