# Kavach Backend

A robust, enterprise-grade secrets management backend built with Go, featuring advanced RBAC, multi-provider support, and comprehensive audit logging.

## 🚀 Features

### 🔐 **Secrets Management**
- **Versioned Secret Storage**: Full versioning support with commit messages and rollback capabilities
- **Environment-based Organization**: Organize secrets by environments (dev, staging, prod)
- **Secret Groups**: Logical grouping of related secrets within organizations
- **Encryption at Rest**: AES-256 encryption for all secret values
- **Diff Tracking**: Compare secret versions to see what changed between deployments

### 🛡️ **Identity & Access Management (IAM)**
- **Role-Based Access Control (RBAC)**: Fine-grained permissions with owner, admin, editor, and viewer roles
- **Multi-level Authorization**: Organization, secret group, and environment-level permissions
- **User Groups**: Manage permissions for groups of users
- **Dynamic Role Bindings**: Grant and revoke access with comprehensive audit trails
- **Casbin Integration**: Advanced policy enforcement engine

### 🔌 **Multi-Provider Support**
- **GitHub Secrets**: Sync secrets to GitHub repositories and environments
- **Google Cloud Platform**: Integration with GCP Secret Manager
- **Azure Key Vault**: Sync secrets to Azure Key Vault

- **Provider Credentials Management**: Secure storage of provider API keys and configurations

### 🔐 **Authentication & Security**
- **OAuth 2.0 Integration**: GitHub OAuth for seamless authentication
- **JWT Tokens**: Secure access and refresh token management
- **Device Flow Support**: CLI-friendly authentication for headless environments
- **Session Management**: Automatic token refresh and validation

### 📊 **Audit & Compliance** *(In Development)*
- **Comprehensive Logging**: All operations logged with detailed context
- **Audit Trails**: Track who accessed what, when, and why *(Coming Soon)*
- **Change History**: Complete history of all secret modifications *(Coming Soon)*
- **Compliance Ready**: Built-in support for security and compliance requirements *(Coming Soon)*

### 🗄️ **Database & Performance**
- **PostgreSQL**: Production-ready database with advanced features
- **Connection Pooling**: Optimized database connection management
- **Migration System**: Version-controlled schema changes with migrate
- **SQLC Integration**: Type-safe database queries with generated Go code

## 🏗️ Architecture

### **Core Components**

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP Layer    │    │  Business Logic │    │   Data Layer    │
│   (Gin Router)  │◄──►│   (Services)    │◄──►│  (PostgreSQL)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Middleware     │    │   Providers     │    │   Migrations    │
│  (Auth, RBAC)   │    │ (GitHub, GCP,   │    │   (Schema Mgmt) │
└─────────────────┘    │   Azure)        │    └─────────────────┘
```

### **Service Layer Architecture**
- **Secret Service**: Manages secret CRUD operations and versioning
- **Provider Service**: Handles external provider integrations
- **IAM Service**: Manages roles, permissions, and access control
- **Auth Service**: Handles authentication and user management
- **Organization Service**: Manages organizational structure

### **Data Flow**
1. **Request Processing**: HTTP requests are validated and authenticated
2. **Authorization Check**: RBAC policies are evaluated using Casbin
3. **Business Logic**: Services execute business operations
4. **Data Persistence**: Changes are persisted to PostgreSQL
5. **Audit Logging**: All operations are logged for compliance
6. **Response**: Results are returned to the client

## 🛠️ Technology Stack

### **Core Technologies**
- **Go 1.23**: High-performance, compiled language
- **Gin**: Fast HTTP web framework
- **PostgreSQL 15**: Advanced open-source database
- **SQLC**: Type-safe SQL code generation
- **Migrate**: Database schema migration tool
- **go-github/v74**: Official GitHub API client for Go

### **Security & Authentication**
- **JWT**: JSON Web Token implementation
- **OAuth 2.0**: Industry-standard authorization protocol
- **Casbin**: Authorization library with RBAC support
- **pgcrypto**: PostgreSQL cryptographic functions

### **Cloud Provider Integrations**
- **GitHub API v74**: Repository and environment secrets using go-github/v74
- **Google Cloud SDK**: Secret Manager integration
- **Azure SDK**: Key Vault integration

### **Development & Testing**
- **Testify**: Testing framework with mocks
- **Docker**: Containerization and development environment
- **Docker Compose**: Multi-service development setup

## 📁 Project Structure

```
backend/
├── cmd/                    # Application entry points
│   └── server/            # Main server binary
├── internal/               # Private application code
│   ├── auth/              # Authentication and JWT management
│   ├── authz/             # Authorization and RBAC
│   ├── config/            # Configuration management
│   ├── db/                # Database connection and migrations
│   ├── errors/            # Custom error types
│   ├── groups/            # User group management
│   ├── iam/               # Identity and access management
│   ├── middleware/        # HTTP middleware components
│   ├── org/               # Organization management
│   ├── provider/          # External provider integrations
│   ├── secret/            # Secret management core
│   ├── secretgroup/       # Secret group operations
│   ├── server/            # HTTP server setup
│   ├── types/             # Shared type definitions
│   └── utils/             # Utility functions
├── infra/                 # Infrastructure configuration
├── migrations/            # Database schema migrations
├── Dockerfile             # Container image definition
├── docker-compose.yaml    # Development environment setup
└── sqlc.yaml             # SQL code generation config
```

## 🚀 Quick Start

### **Prerequisites**
- Go 1.23 or later
- Docker and Docker Compose
- PostgreSQL 15 (or use Docker)

### **Development Setup**

1. **Clone and Navigate**
   ```bash
   cd backend
   ```

2. **Start Development Environment**
   ```bash
   docker-compose up -d
   ```

3. **Verify Services**
   ```bash
   # Check if all services are running
   docker-compose ps
   
   # Check application health
   curl http://localhost:8080/healthz
   ```

4. **View Logs**
   ```bash
   docker-compose logs -f app
   ```

### **Environment Variables**

Create a `.env` file with the following configuration:

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=kavach_user
DB_PASSWORD=your_password
DB_NAME=kavach_db

# Server Configuration
PORT=8080
ENV=development

# JWT Configuration
JWT_SECRET=your_jwt_secret
JWT_ACCESS_TOKEN_SECRET=your_access_token_secret
JWT_REFRESH_TOKEN_SECRET=your_refresh_token_secret
ACCESS_TOKEN_DURATION=1000
REFRESH_TOKEN_DURATION=1440

# GitHub OAuth Configuration
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GITHUB_CALLBACK_URL=http://localhost:8080/api/v1/auth/github/callback

# Encryption Key
ENCRYPTION_KEY=your_32_byte_base64_encryption_key

# Connection Pooling (Optional)
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5
DB_CONN_MAX_IDLE_TIME=5
```

## 🔧 Configuration

### **Database Connection Pooling**

The application supports configurable database connection pooling for optimal performance:

```bash
# Production settings
DB_MAX_OPEN_CONNS=50      # Maximum open connections
DB_MAX_IDLE_CONNS=10      # Maximum idle connections
DB_CONN_MAX_LIFETIME=15   # Connection max lifetime (minutes)
DB_CONN_MAX_IDLE_TIME=10  # Connection max idle time (minutes)
```

See [CONNECTION_POOLING.md](./CONNECTION_POOLING.md) for detailed configuration options.

### **Provider Configuration**

Each cloud provider requires specific configuration:

#### **GitHub**
```json
{
  "provider": "github",
  "credentials": {
    "token": "ghp_your_github_token"
  },
  "config": {
    "owner": "your_org",
    "repository": "your_repo",
    "environment": "production",
    "secret_visibility": "selected"
  }
}
```

#### **Google Cloud Platform**
```json
{
  "provider": "gcp",
  "credentials": {
    "type": "service_account",
    "project_id": "your-project",
    "private_key_id": "key_id",
    "private_key": "-----BEGIN PRIVATE KEY-----...",
    "client_email": "service@project.iam.gserviceaccount.com"
  },
  "config": {
    "project_id": "your-project",
    "secret_manager_location": "us-central1"
  }
}
```

#### **Azure Key Vault**
```json
{
  "provider": "azure",
  "credentials": {
    "tenant_id": "your-tenant-id",
    "client_id": "your-client-id",
    "client_secret": "your-client-secret"
  },
  "config": {
    "subscription_id": "your-subscription-id",
    "resource_group": "your-resource-group",
    "key_vault_name": "your-key-vault"
  }
}
```

## 📚 API Reference

### **Authentication Endpoints**

- `POST /api/v1/auth/device/start` - Start OAuth device flow
- `POST /api/v1/auth/device/poll` - Poll for device flow completion
- `GET /api/v1/auth/github/login` - GitHub OAuth login
- `GET /api/v1/auth/github/callback` - GitHub OAuth callback
- `POST /api/v1/auth/refresh` - Refresh JWT tokens

### **Secrets Management**

- `POST /api/v1/secrets/versions` - Create new secret version
- `GET /api/v1/secrets/versions` - List secret versions
- `GET /api/v1/secrets/versions/{id}` - Get specific version
- `POST /api/v1/secrets/versions/{id}/rollback` - Rollback to version
- `GET /api/v1/secrets/versions/{id}/diff` - Compare versions

### **Provider Operations**

- `POST /api/v1/providers/credentials` - Add provider credentials
- `POST /api/v1/providers/sync` - Sync secrets to provider
- `GET /api/v1/providers/status` - Check provider status

### **IAM & Access Control**

- `POST /api/v1/iam/role-bindings` - Grant role to user/group
- `DELETE /api/v1/iam/role-bindings` - Revoke role
- `GET /api/v1/iam/permissions` - Check user permissions

## 🧪 Testing

### **Run Tests**
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/secret
```

### **Test Data**
The project includes comprehensive test data and mocks for all major components.

## 🐳 Docker

### **Build Image**
```bash
docker build -t kavach-backend .
```

### **Run Container**
```bash
docker run -p 8080:8080 \
  -e DB_HOST=your_db_host \
  -e DB_PASSWORD=your_password \
  kavach-backend
```

### **Development with Docker Compose**
```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Rebuild and restart
docker-compose up --build -d
```

## 📊 Monitoring & Health Checks

### **Health Endpoints**
- `GET /healthz` - Basic health check
- `GET /healthz/detailed` - Detailed health with database pool stats

### **Metrics**
The application provides comprehensive metrics for:
- Database connection pool status
- Request/response times
- Error rates
- Provider sync status

## 🔒 Security Features

### **Data Protection**
- **Encryption at Rest**: All secrets encrypted with AES-256
- **Secure Communication**: HTTPS/TLS for all external communications
- **Token Security**: JWT tokens with configurable expiration
- **Access Control**: Fine-grained RBAC with audit logging

### **Compliance** *(In Development)*
- **Audit Logging**: Complete audit trail for all operations *(Coming Soon)*
- **Data Retention**: Configurable data retention policies *(Coming Soon)*
- **Access Reviews**: Regular access review capabilities *(Coming Soon)*
- **Compliance Reporting**: Built-in compliance reporting *(Coming Soon)*

## 🚀 Deployment

### **Production Considerations**
- Use external PostgreSQL instance
- Configure proper SSL/TLS certificates
- Set up monitoring and alerting
- Implement backup and disaster recovery
- Use secrets management for sensitive configuration

### **Scaling**
- Horizontal scaling with load balancers
- Database read replicas for read-heavy workloads
- Connection pooling optimization
- Caching strategies for frequently accessed data

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

### **Development Guidelines**
- Follow Go best practices and conventions
- Write comprehensive tests
- Update documentation for new features
- Use conventional commit messages

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](./LICENSE) file for details.

## 🆘 Support

### **Documentation**
- [API Documentation](./docs/api.md)
- [Deployment Guide](./docs/deployment.md)
- [Troubleshooting](./docs/troubleshooting.md)

### **Issues**
- Report bugs via GitHub Issues
- Request features via GitHub Discussions
- Security issues: please report privately

### **Community**
- Join our Discord server
- Follow us on Twitter
- Star the repository if you find it useful

---

**Kavach Backend** - Secure, scalable, and enterprise-ready secrets management.