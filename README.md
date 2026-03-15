# Note-Stacks Backend

A RESTful API backend for the Note-Stacks application built with Go 1.21+ and Redis 7.2+.

## Quick Start

### Prerequisites
- Docker and Docker Compose installed on your system

### Running the Application

1. Navigate to the `back` directory:
```bash
cd back
```

2. Start all services (Go app + Redis):
```bash
docker-compose up -d
```

3. Check if services are running:
```bash
docker-compose ps
```

4. View logs:
```bash
docker-compose logs -f
```

5. Stop services:
```bash
docker-compose down
```

### Rebuilding After Code Changes

```bash
docker-compose up -d --build
```

## API Endpoints

Base URL: `http://localhost:8080`

### Health Check
```bash
GET /health
```

### Notes API

#### Create a Note
```bash
POST /api/notes
Content-Type: application/json

{
  "title": "My Note",
  "content": "Note content here",
  "stackId": "stack-1"
}
```

#### Get a Note
```bash
GET /api/notes/{id}
```

#### Update a Note
```bash
PUT /api/notes/{id}
Content-Type: application/json

{
  "title": "Updated Title",
  "content": "Updated content",
  "stackId": "stack-2"
}
```

#### Delete a Note
```bash
DELETE /api/notes/{id}
```

#### Get All Notes in a Stack
```bash
GET /api/stacks/{stackId}/notes
```

## Testing with curl

### Create a note:
```bash
curl -X POST http://localhost:8080/api/notes \
  -H "Content-Type: application/json" \
  -d '{"title":"Test Note","content":"Hello World","stackId":"stack-1"}'
```

### Get a note (replace {id} with actual note ID):
```bash
curl http://localhost:8080/api/notes/{id}
```

### Update a note:
```bash
curl -X PUT http://localhost:8080/api/notes/{id} \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated Title"}'
```

### Delete a note:
```bash
curl -X DELETE http://localhost:8080/api/notes/{id}
```

### Get notes in a stack:
```bash
curl http://localhost:8080/api/stacks/stack-1/notes
```

## Project Structure

```
back/
├── config/              # Configuration management
├── models/              # Data models
├── handlers/            # HTTP handlers
├── repository/          # Data access layer
├── middleware/          # HTTP middleware
├── utils/               # Utility functions
├── main.go              # Application entry point
├── Dockerfile           # Docker configuration
├── docker-compose.yml   # Multi-container setup
├── BLUEPRINT.md         # Architecture documentation
└── README.md            # This file
```

## Environment Variables

You can customize the following environment variables in `docker-compose.yml`:

- `REDIS_HOST`: Redis hostname (default: "redis")
- `REDIS_PORT`: Redis port (default: "6379")
- `REDIS_PASSWORD`: Redis password (default: "")
- `SERVER_PORT`: Application port (default: "8080")

## Troubleshooting

### Check if Redis is running:
```bash
docker-compose exec redis redis-cli ping
```

### Access Redis CLI:
```bash
docker-compose exec redis redis-cli
```

### View application logs:
```bash
docker-compose logs app
```

### Restart services:
```bash
docker-compose restart
```

## Development

For detailed architecture information, see [BLUEPRINT.md](../.claude/BLUEPRINT.md).

### Local Development Without Docker

1. Install Go 1.21+
2. Install Redis 7.2+
3. Set environment variables:
```bash
export REDIS_HOST=localhost
export REDIS_PORT=6379
export SERVER_PORT=8080
```
4. Run the application:
```bash
go run main.go
```

## License

MIT
