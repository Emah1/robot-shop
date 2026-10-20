# Robot Shop — DevOps Capstone Project

Stan's Robot Shop is a sample microservice application for learning containerised application orchestration, deployment, monitoring, and troubleshooting. This repository is being used as a DevOps capstone project to demonstrate Kubernetes deployment and observability.

## Application Overview

The application consists of the following technologies and services:

- Node.js (Express)
- Java (Spring Boot)
- Python (Flask)
- Go
- PHP (Apache)
- MongoDB
- Redis
- MySQL
- RabbitMQ
- Nginx
- AngularJS

For more information about the original application, see the [Robot Shop introduction](https://www.instana.com/blog/stans-robot-shop-sample-microservice-application/).

## DevOps Capstone Enhancements

This project includes Kubernetes and Helm configuration changes, along with observability components for monitoring application health, collecting logs, and investigating distributed requests.

### Observability Stack

| Component | Purpose |
|---|---|
| Prometheus | Collects and stores metrics |
| Grafana | Visualizes metrics through dashboards |
| Fluent Bit | Collects and forwards application logs |
| Loki | Aggregates and queries logs |
| Jaeger | Provides distributed tracing |
| OpenTelemetry | Instruments the dispatch service for tracing |

Configuration files for the observability stack are located in the `observability/` directory.

### Kubernetes and Helm

The application is deployed to a local Kubernetes cluster using Kind and Helm.

- Application namespace: `robot-shop`
- Monitoring namespace: `monitoring`
- Helm templates configure application deployments and services.
- RabbitMQ tracing configuration is provided through a ConfigMap.
- The dispatch service includes OpenTelemetry instrumentation.

### Build from Source

To build the application images, use Docker Compose from the project root:

```bash
docker-compose build
```

Some original image builds may require an Instana agent key. Follow the original application's instructions if you need to build those components.

### Run Locally with Docker Compose

Pull the prebuilt images:

```bash
docker-compose pull
```

Start the application:

```bash
docker-compose up
```

To start the application with the optional load generator:

```bash
docker-compose -f docker-compose.yaml -f docker-compose-load.yaml up
```

### Kubernetes Deployment

The Helm chart is located in `K8s/helm/`.

Check the current Kubernetes contexts and namespaces:

```bash
kubectl config get-contexts
kubectl get namespaces
kubectl get pods -n robot-shop
kubectl get pods -n monitoring
```

Use the project's Helm chart instructions in [`K8s/helm/README.md`](K8s/helm/README.md) for deployment and configuration details.

### Accessing the Store

For a local Docker Compose deployment, the store is typically available at:

http://localhost:8080

For a Kubernetes deployment, inspect the web service:

```bash
kubectl get svc -n robot-shop
```

Access the application using the appropriate service or port-forward configuration for your environment.

### Prometheus Metrics

The original cart and payment services expose Prometheus metrics at `/metrics`.

Example requests:

```bash
curl http://localhost:8080/api/cart/metrics
curl http://localhost:8080/api/payment/metrics
```

The available metrics depend on the service and its configuration.

### Load Generation

The original project includes a load-generation utility in the `load-gen/` directory. See [`load-gen/README.md`](load-gen/README.md) for usage instructions.

Load generation can help exercise the application and produce traffic for monitoring and troubleshooting.

## Project Structure

```text
.
├── K8s/
│   └── helm/
│       └── templates/
├── dispatch/
├── observability/
├── load-gen/
└── README.md
```

The repository may contain additional application service directories and configuration files.

## Verification

The local development environment was used to test the Kubernetes application and observability stack.

The following checks were performed during development:

- Kubernetes application pods reached the Running and Ready states.
- Prometheus targets were reported as up.
- Fluent Bit forwarded application logs to Loki.
- Grafana and Jaeger interfaces were accessible locally.
- Dispatch service tracing instrumentation was added.

**Note:** Successful deployment and availability of a component do not necessarily mean that every distributed trace or dashboard has been fully verified. Confirm trace ingestion in Jaeger when testing the dispatch workflow.

## Security and Cost Considerations

This is a learning and demonstration project, not a production-ready reference implementation.

- Do not commit passwords, access keys, tokens, or other secrets.
- Review container images and security settings before production use.
- Review Kubernetes resource requests and limits before deployment.
- Test Terraform plans before applying infrastructure changes.
- Do not deploy billable cloud infrastructure without reviewing and approving the expected costs.

## References

- [Original Robot Shop repository](https://github.com/instana/robot-shop)
- [Kubernetes documentation](https://kubernetes.io/docs/)
- [Helm documentation](https://helm.sh/docs/)
- [Prometheus documentation](https://prometheus.io/docs/)
- [Grafana documentation](https://grafana.com/docs/)
- [Fluent Bit documentation](https://docs.fluentbit.io/)
- [Loki documentation](https://grafana.com/docs/loki/latest/)
- [Jaeger documentation](https://www.jaegertracing.io/docs/)
- [OpenTelemetry documentation](https://opentelemetry.io/docs/)
