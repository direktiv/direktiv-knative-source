# Direktiv Knative Source

This repository contains the Direktiv source for Knative. It forwards events created in [Direktiv](https://github.com/direktiv/direktiv) to Knative. These events can be either coming from an explicit [GenerateEvent state](https://docs.direktiv.io/docs/specification.html#generateeventstate) or actions within the system, e.g. creating namespaces.

<p align="center">
<img src="../assets/source.png"/>
</p>

The source is implemented as [ContainerSource](https://knative.dev/docs/eventing/samples/container-source/) and requires one argument to connect to the Direktiv instance via GRPC. Direktiv will then stream events to this source.

```yaml
apiVersion: sources.knative.dev/v1
kind: ContainerSource
metadata:
  name: direktiv-source
spec:
  template:
    spec:
      containers:
        - image: direktiv/direktiv-source
          name: direktiv-source
          args:
            - --direktiv=direktiv-flow.default:3333
  sink:
    ref:
      apiVersion: eventing.knative.dev/v1
      kind: Broker
      name: default
```

It is required to enable eventing during the Helm installation of [Direktiv](https://github.com/direktiv/direktiv):

```yaml
eventing:
  enabled: true
```

To install an in-memory channel and MT broker for testing the script *scripts/install-eventing.sh* in the Direktiv Github repository can be used. 

## Example

For a full Direktiv, Knative, Kafka example, click [here](https://docs.direktiv.io/docs/events/knative/example.html).
