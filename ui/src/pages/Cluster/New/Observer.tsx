import { intl } from '@/utils/intl';
import {
  Button,
  Card,
  Col,
  Form,
  Input,
  InputNumber,
  Row,
  Tooltip,
} from 'antd';

import ClassSelect from '@/components/ClassSelect';
import { MINIMAL_CONFIG, SUFFIX_UNIT } from '@/constants';
import { MIRROR_SERVER } from '@/constants/doc';
import { clone } from 'lodash';
import styles from './index.less';

const observerToolTipText = intl.formatMessage({
  id: 'OBDashboard.Cluster.New.Observer.TheImageShouldBeFully',
  defaultMessage:
    '镜像应写全 registry/image:tag，例如 oceanbase/oceanbase-cloud-native:4.2.0.0-101000032023091319',
});
export default function Observer({ storageClasses, form }: any) {
  const CustomItem = (prop: any) => {
    const { label } = prop;
    return (
      <Form.Item
        {...prop}
        rules={[
          {
            required: true,
            message: intl.formatMessage(
              {
                id: 'OBDashboard.Cluster.New.Observer.EnterLabel',
                defaultMessage: '请输入{label}',
              },
              { label: label },
            ),
          },
        ]}
      >
        {prop.children}
      </Form.Item>
    );
  };

  const setMinimalConfiguration = () => {
    let originObserver = clone(form.getFieldsValue());
    form.setFieldsValue({
      ...originObserver,
      observer: {
        ...originObserver.observer,
        resource: {
          cpu: MINIMAL_CONFIG.cpu,
          memory: MINIMAL_CONFIG.memory,
        },
        storage: {
          ...originObserver.observer.storage,
          data: {
            size: MINIMAL_CONFIG.data,
          },
          log: {
            size: MINIMAL_CONFIG.log,
          },
          redoLog: {
            size: MINIMAL_CONFIG.redoLog,
          },
        },
      },
    });
  };

  return (
    <Col span={24}>
      <Card
        title="Observer"
        extra={
          <Button type="primary" onClick={setMinimalConfiguration}>
            {intl.formatMessage({
              id: 'Dashboard.Cluster.New.Observer.MinimumSpecificationConfiguration',
              defaultMessage: '最小规格配置',
            })}
          </Button>
        }
      >
        <Tooltip title={observerToolTipText}>
          <CustomItem
            style={{ width: '50%' }}
            label={
              <>
                {intl.formatMessage({
                  id: 'Dashboard.Cluster.New.Observer.Image',
                  defaultMessage: '镜像',
                })}{' '}
                <a href={MIRROR_SERVER} rel="noreferrer" target="_blank">
                  {intl.formatMessage({
                    id: 'Dashboard.Cluster.New.Observer.ImageList',
                    defaultMessage: '（镜像列表）',
                  })}
                </a>
              </>
            }
            name={['observer', 'image']}
          >
            <Input
              placeholder={intl.formatMessage({
                id: 'OBDashboard.Cluster.New.Observer.EnterAnImage',
                defaultMessage: '请输入镜像',
              })}
            />
          </CustomItem>
        </Tooltip>
        <Row>
          <Col>
            <p className={styles.titleText}>
              {intl.formatMessage({
                id: 'OBDashboard.Cluster.New.Observer.Resources',
                defaultMessage: '资源',
              })}
            </p>

            <div className={styles.resourceContent}>
              <CustomItem
                className={styles.leftContent}
                label="cpu"
                name={['observer', 'resource', 'cpu']}
              >
                <InputNumber
                  min={MINIMAL_CONFIG.cpu}
                  placeholder={intl.formatMessage({
                    id: 'OBDashboard.Cluster.New.Observer.PleaseEnter',
                    defaultMessage: '请输入',
                  })}
                />
              </CustomItem>
              <CustomItem
                label={intl.formatMessage({
                  id: 'OBDashboard.Cluster.New.Observer.Memory',
                  defaultMessage: '内存',
                })}
                name={['observer', 'resource', 'memory']}
              >
                <InputNumber
                  min={MINIMAL_CONFIG.memory}
                  addonAfter={SUFFIX_UNIT}
                  placeholder={intl.formatMessage({
                    id: 'OBDashboard.Cluster.New.Observer.PleaseEnter',
                    defaultMessage: '请输入',
                  })}
                />
              </CustomItem>
            </div>
          </Col>
        </Row>
        <p className={styles.titleText}>storage</p>

        <Row gutter={16}>
          <Col span={8}>
            <p className={styles.subTitleText}>
              {intl.formatMessage({
                id: 'OBDashboard.Cluster.New.Observer.Data',
                defaultMessage: '数据',
              })}
            </p>
            <div className={styles.dataContent}>
              <CustomItem
                className={styles.leftContent}
                label="size"
                name={['observer', 'storage', 'data', 'size']}
              >
                <InputNumber
                  min={MINIMAL_CONFIG.data}
                  addonAfter={SUFFIX_UNIT}
                  placeholder={intl.formatMessage({
                    id: 'OBDashboard.Cluster.New.Observer.PleaseEnter',
                    defaultMessage: '请输入',
                  })}
                />
              </CustomItem>
              <CustomItem
                label="storageClass"
                name={['observer', 'storage', 'data', 'storageClass']}
              >
                {storageClasses && (
                  <ClassSelect
                    name={['observer', 'storage', 'data', 'storageClass']}
                    form={form}
                    selectList={storageClasses}
                  />
                )}

                {/* <CustomSelect /> */}
              </CustomItem>
            </div>
          </Col>
          <Col span={8}>
            <p className={styles.subTitleText}>
              {intl.formatMessage({
                id: 'OBDashboard.Cluster.New.Observer.Log',
                defaultMessage: '日志',
              })}
            </p>
            <div className={styles.logContent}>
              <CustomItem
                className={styles.leftContent}
                label="size"
                name={['observer', 'storage', 'log', 'size']}
              >
                <InputNumber
                  min={MINIMAL_CONFIG.log}
                  addonAfter={SUFFIX_UNIT}
                  placeholder={intl.formatMessage({
                    id: 'OBDashboard.Cluster.New.Observer.PleaseEnter',
                    defaultMessage: '请输入',
                  })}
                />
              </CustomItem>
              <CustomItem
                label="storageClass"
                name={['observer', 'storage', 'log', 'storageClass']}
              >
                {storageClasses && (
                  <ClassSelect
                    form={form}
                    name={['observer', 'storage', 'log', 'storageClass']}
                    selectList={storageClasses}
                  />
                )}
              </CustomItem>
            </div>
          </Col>
          <Col span={8}>
            <p className={styles.subTitleText}>redoLog</p>
            <div className={styles.redologContent}>
              <CustomItem
                className={styles.leftContent}
                label="size"
                name={['observer', 'storage', 'redoLog', 'size']}
              >
                <InputNumber
                  min={MINIMAL_CONFIG.redoLog}
                  addonAfter={SUFFIX_UNIT}
                  placeholder={intl.formatMessage({
                    id: 'OBDashboard.Cluster.New.Observer.PleaseEnter',
                    defaultMessage: '请输入',
                  })}
                />
              </CustomItem>
              <CustomItem
                label="storageClass"
                validateTrigger="onBlur"
                name={['observer', 'storage', 'redoLog', 'storageClass']}
              >
                {storageClasses && (
                  <ClassSelect
                    form={form}
                    name={['observer', 'storage', 'redoLog', 'storageClass']}
                    selectList={storageClasses}
                  />
                )}
              </CustomItem>
            </div>
          </Col>
        </Row>
      </Card>
    </Col>
  );
}
