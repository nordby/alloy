---
canonical: https://grafana.com/docs/alloy/latest/reference/components/otelcol/otelcol.auth.sigv4/
aliases:
  - ../otelcol.auth.sigv4/ # /docs/alloy/latest/reference/components/otelcol.auth.sigv4/
description: Learn about otelcol.auth.sigv4
title: otelcol.auth.sigv4
---

# otelcol.auth.sigv4

`otelcol.auth.sigv4` exposes a `handler` that can be used by other `otelcol`
components to authenticate requests to AWS services using the AWS Signature Version 4 (SigV4) protocol.
For more information about SigV4 see the AWS documentation about [Signing AWS API requests][].

This component only supports client authentication. 

[Signing AWS API requests]: https://docs.aws.amazon.com/general/latest/gr/signing-aws-api-requests.html

> **NOTE**: `otelcol.auth.sigv4` is a wrapper over the upstream OpenTelemetry
> Collector `sigv4auth` extension. Bug reports or feature requests will be
> redirected to the upstream repository, if necessary.

Multiple `otelcol.auth.sigv4` components can be specified by giving them
different labels.

{{< admonition type="note" >}}
{{< param "PRODUCT_NAME" >}} must have valid AWS credentials as used by the [AWS SDK for Go][].

[AWS SDK for Go]: https://aws.github.io/aws-sdk-go-v2/docs/configuring-sdk/#specifying-credentials
{{< /admonition >}}

## Usage

```alloy
otelcol.auth.sigv4 "LABEL" {
}
```

## Arguments

Name      | Type     | Description                   | Default | Required
----------|----------|-------------------------------|---------|---------
`region`  | `string` | The AWS region to sign with.  | ""      | no
`service` | `string` | The AWS service to sign with. | ""      | no

If `region` and `service` are left empty, their values are inferred from the URL of the exporter
using the following rules:

* If the exporter URL starts with `aps-workspaces` and `service` is empty, `service` will be set to `aps`.
* If the exporter URL starts with `search-` and `service` is empty, `service` will be set to `es`.
* If the exporter URL starts with either `aps-workspaces` or `search-` and `region` is empty, `region` will be set to the value between the first and second `.` character in the exporter URL.

If none of the above rules apply, then `region` and `service` must be specified.

A list of valid AWS regions can be found on Amazon's documentation for [Regions, Availability Zones, and Local Zones][].

[Regions, Availability Zones, and Local Zones]: https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Concepts.RegionsAndAvailabilityZones.html

## Blocks

The following blocks are supported inside the definition of
`otelcol.auth.sigv4`:

Hierarchy   | Block           | Description                        | Required
------------|-----------------|------------------------------------|---------
assume_role | [assume_role][] | Configuration for assuming a role. | no
debug_metrics | [debug_metrics][] | Configures the metrics that this component generates to monitor its state. | no

[assume_role]: #assume_role-block
[debug_metrics]: #debug_metrics-block

### assume_role block

The `assume_role` block specifies the configuration needed to assume a role.

Name           | Type     | Description                                                     | Default | Required
---------------|----------|-----------------------------------------------------------------|---------|---------
`arn`          | `string` | The Amazon Resource Name (ARN) of a role to assume.             | ""      | no
`session_name` | `string` | The name of a role session.                                     | ""      | no
`sts_region`   | `string` | The AWS region where STS is used to assume the configured role. | ""      | no

If the `assume_role` block is specified in the config and `sts_region` is not set, then `sts_region` will default to the value for `region`.

For cross region authentication, `region` and `sts_region` can be set different to different values.

### debug_metrics block

{{< docs/shared lookup="reference/components/otelcol-debug-metrics-block.md" source="alloy" version="<ALLOY_VERSION>" >}}

## Exported fields

The following fields are exported and can be referenced by other components:

Name      | Type                       | Description
----------|----------------------------|----------------------------------------------------------------
`handler` | `capsule(otelcol.Handler)` | A value that other components can use to authenticate requests.

## Component health

`otelcol.auth.sigv4` is only reported as unhealthy if given an invalid
configuration.

## Debug information

`otelcol.auth.sigv4` does not expose any component-specific debug information.

## Examples

### Inferring the "region" and "service" from an "aps-workspaces" exporter endpoint

In this example the exporter endpoint starts with `aps-workspaces`. Hence `service` is inferred to be `aps`
and `region` is inferred to be `us-east-1`.

```alloy
otelcol.exporter.otlp "example" {
  client {
    endpoint = "https://aps-workspaces.us-east-1.amazonaws.com/workspaces/ws-XXX/api/v1/remote_write"
    auth     = otelcol.auth.sigv4.creds.handler
  }
}

otelcol.auth.sigv4 "creds" {
}
```

### Inferring the "region" and "service" from a "search-" exporter endpoint

In this example the exporter endpoint starts with `search-`. Hence `service` is inferred to be `es`
and `region` is inferred to be `us-east-1`.

```alloy
otelcol.exporter.otlp "example" {
  client {
    endpoint = "https://search-my-domain.us-east-1.es.amazonaws.com/_search?q=house"
    auth     = otelcol.auth.sigv4.creds.handler
  }
}

otelcol.auth.sigv4 "creds" {
}
```

### Specifying "region" and "service" explicitly

In this example the exporter endpoint does not begin with `search-` or with `aps-workspaces`.
Hence, we need to specify `region` and `service` explicitly.

```alloy
otelcol.exporter.otlp "example" {
  client {
    endpoint = "my-otlp-grpc-server:4317"
    auth     = otelcol.auth.sigv4.creds.handler
  }
}

otelcol.auth.sigv4 "creds" {
    region = "example_region"
    service = "example_service"
}
```

### Specifying "region" and "service" explicitly and adding a "role" to assume

In this example we have also specified configuration to assume a role. `sts_region` hasn't been provided, so it will default to the value of `region` which is `example_region`.

```alloy
otelcol.exporter.otlp "example" {
  client {
    endpoint = "my-otlp-grpc-server:4317"
    auth     = otelcol.auth.sigv4.creds.handler
  }
}

otelcol.auth.sigv4 "creds" {
  region  = "example_region"
  service = "example_service"

  assume_role {
    session_name = "role_session_name"
  }
}
```
