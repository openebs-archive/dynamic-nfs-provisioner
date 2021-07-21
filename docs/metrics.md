Metrics are useful for debugging and monitoring the application in real-time. dynamic-nfs-provisioner uses [Prometheus](https://github.com/prometheus/prometheus) to report the metrics. dynamic-nfs-provisioner doesn't persist its metrics. If the dynamic-nfs-provisioner pod restarts, metrics will be reset.

By default, Dynamic-nfs-provisioner exposes the available metrics at path `<POD_IP>:9500/metrics`. You can change the metrics path by setting `metrics-path` and `listen-address` arguments in the pod template.

You can collect the metrics using cURL request to endpoint `<POD_IP>:9500/metrics`.

Follow the [Prometheus getting started doc](https://prometheus.io/docs/prometheus/latest/getting_started/) to spin up a Prometheus server to collect the dynamic-nfs-provisioner metrics.

Dynamic-nfs-provisioner metrics name has an `nfs_volume_provisioner` prefix as its namespace and subsystem prefix `persistentvolume`.

Metrics under `persistentvolume` subsystem describe the status of provisioner. To detect the failure in volume provisioning, These metrics should be monitored closely.

The following table lists all the metrics available under `persistentvolume` subsystem:

| Name | Description |
| ---- | ----------- |
| create_total | Total number of persistent volumes created |
| create_failed_total | Total number of persistent volume creation failed attempts |
| delete_total | Total number of persistent volumes deleted |
| delete_failed_total | Total number of persistent volume delete failed attempts |

- `create_total` records the total number of successfully provisioned volume requests. This counter should increase over the time if dynamic-nfs-provisioner remains in healthy condition.
- `create_failed_total` record the total number of failed provisioning request. This counter indicates temporary failure in provisioning because of invalid requests. Rising `create_failed_total` indicates the dynamic-provisioner is unable to serve the provisioning request because of the issue in provisioner or cluster health.
- `delete_total` records the total number of successfully de-provisioned volume requests. This counter should increase over the time if dynamic-nfs-provisioner remains in healthy condition.
- `delete_failed_total` records the total number of failed de-provisioning requests. This counter indicates temporary failure in de-provisioning because of invalid requests. Rising `delete_failed_total` indicates the dynamic-provisioner is unable to serve the de-provisioning request because of the issue in provisioner or cluster health.


To get the nfs server statistics, you can use the node_exporter. node_exporter exposes the nfs client metrics through collector `nfs` and nfs server metrics through collector `nfsd`. A detailed guide on how to install node_exporter can be found [here](https://prometheus.io/docs/guides/node-exporter/).


