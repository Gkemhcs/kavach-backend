# GitHub Actions Workflows for Kavach Backend

This directory contains comprehensive GitHub Actions workflows for the Kavach Backend project, implementing modern CI/CD practices with Google Cloud Platform integration using Workload Identity.

## 📋 Workflows Overview

### 1. **Infrastructure Automation** (`infrastructure.yml`)
Automatically manages infrastructure using Terraform when changes are detected in the `infra/` directory.

### 2. **CI/CD Pipeline** (`ci-cd.yml`)
Builds Docker images, runs tests, and deploys to Cloud Run with comprehensive testing and monitoring.

## 🏗️ Infrastructure Automation

### Features
- **Path-based triggers**: Only runs when `infra/` directory changes
- **Multi-environment support**: Staging and production environments
- **Terraform integration**: Full Terraform workflow with plan/apply
- **Security scanning**: Automated security checks post-deployment
- **PR comments**: Automatic plan summaries in pull requests
- **Workload Identity**: Secure authentication without service account keys

### Triggers
- **Push to main/develop**: Automatic apply for infrastructure changes
- **Pull requests**: Plan-only mode with detailed comments
- **Manual dispatch**: Manual triggering with environment selection

### Workflow Steps
1. **Terraform Plan**: Validates and plans infrastructure changes
2. **Terraform Apply**: Applies changes to the selected environment
3. **Security Scan**: Runs post-deployment security checks
4. **Health Check**: Verifies deployed services are healthy
5. **Notification**: Provides deployment summary and links

### Outputs
- **Deployment URLs**: Cloud Run service URLs
- **Database connections**: PostgreSQL connection information
- **Security status**: Security scan results
- **Health status**: Service health check results

## 🚀 CI/CD Pipeline

### Features
- **Multi-stage deployment**: Staging → Production pipeline
- **Docker optimization**: Multi-platform builds with caching
- **Security scanning**: Trivy vulnerability scanning
- **Comprehensive testing**: Unit, integration, and smoke tests
- **Health monitoring**: Automated health checks and load testing
- **Rollback capability**: Automatic issue creation on failures

### Triggers
- **Push to main**: Deploys to staging, then production
- **Push to develop**: Deploys to staging only
- **Pull requests**: Runs tests and builds (no deployment)
- **Manual dispatch**: Manual deployment with environment selection

### Workflow Steps
1. **Test**: Runs linting, unit tests, and integration tests
2. **Build**: Builds multi-platform Docker images
3. **Scan**: Performs vulnerability scanning
4. **Deploy Staging**: Deploys to staging environment
5. **Deploy Production**: Deploys to production (if staging succeeds)
6. **Notify**: Provides deployment summary and creates issues on failures

### Environments
- **Staging**: Lower resource limits, for testing
- **Production**: Higher resource limits, with warm instances

## 🔐 Security Features

### Workload Identity
Both workflows use Google Cloud Workload Identity for secure authentication:
- No service account keys stored in secrets
- Short-lived credentials
- Principle of least privilege

### Security Scanning
- **Trivy**: Container vulnerability scanning
- **Infrastructure security**: Checks for public resources
- **Firewall validation**: Ensures proper network security

### Secrets Management
- **Secret Manager**: Environment variables and secrets
- **Encrypted state**: Terraform state encryption
- **Secure outputs**: Sensitive data protection

## 📊 Monitoring and Observability

### Health Checks
- **Application health**: `/healthz` endpoint monitoring
- **Readiness checks**: Service readiness validation
- **Load testing**: Basic load testing for production

### Logging
- **Structured logging**: JSON format logs
- **Cloud Logging**: Centralized log management
- **Error tracking**: Automatic error reporting

### Metrics
- **Deployment metrics**: Success/failure rates
- **Performance metrics**: Response times and throughput
- **Resource utilization**: CPU, memory, and network usage

## 🛠️ Setup Instructions

### Prerequisites
1. **Google Cloud Project**: Set up with required APIs enabled
2. **Workload Identity**: Configure Workload Identity Federation
3. **Service Account**: Create with appropriate permissions
4. **Artifact Registry**: Set up for Docker images
5. **Cloud Run**: Configure for deployments

### Required Secrets
```yaml
# Google Cloud Configuration
GOOGLE_CLOUD_PROJECT: "your-project-id"
WIF_PROVIDER: "projects/123456789/locations/global/workloadIdentityPools/github-actions/providers/github"
GCP_SA_EMAIL: "github-actions@your-project.iam.gserviceaccount.com"

# Terraform Configuration
TF_STATE_BUCKET: "your-terraform-state-bucket"
TF_STATE_ENCRYPTION_KEY: "your-encryption-key"
```

### Environment Setup
1. **Staging Environment**: Configure in GitHub repository settings
2. **Production Environment**: Configure with protection rules
3. **Required reviewers**: Set up for production deployments

## 🔄 Workflow Lifecycle

### Infrastructure Changes
1. **Developer pushes** changes to `infra/` directory
2. **Workflow triggers** automatically
3. **Terraform plan** generates and comments on PR
4. **On merge**: Terraform applies changes
5. **Security scan** validates deployment
6. **Health check** confirms services are running

### Application Changes
1. **Developer pushes** code changes
2. **Tests run** automatically (linting, unit tests)
3. **Docker image** builds and pushes to Artifact Registry
4. **Staging deployment** occurs automatically
5. **Production deployment** follows (if staging succeeds)
6. **Health checks** and smoke tests run
7. **Notification** provides deployment summary

## 📈 Production Considerations

### High Availability
- **Multi-region deployment**: Consider cross-region deployment
- **Load balancing**: Global load balancer configuration
- **Auto-scaling**: Proper scaling policies

### Disaster Recovery
- **Backup strategies**: Database and application backups
- **Rollback procedures**: Quick rollback mechanisms
- **Monitoring alerts**: Proactive issue detection

### Performance
- **Resource optimization**: Right-sizing Cloud Run services
- **Caching strategies**: CDN and application caching
- **Database optimization**: Connection pooling and indexing

## 🚨 Troubleshooting

### Common Issues
1. **Authentication failures**: Check Workload Identity configuration
2. **Build failures**: Verify Dockerfile and dependencies
3. **Deployment failures**: Check Cloud Run configuration
4. **Health check failures**: Verify application endpoints

### Debugging Steps
1. **Check workflow logs**: Detailed execution logs
2. **Verify secrets**: Ensure all required secrets are set
3. **Test locally**: Run commands locally to verify
4. **Check permissions**: Verify service account permissions

### Support
- **Documentation**: Check this README and inline comments
- **Logs**: Review Cloud Logging for detailed information
- **Issues**: Create GitHub issues for workflow problems

## 🔮 Future Enhancements

### Planned Features
- **Blue-green deployments**: Zero-downtime deployments
- **Canary releases**: Gradual rollout capabilities
- **Performance testing**: Automated performance benchmarks
- **Cost optimization**: Resource usage monitoring and optimization

### Monitoring Improvements
- **Custom metrics**: Application-specific metrics
- **Alerting**: Proactive alert configuration
- **Dashboards**: Deployment and performance dashboards
- **SLA monitoring**: Service level agreement tracking

---

## 📝 Notes

- All workflows use Workload Identity for secure authentication
- Infrastructure changes are automatically applied on main branch
- Production deployments require staging to succeed first
- Comprehensive logging and monitoring are built-in
- Security scanning runs automatically on all deployments

For questions or issues, please create a GitHub issue or contact the development team. 