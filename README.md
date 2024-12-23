# ContainerEye

ContainerEye is a powerful container monitoring and alerting tool that provides real-time insights into your Docker containers' performance and health.

## Features

- **Real-time Monitoring**: Track CPU, memory, network, and disk I/O metrics in real-time
- **Historical Data**: Store and analyze historical performance data
- **Smart Alerting**: Configure flexible alert rules based on various metrics
- **Multiple Notification Channels**: Receive alerts via Slack, Email, or Webhooks
- **REST API**: Integrate with your existing tools and dashboards
- **Command-line Interface**: Manage and monitor containers from the terminal

## Installation

### Prerequisites

- Go 1.21 or later
- Docker Engine
- SQLite3

### Building from Source

```bash
# Clone the repository
git clone https://github.com/haitaoxw/ContainerEye.git
cd ContainerEye

# Build the server
go build -o bin/containereye-server ./cmd/main.go

# Build the CLI
go build -o bin/containereye ./cmd/cli/main.go
```

## Configuration

Create a `config.yaml` file in the project root directory:

```yaml
database:
  path: "data/containereye.db"

alert:
  slack:
    token: "your-slack-token"
    channel: "monitoring"
  email:
    smtp_host: "smtp.gmail.com"
    smtp_port: 587
    from: "alerts@yourdomain.com"
    password: "your-email-password"
    to_receivers:
      - "admin@yourdomain.com"

server:
  port: 8080
```

## Usage

### Starting the Server

```bash
./bin/containereye-server
```

### Using the CLI

1. List Containers:
```bash
containereye container list
```

2. View Container Stats:
```bash
# Show real-time stats
containereye stats show <container_id> --watch

# View historical stats
containereye stats history <container_id> --from "2024-01-01T00:00:00Z" --to "2024-01-02T00:00:00Z"

# Export stats to CSV
containereye stats export <container_id> --format csv --output stats.csv
```

3. Managing Alerts:
```bash
# List all alerts
containereye alert list

# List critical alerts
containereye alert list --level critical

# Acknowledge an alert
containereye alert acknowledge <alert_id> --comment "Investigating"

# Resolve an alert
containereye alert resolve <alert_id> --comment "Fixed"
```

### Using the API

The server exposes a REST API that can be accessed using the following endpoints:

1. Containers:
- `GET /api/v1/containers`: List all containers
- `GET /api/v1/containers/{id}`: Get container details
- `GET /api/v1/containers/{id}/stats`: Get container statistics

2. Alerts:
- `GET /api/v1/alerts`: List alerts
- `POST /api/v1/alerts/{id}/acknowledge`: Acknowledge an alert
- `POST /api/v1/alerts/{id}/resolve`: Resolve an alert

All API requests require an API key in the `X-API-Key` header.

## Development

### Project Structure

```
containereye/
├── cmd/
│   ├── main.go           # Server entry point
│   └── cli/
│       └── main.go       # CLI entry point
├── internal/
│   ├── alert/           # Alert management
│   ├── api/             # HTTP API
│   ├── auth/            # Authentication
│   ├── cli/             # CLI commands
│   ├── config/          # Configuration
│   ├── database/        # Database operations
│   ├── models/          # Data models
│   └── monitor/         # Container monitoring
├── templates/           # Email templates
├── config.example.yaml  # Example configuration
└── README.md
```

### Running Tests

```bash
go test ./...
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Docker Engine API
- Gin Web Framework
- GORM
- Cobra CLI Framework
