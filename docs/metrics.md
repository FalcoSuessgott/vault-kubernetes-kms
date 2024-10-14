# Prometheus Metrics
Beginning with `v0.3.0` `vault-kubernetes-kms` exposes metrics under `<Pod or Service>:8080/metrics` (You can configure the metrics port by specifying `-health-port` or setting `HEALTH_PORT`).

The following metrics are available:

!!! note
        If you miss a specific metrics, just let me know and we can discuss a possible implementation

## Current metrics
| Metric Name                                                | Type      | Description                                                 |
|------------------------------------------------------------|-----------|-------------------------------------------------------------|
| vault_kubernetes_kms_encryption_operations_total           | Counter   | total number of encryption operations                       |
| vault_kubernetes_kms_decryption_operations_total           | Counter   | total number of decryption operations                       |
| vault_kubernetes_kms_encryption_errors_total               | Counter   | total number of errors during encryption operations         |
| vault_kubernetes_kms_decryption_errors_total               | Counter   | total number of errors during decryption operations         |
| vault_kubernetes_kms_encryption_operation_duration_seconds | Histogram | duration of encryption operations                           |
| vault_kubernetes_kms_decryption_operation_duration_seconds | Histogram | duration of decryption operations                           |
| vault_kubernetes_kms_vault_requests_total                  | Counter   | total number of API requests sent to vault                  |
| vault_kubernetes_kms_vault_requests_errors_total           | Counter   | total number of errors during API requests sent to vault    |
| vault_kubernetes_kms_vault_request_duration_seconds        | Histogram | duration of API requests sent to vault                      |
| vault_kubernetes_kms_token_renewals_total                  | Counter   | total number of token renewals                              |
| vault_kubernetes_kms_token_expiry_seconds                  | Gauge     | time remaining until the current token expires              |

Plus the metrics defined in the [Prometheus ProcessCollector](https://github.com/prometheus/client_golang/blob/main/prometheus/process_collector.go#L38)

## Example
Since the `kubelet` automatically sends healhtZ request to the plugin, you should immediately see an increasing in some metrics.
To see the metrics, you can simply port-forward to the pod:

```bash
$> kubectl port-forward -n kube-system pod/vault-kubernetes-kms-kms-control-plane 8080:8080
```

And in another terminal run:

```bash
$> curl -s http://127.0.0.1:8080/metrics
# HELP vault_kubernetes_kms_process_cpu_seconds_total Total user and system CPU time spent in seconds.
# TYPE vault_kubernetes_kms_process_cpu_seconds_total counter
vault_kubernetes_kms_process_cpu_seconds_total 2.16
# HELP vault_kubernetes_kms_process_max_fds Maximum number of open file descriptors.
# TYPE vault_kubernetes_kms_process_max_fds gauge
vault_kubernetes_kms_process_max_fds 1.048576e+06
# HELP vault_kubernetes_kms__process_network_receive_bytes_total Number of bytes received by the process over the network.
# TYPE vault_kubernetes_kms__process_network_receive_bytes_total counter
vault_kubernetes_kms_process_network_receive_bytes_total 2.9131506e+07
# HELP vault_kubernetes_kms_process_network_transmit_bytes_total Number of bytes sent by the process over the network.
# TYPE vault_kubernetes_kms_process_network_transmit_bytes_total counter
vault_kubernetes_kms_process_network_transmit_bytes_total 1.7502325e+07
# HELP vault_kubernetes_kms_process_open_fds Number of open file descriptors.
# TYPE vault_kubernetes_kms_process_open_fds gauge
vault_kubernetes_kms_process_open_fds 10
# HELP vault_kubernetes_kms_process_resident_memory_bytes Resident memory size in bytes.
# TYPE vault_kubernetes_kms_process_resident_memory_bytes gauge
vault_kubernetes_kms_process_resident_memory_bytes 1.8931712e+07
# HELP vault_kubernetes_kms_process_start_time_seconds Start time of the process since unix epoch in seconds.
# TYPE vault_kubernetes_kms_process_start_time_seconds gauge
vault_kubernetes_kms_process_start_time_seconds 1.72456278372e+09
# HELP vault_kubernetes_kms_process_virtual_memory_bytes Virtual memory size in bytes.
# TYPE vault_kubernetes_kms_process_virtual_memory_bytes gauge
vault_kubernetes_kms_process_virtual_memory_bytes 1.269485568e+09
# HELP vault_kubernetes_kms_process_virtual_memory_max_bytes Maximum amount of virtual memory available in bytes.
# TYPE vault_kubernetes_kms_process_virtual_memory_max_bytes gauge
vault_kubernetes_kms_process_virtual_memory_max_bytes 1.8446744073709552e+19
# HELP vault_kubernetes_kms_decryption_errors_total total number of errors during decryption operations
# TYPE vault_kubernetes_kms_decryption_errors_total counter
vault_kubernetes_kms_decryption_errors_total 0
# HELP vault_kubernetes_kms_decryption_operation_duration_seconds duration of decryption operations
# TYPE vault_kubernetes_kms_decryption_operation_duration_seconds histogram
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="0.005"} 92
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="0.01"} 99
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="0.025"} 100
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="0.05"} 101
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="0.1"} 101
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="0.25"} 101
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="0.5"} 101
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="1"} 101
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="2.5"} 101
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="5"} 101
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="10"} 101
vault_kubernetes_kms_decryption_operation_duration_seconds_bucket{le="+Inf"} 101
vault_kubernetes_kms_decryption_operation_duration_seconds_sum 0.333429742
vault_kubernetes_kms_decryption_operation_duration_seconds_count 101
# HELP vault_kubernetes_kms_decryption_operations_total total number of decryption operations
# TYPE vault_kubernetes_kms_decryption_operations_total counter
vault_kubernetes_kms_decryption_operations_total 101
# HELP vault_kubernetes_kms_encryption_errors_total total number of errors during encryption operations
# TYPE vault_kubernetes_kms_encryption_errors_total counter
vault_kubernetes_kms_encryption_errors_total 0
# HELP vault_kubernetes_kms_encryption_operation_duration_seconds duration of encryption operations
# TYPE vault_kubernetes_kms_encryption_operation_duration_seconds histogram
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="0.005"} 39
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="0.01"} 86
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="0.025"} 95
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="0.05"} 101
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="0.1"} 102
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="0.25"} 102
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="0.5"} 102
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="1"} 102
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="2.5"} 102
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="5"} 102
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="10"} 102
vault_kubernetes_kms_encryption_operation_duration_seconds_bucket{le="+Inf"} 102
vault_kubernetes_kms_encryption_operation_duration_seconds_sum 0.910400441
vault_kubernetes_kms_encryption_operation_duration_seconds_count 102
# HELP vault_kubernetes_kms_encryption_operations_total total number of encryption operations
# TYPE vault_kubernetes_kms_encryption_operations_total counter
vault_kubernetes_kms_encryption_operations_total 102
# HELP vault_kubernetes_kms_operation_latency_seconds latency distribution of KMS operations
# TYPE vault_kubernetes_kms_operation_latency_seconds summary
vault_kubernetes_kms_operation_latency_seconds_sum 0
vault_kubernetes_kms_operation_latency_seconds_count 0
# HELP vault_kubernetes_kms_operation_throughput_seconds procession rate of KMS operations
# TYPE vault_kubernetes_kms_operation_throughput_seconds summary
vault_kubernetes_kms_operation_throughput_seconds_sum 0
vault_kubernetes_kms_operation_throughput_seconds_count 0
# HELP vault_kubernetes_kms_provider_cpu_usage_seconds cpu usage of the KMS provider
# TYPE vault_kubernetes_kms_provider_cpu_usage_seconds gauge
vault_kubernetes_kms_provider_cpu_usage_seconds 0
# HELP vault_kubernetes_kms_provider_memory_usage_bytes memory usage of the KMS provider
# TYPE vault_kubernetes_kms_provider_memory_usage_bytes gauge
vault_kubernetes_kms_provider_memory_usage_bytes 0
# HELP vault_kubernetes_kms_provider_uptime_seconds total uptime of the KMS provider
# TYPE vault_kubernetes_kms_provider_uptime_seconds gauge
vault_kubernetes_kms_provider_uptime_seconds 390.201346128
# HELP vault_kubernetes_kms_token_expiry_seconds time remaining until the current token expires
# TYPE vault_kubernetes_kms_token_expiry_seconds gauge
vault_kubernetes_kms_token_expiry_seconds 0
# HELP vault_kubernetes_kms_token_renewals_total total number of token renewals
# TYPE vault_kubernetes_kms_token_renewals_total counter
vault_kubernetes_kms_token_renewals_total 0
# HELP vault_kubernetes_kms_vault_request_duration_seconds duration of API requests sent to vault
# TYPE vault_kubernetes_kms_vault_request_duration_seconds histogram
vault_kubernetes_kms_vault_request_duration_seconds_bucket{le="0.05"} 343
vault_kubernetes_kms_vault_request_duration_seconds_bucket{le="0.1"} 343
vault_kubernetes_kms_vault_request_duration_seconds_bucket{le="0.25"} 344
vault_kubernetes_kms_vault_request_duration_seconds_bucket{le="0.5"} 344
vault_kubernetes_kms_vault_request_duration_seconds_bucket{le="1"} 344
vault_kubernetes_kms_vault_request_duration_seconds_bucket{le="2.5"} 344
vault_kubernetes_kms_vault_request_duration_seconds_bucket{le="5"} 344
vault_kubernetes_kms_vault_request_duration_seconds_bucket{le="10"} 344
vault_kubernetes_kms_vault_request_duration_seconds_bucket{le="+Inf"} 344
vault_kubernetes_kms_vault_request_duration_seconds_sum 0.9637442500000003
vault_kubernetes_kms_vault_request_duration_seconds_count 344
# HELP vault_kubernetes_kms_vault_requests_errors_total total number of errors during API requests sent to vault
# TYPE vault_kubernetes_kms_vault_requests_errors_total counter
vault_kubernetes_kms_vault_requests_errors_total 19
# HELP vault_kubernetes_kms_vault_requests_total total number of API requests sent to vault
# TYPE vault_kubernetes_kms_vault_requests_total counter
vault_kubernetes_kms_vault_requests_total 344
```
