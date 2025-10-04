# Deployment Guide

This guide covers deploying the Job Coordinator to production environments.

## Deployment Options

- **Docker**: Containerized deployment for any environment
- **Google Cloud Run**: Fully managed serverless container platform
- **Kubernetes**: Self-managed container orchestration
- **Binary**: Direct deployment on VMs or bare metal

## Docker Deployment

### Build Docker Image

The project includes a multi-stage Dockerfile for optimized builds:

```bash
docker build -t job-coordinator:latest .
```

### Run Docker Container

**Stdout Mode**:
```bash
docker run -it \
  -v /data/bundles:/data/bundles \
  -p 8080:8080 \
  job-coordinator:latest \
  --job-id="docker-job" \
  --base-path="/data/bundles" \
  --measures-to-run="/measures/measure1.json" \
  --distributor-type=stdout \
  --auto-start=true
```

**File Mode**:
```bash
docker run -d \
  -v /data/bundles:/data/bundles \
  -v /data/output:/output \
  -p 8080:8080 \
  job-coordinator:latest \
  --job-id="docker-job" \
  --base-path="/data/bundles" \
  --measures-to-run="/measures/measure1.json" \
  --distributor-type=file \
  --distributor-config='{"output_dir":"/output"}' \
  --auto-start=true
```

**Pub/Sub Mode**:
```bash
docker run -d \
  -v /data/bundles:/data/bundles \
  -v /credentials:/credentials \
  -e GOOGLE_APPLICATION_CREDENTIALS=/credentials/key.json \
  -p 8080:8080 \
  job-coordinator:latest \
  --job-id="docker-job" \
  --base-path="/data/bundles" \
  --measures-to-run="/measures/measure1.json" \
  --distributor-type=pubsub \
  --distributor-config='{"project_id":"my-project","topic":"work-units"}' \
  --auto-start=true
```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  coordinator:
    build: .
    image: job-coordinator:latest
    container_name: job-coordinator
    ports:
      - "8080:8080"
    volumes:
      - ./data/bundles:/data/bundles
      - ./data/output:/output
    environment:
      - JOB_COORDINATOR_JOB_ID=compose-job
      - JOB_COORDINATOR_BASE_PATH=/data/bundles
      - JOB_COORDINATOR_MEASURES_TO_RUN=/measures/measure1.json,/measures/measure2.json
      - JOB_COORDINATOR_DISTRIBUTOR_TYPE=file
      - JOB_COORDINATOR_DISTRIBUTOR_CONFIG={"output_dir":"/output"}
      - JOB_COORDINATOR_AUTO_START=true
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

Run with Docker Compose:

```bash
docker-compose up -d
```

## Google Cloud Run Deployment

Cloud Run is ideal for serverless, fully managed deployments.

### Prerequisites

1. **Google Cloud SDK**: Install [gcloud CLI](https://cloud.google.com/sdk/docs/install)
2. **Enable APIs**:
   ```bash
   gcloud services enable run.googleapis.com
   gcloud services enable containerregistry.googleapis.com
   ```

3. **Authenticate**:
   ```bash
   gcloud auth login
   gcloud config set project YOUR_PROJECT_ID
   ```

### Build and Push Image

**Option 1: Cloud Build**:
```bash
gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/job-coordinator
```

**Option 2: Local Build + Push**:
```bash
docker build -t gcr.io/YOUR_PROJECT_ID/job-coordinator .
docker push gcr.io/YOUR_PROJECT_ID/job-coordinator
```

### Deploy to Cloud Run

**Basic Deployment**:
```bash
gcloud run deploy job-coordinator \
  --image gcr.io/YOUR_PROJECT_ID/job-coordinator \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --port 8080 \
  --memory 2Gi \
  --cpu 2 \
  --timeout 3600 \
  --set-env-vars "JOB_COORDINATOR_JOB_ID=cloudrun-job" \
  --set-env-vars "JOB_COORDINATOR_BASE_PATH=/data/bundles" \
  --set-env-vars "JOB_COORDINATOR_MEASURES_TO_RUN=/measures/measure1.json" \
  --set-env-vars "JOB_COORDINATOR_DISTRIBUTOR_TYPE=pubsub" \
  --set-env-vars 'JOB_COORDINATOR_DISTRIBUTOR_CONFIG={"project_id":"YOUR_PROJECT_ID","topic":"work-units"}' \
  --set-env-vars "JOB_COORDINATOR_AUTO_START=false"
```

**With Cloud Storage Mount** (for bundles):
```bash
gcloud run deploy job-coordinator \
  --image gcr.io/YOUR_PROJECT_ID/job-coordinator \
  --platform managed \
  --region us-central1 \
  --execution-environment gen2 \
  --add-volume name=bundles,type=cloud-storage,bucket=YOUR_BUCKET_NAME \
  --add-volume-mount volume=bundles,mount-path=/data/bundles \
  --set-env-vars "JOB_COORDINATOR_BASE_PATH=/data/bundles" \
  --other-flags-as-above
```

**With Service Account** (for Pub/Sub):
```bash
gcloud run deploy job-coordinator \
  --image gcr.io/YOUR_PROJECT_ID/job-coordinator \
  --service-account coordinator-sa@YOUR_PROJECT_ID.iam.gserviceaccount.com \
  --other-flags-as-above
```

### Cloud Run Configuration

**Health Checks**:
Cloud Run automatically uses `/health` endpoint for health checks.

**Autoscaling**:
```bash
gcloud run services update job-coordinator \
  --min-instances 0 \
  --max-instances 10 \
  --concurrency 80
```

**Traffic Management**:
```bash
# Deploy new revision with 0% traffic
gcloud run deploy job-coordinator --no-traffic --tag v2

# Gradually shift traffic
gcloud run services update-traffic job-coordinator --to-revisions v2=50

# Rollback if needed
gcloud run services update-traffic job-coordinator --to-revisions v1=100
```

## Kubernetes Deployment

### Create Deployment

Create `k8s/deployment.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: job-coordinator
  labels:
    app: job-coordinator
spec:
  replicas: 1
  selector:
    matchLabels:
      app: job-coordinator
  template:
    metadata:
      labels:
        app: job-coordinator
    spec:
      containers:
      - name: coordinator
        image: gcr.io/YOUR_PROJECT_ID/job-coordinator:latest
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: JOB_COORDINATOR_JOB_ID
          value: "k8s-job"
        - name: JOB_COORDINATOR_BASE_PATH
          value: "/data/bundles"
        - name: JOB_COORDINATOR_MEASURES_TO_RUN
          value: "/measures/measure1.json"
        - name: JOB_COORDINATOR_DISTRIBUTOR_TYPE
          value: "pubsub"
        - name: JOB_COORDINATOR_DISTRIBUTOR_CONFIG
          value: '{"project_id":"YOUR_PROJECT_ID","topic":"work-units"}'
        - name: JOB_COORDINATOR_AUTO_START
          value: "false"
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
        volumeMounts:
        - name: bundles
          mountPath: /data/bundles
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: bundles
        persistentVolumeClaim:
          claimName: bundles-pvc
```

### Create Service

Create `k8s/service.yaml`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: job-coordinator
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
    name: http
  selector:
    app: job-coordinator
```

### Deploy to Kubernetes

```bash
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

### Monitor Deployment

```bash
# Check pod status
kubectl get pods -l app=job-coordinator

# View logs
kubectl logs -l app=job-coordinator -f

# Check service
kubectl get svc job-coordinator
```

## Binary Deployment

### Build Binary

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -ldflags "-s -w" \
  -o coordinator \
  ./cmd/coordinator
```

### Create Systemd Service

Create `/etc/systemd/system/job-coordinator.service`:

```ini
[Unit]
Description=Job Coordinator
After=network.target

[Service]
Type=simple
User=coordinator
WorkingDirectory=/opt/job-coordinator
ExecStart=/opt/job-coordinator/coordinator \
  --job-id=production-job \
  --base-path=/data/bundles \
  --measures-to-run=/measures/measure1.json \
  --distributor-type=file \
  --distributor-config='{"output_dir":"/data/output"}' \
  --auto-start=true
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
```

### Install and Start Service

```bash
# Create user
sudo useradd -r -s /bin/false coordinator

# Install binary
sudo mkdir -p /opt/job-coordinator
sudo cp coordinator /opt/job-coordinator/
sudo chown -R coordinator:coordinator /opt/job-coordinator

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable job-coordinator
sudo systemctl start job-coordinator

# Check status
sudo systemctl status job-coordinator
```

## Environment Variables

All CLI flags can be set via environment variables with `JOB_COORDINATOR_` prefix:

```bash
export JOB_COORDINATOR_JOB_ID=my-job
export JOB_COORDINATOR_BASE_PATH=/data/bundles
export JOB_COORDINATOR_MEASURES_TO_RUN=/measures/measure1.json
export JOB_COORDINATOR_DISTRIBUTOR_TYPE=stdout
export JOB_COORDINATOR_AUTO_START=true
```

See [configuration.md](configuration.md) for full list.

## Production Considerations

### Security

1. **Authentication**: Add authentication middleware or deploy behind authenticated gateway
2. **Secrets Management**: Use Secret Manager (GCP) or Secrets (K8s) for sensitive config
3. **Network Security**: Deploy in private VPC with firewall rules
4. **Least Privilege**: Use service accounts with minimal IAM permissions

### Monitoring

1. **Logging**: Aggregate logs with Cloud Logging or ELK stack
2. **Metrics**: Export metrics to Cloud Monitoring or Prometheus
3. **Alerting**: Set up alerts for job failures and API errors
4. **Tracing**: Add distributed tracing with OpenTelemetry

### Performance

1. **Concurrency**: Tune `--concurrent-file-processors` based on CPU/memory
2. **Batch Size**: Adjust `--batch-size` for optimal work unit size
3. **Resource Limits**: Set appropriate memory/CPU limits
4. **Autoscaling**: Configure horizontal pod autoscaling (HPA) in Kubernetes

### High Availability

1. **Multiple Replicas**: Run multiple instances behind load balancer
2. **Health Checks**: Configure health checks with `/health` endpoint
3. **Graceful Shutdown**: Service handles SIGTERM gracefully (TTL-based)
4. **State Management**: Job state is in-memory (consider external state store for HA)

### Cost Optimization

1. **Cloud Run**: Set `--min-instances=0` for scale-to-zero
2. **Kubernetes**: Use cluster autoscaling and node pools
3. **Spot Instances**: Use preemptible/spot VMs for non-critical workloads
4. **Resource Requests**: Set accurate resource requests to avoid over-provisioning

## Terraform Example

Create `main.tf` for Cloud Run deployment:

```hcl
resource "google_cloud_run_service" "coordinator" {
  name     = "job-coordinator"
  location = "us-central1"

  template {
    spec {
      containers {
        image = "gcr.io/YOUR_PROJECT_ID/job-coordinator"
        ports {
          container_port = 8080
        }
        env {
          name  = "JOB_COORDINATOR_JOB_ID"
          value = "terraform-job"
        }
        env {
          name  = "JOB_COORDINATOR_BASE_PATH"
          value = "/data/bundles"
        }
        env {
          name  = "JOB_COORDINATOR_MEASURES_TO_RUN"
          value = "/measures/measure1.json"
        }
        env {
          name  = "JOB_COORDINATOR_DISTRIBUTOR_TYPE"
          value = "pubsub"
        }
        env {
          name  = "JOB_COORDINATOR_DISTRIBUTOR_CONFIG"
          value = jsonencode({
            project_id = var.project_id
            topic      = "work-units"
          })
        }
        resources {
          limits = {
            memory = "2Gi"
            cpu    = "2"
          }
        }
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}

resource "google_cloud_run_service_iam_member" "public" {
  service  = google_cloud_run_service.coordinator.name
  location = google_cloud_run_service.coordinator.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}
```

Apply with Terraform:

```bash
terraform init
terraform plan
terraform apply
```

## Troubleshooting

### Container Won't Start

- Check logs: `docker logs <container_id>`
- Verify mounts: Ensure volume paths exist
- Check environment variables: Verify configuration is valid

### Cloud Run Deployment Fails

- Check service account permissions
- Verify Cloud Storage bucket access
- Check memory/CPU limits
- Review Cloud Build logs

### High Memory Usage

- Reduce `--concurrent-file-processors`
- Decrease `--batch-size`
- Monitor with profiling tools
- Check for memory leaks

### Job Not Processing

- Check `/job/status` endpoint
- Verify `--auto-start=true` or call `/job/start`
- Check distributor configuration
- Review logs for errors

## Resources

- [Dockerfile](../Dockerfile)
- [Configuration Guide](configuration.md)
- [API Documentation](api.md)
- [Development Guide](development.md)
