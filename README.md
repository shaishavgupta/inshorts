# Contextual News Data Retrieval System

A Go-based backend application that processes natural language queries using Large Language Models (LLMs) to retrieve and enrich news articles from a PostgreSQL database with PostGIS and pgvector extensions.

## Features

- **Natural Language Query Processing**: Uses LLMs to understand user queries and extract entities and intents
- **Multi-Intent Filter Chain**: Supports complex queries with multiple retrieval strategies
- **Spatial Queries**: Location-based news retrieval using PostGIS
- **Vector Search**: Semantic text search using pgvector
- **Trending News**: Compute trending articles based on user engagement and location
- **Article Enrichment**: LLM-generated summaries for each article
- **RESTful API**: Clean API design with proper error handling
- **Dockerized Deployment**: Easy deployment with Docker and Docker Compose

## Architecture

The application follows a three-layer architecture:

- **Controller Layer**: Handles HTTP requests and responses
- **Service Layer**: Implements business logic and orchestrates operations
- **Repository Layer**: Manages database interactions

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (for containerized deployment)
- PostgreSQL 16 with PostGIS and pgvector extensions (if running locally)
- OpenAI API key or compatible LLM API

## Environment Variables

The application is configured using environment variables. Copy `.env.example` to `.env` and update the values:

### Database Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_URL` | PostgreSQL connection string | - | Yes |
| `DB_MAX_OPEN_CONNS` | Maximum number of open connections to the database | `25` | No |
| `DB_MAX_IDLE_CONNS` | Maximum number of idle connections in the pool | `5` | No |
| `DB_CONN_MAX_LIFETIME` | Maximum lifetime of a connection (e.g., `5m`, `1h`) | `5m` | No |
| `DB_CONN_MAX_IDLE_TIME` | Maximum idle time of a connection (e.g., `5m`, `1h`) | `5m` | No |

**Example DATABASE_URL formats:**
```
postgres://username:password@localhost:5432/dbname?sslmode=disable
postgres://username:password@host:5432/dbname?sslmode=require
```

### Server Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PORT` | HTTP server port | `8080` | No |
| `SERVER_READ_TIMEOUT` | Maximum duration for reading the entire request (e.g., `10s`, `30s`) | `10s` | No |
| `SERVER_WRITE_TIMEOUT` | Maximum duration before timing out writes of the response (e.g., `10s`, `30s`) | `10s` | No |

### LLM API Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LLM_API_KEY` | API key for the LLM service (e.g., OpenAI API key) | - | Yes |
| `LLM_API_URL` | Base URL for the LLM API | `https://api.openai.com/v1` | No |

**Supported LLM Providers:**
- OpenAI (default): `https://api.openai.com/v1`
- Azure OpenAI: `https://<resource-name>.openai.azure.com`
- Other OpenAI-compatible APIs

### Cache Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CACHE_TTL` | Time-to-live for cached trending results (e.g., `5m`, `10m`, `1h`) | `5m` | No |

### Logging Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LOG_LEVEL` | Logging level: `debug`, `info`, `warn`, or `error` | `info` | No |

**Log Levels:**
- `debug`: Detailed information for debugging
- `info`: General informational messages
- `warn`: Warning messages for potentially harmful situations
- `error`: Error messages for serious problems

## Quick Start

### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone <repository-url>
cd contextual-news-retrieval-system
```

2. Create a `.env` file from the example:
```bash
cp .env.example .env
```

3. Update the `.env` file with your LLM API key:
```bash
LLM_API_KEY=your-actual-api-key-here
```

4. Start the application:
```bash
docker-compose up -d
```

5. Verify the application is running:
```bash
curl http://localhost:8080/health
```

### Local Development

1. Install dependencies:
```bash
go mod download
```

2. Set up PostgreSQL with PostGIS and pgvector extensions:
```sql
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS vector;
```

3. Run the database initialization script:
```bash
psql -U newsuser -d newsdb -f init.sql
```

4. Set environment variables:
```bash
export DATABASE_URL="postgres://newsuser:newspass@localhost:5432/newsdb?sslmode=disable"
export LLM_API_KEY="your-api-key-here"
export LOG_LEVEL="debug"
```

5. Run the application:
```bash
go run cmd/api/main.go
```

## Loading News Data

Before you can query news articles, you need to load data into the database. The system provides two methods for loading news articles from JSON files.

### JSON File Format

Your JSON file should contain an array of article objects with the following structure:

```json
[
  {
    "id": "optional-uuid",
    "title": "Article Title",
    "description": "Article description text",
    "url": "https://example.com/article",
    "publication_date": "2024-04-28T10:00:00Z",
    "source_name": "News Source",
    "category": ["Technology", "Business"],
    "relevance_score": 0.85,
    "latitude": 37.7749,
    "longitude": -122.4194
  }
]
```

**Field Requirements:**
- `id`: Optional UUID. If not provided, one will be generated automatically
- `title`: Required. Article headline
- `description`: Optional. Article summary or excerpt
- `url`: Required. Valid URL to the full article
- `publication_date`: Required. ISO 8601 timestamp
- `source_name`: Required. Name of the news source
- `category`: Required. Array of category strings (at least one)
- `relevance_score`: Required. Float between 0 and 1
- `latitude`: Required. Float between -90 and 90
- `longitude`: Required. Float between -180 and 180

### Method 1: CLI Data Loader (Recommended)

The CLI loader is a standalone command-line tool for loading data efficiently.

**Build the loader:**
```bash
# Using Make (recommended)
make build-loader

# Or manually
go build -o loader cmd/loader/main.go
```

**Run the loader:**
```bash
./loader -file path/to/articles.json
```

**Load sample data:**
```bash
# Ensure DATABASE_URL and LLM_API_KEY are set in your environment
make load-data
```

**With Docker:**
```bash
# Copy your JSON file into the container
docker cp articles.json contextual-news-api:/tmp/articles.json

# Run the loader inside the container
docker exec contextual-news-api ./loader -file /tmp/articles.json
```

**Features:**
- Progress logging every 100 articles
- Detailed statistics on success/failure
- Batch insert for performance
- Automatic UUID generation for articles without IDs
- Handles duplicate IDs with upsert logic

### Method 2: API Endpoint

You can also load data via the REST API endpoint.

```http
POST /api/v1/news/load
Content-Type: application/json

{
  "filepath": "/path/to/articles.json"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Data loaded successfully"
}
```

**Note:** The filepath must be accessible from the server's filesystem. This method is useful for automated workflows or when the JSON file is already on the server.

**Example with curl:**
```bash
curl -X POST http://localhost:8080/api/v1/news/load \
  -H "Content-Type: application/json" \
  -d '{"filepath": "/data/articles.json"}'
```

### Loading Progress and Statistics

Both methods provide detailed logging:

- **Progress Updates**: Every 100 articles loaded
- **Success Count**: Number of articles successfully inserted
- **Error Count**: Number of articles that failed to insert
- **Total Duration**: Time taken to complete the load

Check the application logs for detailed statistics:
```bash
# Docker logs
docker logs contextual-news-api

# Local logs (if running directly)
# Logs are output to stdout
```

## API Endpoints

### Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "service": "contextual-news-api"
}
```

### Query News

Process a natural language query and retrieve relevant news articles.

```http
POST /api/v1/news/query
Content-Type: application/json

{
  "query": "Latest technology news about AI near San Francisco",
  "location": {
    "latitude": 37.7749,
    "longitude": -122.4194
  }
}
```

**Response:**
```json
{
  "articles": [
    {
      "id": "uuid",
      "title": "Article Title",
      "description": "Article description...",
      "url": "https://example.com/article",
      "publication_date": "2024-04-28T10:00:00Z",
      "source_name": "Tech News",
      "category": ["Technology"],
      "relevance_score": 0.92,
      "latitude": 37.7749,
      "longitude": -122.4194,
      "llm_summary": "This article discusses...",
      "distance": 2.5
    }
  ]
}
```

### Get Trending News

Retrieve trending news articles based on location and user engagement.

```http
GET /api/v1/news/trending?lat=37.7749&lon=-122.4194&limit=10
```

**Query Parameters:**
- `lat` (required): Latitude
- `lon` (required): Longitude
- `limit` (optional): Number of articles to return (default: 10)

**Response:**
```json
{
  "articles": [
    {
      "id": "uuid",
      "title": "Trending Article",
      "description": "Article description...",
      "url": "https://example.com/article",
      "publication_date": "2024-04-28T10:00:00Z",
      "source_name": "News Source",
      "category": ["Technology"],
      "relevance_score": 0.88,
      "latitude": 37.7749,
      "longitude": -122.4194,
      "trending_score": 0.95
    }
  ]
}
```

### Record User Interaction

Record a user interaction event with an article.

```http
POST /api/v1/interactions
Content-Type: application/json

{
  "user_id": "user123",
  "article_id": "article-uuid",
  "event_type": "view",
  "location": {
    "latitude": 37.7749,
    "longitude": -122.4194
  }
}
```

**Event Types:**
- `view`: User viewed the article
- `click`: User clicked on the article

**Response:**
```json
{
  "success": true,
  "event_id": "event-uuid"
}
```

## Query Examples

### Category-based Query
```json
{
  "query": "Show me sports news"
}
```

### Location-based Query
```json
{
  "query": "News near me",
  "location": {
    "latitude": 37.7749,
    "longitude": -122.4194
  }
}
```

### Source-based Query
```json
{
  "query": "Latest articles from Reuters"
}
```

### Complex Multi-intent Query
```json
{
  "query": "Technology news about artificial intelligence from New York Times near San Francisco",
  "location": {
    "latitude": 37.7749,
    "longitude": -122.4194
  }
}
```

## Error Handling

The API returns appropriate HTTP status codes and error messages:

| Status Code | Description |
|-------------|-------------|
| 200 | Success |
| 400 | Bad Request - Invalid input parameters |
| 404 | Not Found - Resource not found |
| 500 | Internal Server Error |
| 503 | Service Unavailable - LLM or database unavailable |

**Error Response Format:**
```json
{
  "error": "Descriptive error message"
}
```

## Project Structure

```
.
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── src/
│   ├── config/
│   │   ├── config.go            # Configuration management
│   │   └── database.go          # Database initialization
│   ├── controllers/
│   │   ├── news_controller.go
│   │   └── user_interaction_controller.go
│   ├── middleware/
│   │   └── error_handler.go
│   ├── models/
│   │   └── models.go            # Data models
│   ├── repositories/
│   │   ├── article_repository.go
│   │   └── user_event_repository.go
│   └── services/
│       ├── filter_chain.go
│       ├── filters.go
│       ├── llm_service.go
│       ├── news_service.go
│       └── trending_service.go
├── pkg/
│   └── logger/
│       └── logger.go            # Singleton logger
├── .env.example                 # Example environment variables
├── docker-compose.yml           # Docker Compose configuration
├── Dockerfile                   # Docker image definition
├── go.mod                       # Go module definition
├── init.sql                     # Database initialization script
└── README.md                    # This file
```

## Development

### Using Make

The project includes a Makefile for common tasks:

```bash
# Show all available commands
make help

# Build both API and loader
make build

# Build only the API
make build-api

# Build only the loader
make build-loader

# Run the API server
make run

# Run tests
make test

# Load sample data
make load-data

# Clean build artifacts
make clean
```

### Running Tests

```bash
make test
# Or
go test ./...
```

### Building the Application

```bash
# Using Make
make build-api

# Or manually
go build -o api cmd/api/main.go
```

### Running with Custom Configuration

```bash
export DATABASE_URL="postgres://user:pass@localhost:5432/newsdb"
export LLM_API_KEY="your-key"
export LOG_LEVEL="debug"
./news-api
```

## Deployment

### Docker

Build and run the Docker image:

```bash
docker build -t contextual-news-api .
docker run -p 8080:8080 \
  -e DATABASE_URL="postgres://user:pass@db:5432/newsdb" \
  -e LLM_API_KEY="your-key" \
  contextual-news-api
```

### Docker Compose

The recommended deployment method:

```bash
docker-compose up -d
```

This will start:
- PostgreSQL with PostGIS and pgvector extensions
- The API service

### Production Considerations

1. **Security:**
   - Use strong database passwords
   - Enable SSL/TLS for database connections (`sslmode=require`)
   - Secure your LLM API keys
   - Configure CORS appropriately for your domain

2. **Performance:**
   - Adjust database connection pool settings based on load
   - Configure appropriate cache TTL
   - Monitor and optimize database queries
   - Consider using a reverse proxy (nginx, Caddy)

3. **Monitoring:**
   - Set `LOG_LEVEL=info` or `LOG_LEVEL=warn` in production
   - Implement log aggregation (e.g., ELK stack, Loki)
   - Monitor database performance
   - Track LLM API usage and costs

4. **Scaling:**
   - Use a managed PostgreSQL service (AWS RDS, Google Cloud SQL)
   - Consider Redis for distributed caching
   - Deploy multiple API instances behind a load balancer

## Troubleshooting

### Database Connection Issues

**Error:** `failed to connect to database`

**Solution:**
- Verify `DATABASE_URL` is correct
- Ensure PostgreSQL is running
- Check network connectivity
- Verify database credentials

### LLM API Issues

**Error:** `LLM service unavailable`

**Solution:**
- Verify `LLM_API_KEY` is valid
- Check `LLM_API_URL` is correct
- Ensure you have API credits/quota
- Check network connectivity to LLM provider

### Extension Issues

**Error:** `extension "postgis" does not exist`

**Solution:**
```sql
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS vector;
```

Or use the provided `init.sql` script.

## License

[Your License Here]

## Contributing

[Your Contributing Guidelines Here]

## Support

For issues and questions, please open an issue on the repository.
