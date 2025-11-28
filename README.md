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

## Coding Patterns

This codebase implements several well-established design patterns and architectural principles:

### 1. **Layered Architecture (Separation of Concerns)**

The application is organized into distinct layers with clear responsibilities:

- **Controllers** (`src/controllers/`): Handle HTTP requests, validate input, and format responses
- **Services** (`src/services/`): Implement business logic and orchestrate operations
- **Repositories** (`src/repositories/`): Abstract database operations and data access
- **Models** (`src/models/`): Define domain entities and data structures
- **Types** (`src/types/`): Define request/response DTOs with validation

### 2. **Dependency Injection**

All dependencies are injected through constructors, promoting testability and loose coupling:

```go
// Example: Service receives dependencies via constructor
func NewArticleService(
    llmService LLMService,
    filterChain *FilterChain,
    trendingService TrendingService,
    articleRepo repositories.ArticleRepository,
) ArticleService
```

**Benefits:**
- Easy to mock dependencies for testing
- Clear dependency relationships
- Flexible component replacement

### 3. **Interface-Based Design**

Services and repositories use interfaces to define contracts:

```go
// Service interface
type ArticleService interface {
    ProcessArticleQuery(query string, location *models.Location) ([]models.Article, error)
    GetTrendingNews(lat, lon float64, limit int) ([]models.Article, error)
    // ...
}
```

**Benefits:**
- Enables polymorphism and easy testing
- Allows multiple implementations
- Reduces coupling between layers

### 4. **Factory Pattern**

Used for creating filters dynamically based on intent types:

```go
// FilterFactory creates filters from intent parameters
type FilterFactory func(params map[string]interface{}) Filter

// Registry maps intent types to factories
filterRegistry map[string]FilterFactory
```

**Usage:** The `FilterChain` uses factories to create appropriate filters based on LLM-extracted intents.

### 5. **Chain of Responsibility Pattern**

The filter chain pattern processes articles through a series of filters:

```go
// Chain composes multiple filters into a pipeline
func Chain(filters ...Filter) Filter {
    return func(ctx context.Context, articles []models.Article) ([]models.Article, error) {
        // Apply filters sequentially
    }
}
```

**Benefits:**
- Flexible query processing
- Easy to add/remove filters
- Supports complex multi-intent queries

### 6. **Singleton Pattern**

The logger uses the singleton pattern to ensure a single instance across the application:

```go
var (
    instance *structuredLogger
    once     sync.Once
)

func GetLogger() Logger {
    once.Do(func() {
        // Initialize logger once
    })
    return instance
}
```

**Benefits:**
- Consistent logging configuration
- Thread-safe initialization
- Global access point

### 7. **Repository Pattern**

Data access is abstracted through repository interfaces:

```go
type ArticleRepository interface {
    Insert(article *models.Article) error
    FindAll() ([]models.Article, error)
    FilterArticles(params types.FilterArticlesRequest) ([]models.Article, error)
    // ...
}
```

**Benefits:**
- Decouples business logic from data access
- Easy to swap database implementations
- Centralized data access logic

### 8. **Structured Logging**

All logging uses structured fields for better observability:

```go
logger.Info("Processing query", map[string]interface{}{
    "query": query,
    "user_id": userID,
    "duration": duration,
})
```

**Benefits:**
- Machine-readable logs
- Easy to query and filter
- Better debugging and monitoring

### 9. **Request/Response DTOs with Validation**

Separate types for API requests and responses with built-in validation:

```go
type QueryArticlesRequest struct {
    Query string  `query:"query" validate:"required"`
    Lat   float64 `query:"lat" validate:"omitempty,min=-90,max=90"`
    Lon   float64 `query:"lon" validate:"omitempty,min=-180,max=180"`
}

func (r *QueryArticlesRequest) Validate() error {
    // Custom validation logic
}
```

**Benefits:**
- Clear API contracts
- Centralized validation
- Type safety

### 10. **Error Handling Middleware**

Centralized error handling using Fiber's error handler:

```go
type AppError struct {
    Code    int
    Message string
    Err     error
}

func ErrorHandler(c *fiber.Ctx, err error) error {
    // Convert errors to appropriate HTTP responses
}
```

**Benefits:**
- Consistent error responses
- Centralized error logging
- Clean error propagation

### 11. **Caching Strategy**

Redis caching for expensive operations (trending news):

```go
// Check cache first
cachedArticles, found := trendingService.GetCachedTrending(lat, lon, limit)
if found {
    return cachedArticles, nil
}
// Compute and cache
trendingService.CacheTrending(lat, lon, articles)
```

**Benefits:**
- Improved performance
- Reduced database load
- Configurable TTL

### 12. **Context Pattern**

Filters use `context.Context` for cancellation and timeout support:

```go
type Filter func(ctx context.Context, in []models.Article) ([]models.Article, error)
```

**Benefits:**
- Request cancellation support
- Timeout handling
- Request-scoped values

### 13. **Service Container Pattern**

A `Services` struct holds all service instances for organized dependency management:

```go
type Services struct {
    LLM         LLMService
    Trending    TrendingService
    Article     ArticleService
    FilterChain *FilterChain
    Repos       *repositories.Repositories
}
```

**Benefits:**
- Centralized service initialization
- Easy dependency management
- Clear service relationships

### 14. **Infrastructure Abstraction**

Infrastructure components (DB, Redis, Logger) are abstracted:

```go
type Infrastructure struct {
    DB     *gorm.DB
    Redis  *redis.Client
    Logger Logger
}
```

**Benefits:**
- Easy to mock for testing
- Centralized resource management
- Graceful shutdown support

### 15. **Custom JSON Unmarshaling**

Models implement custom JSON unmarshaling for date parsing:

```go
func (a *Article) UnmarshalJSON(data []byte) error {
    // Custom date parsing logic
}
```

**Benefits:**
- Flexible date format handling
- Type conversion at unmarshal time
- Cleaner API

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

You can also load data via the REST API endpoint. See the [Load Data from JSON](#load-data-from-json) endpoint documentation for details.

**Example with curl:**
```bash
curl -X POST http://localhost:8080/api/v1/news/load \
  -H "Content-Type: application/json" \
  -d '{"filepath": "/data/articles.json"}'
```

**Note:** The filepath must be accessible from the server's filesystem. This method is useful for automated workflows or when the JSON file is already on the server.

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

**Description:** Health check endpoint to verify the API is running.

**Response:**
```json
{
  "status": "healthy",
  "service": "inshorts-api"
}
```

**Status Codes:**
- `200 OK`: Service is healthy

---

### Create Article

```http
POST /api/v1/news
Content-Type: application/json
```

**Description:** Create a new article in the database. The article will be automatically enriched with an LLM-generated summary if not provided.

**Request Body:**
```json
{
  "title": "Article Title",
  "description": "Article description text",
  "url": "https://example.com/article",
  "publication_date": "2024-04-28T10:00:00",
  "source_name": "News Source",
  "category": ["Technology", "Business"],
  "relevance_score": 0.85,
  "latitude": 37.7749,
  "longitude": -122.4194,
  "summary": "Optional pre-generated summary"
}
```

**Field Requirements:**
- `title` (required): Article headline
- `url` (required): Valid URL to the full article
- `publication_date` (required): ISO 8601 format: `2006-01-02T15:04:05`
- `source_name` (required): Name of the news source
- `category` (required): Array of category strings (at least one)
- `relevance_score` (required): Float between 0 and 1
- `latitude` (required): Float between -90 and 90
- `longitude` (required): Float between -180 and 180
- `description` (optional): Article summary or excerpt
- `summary` (optional): LLM-generated summary (auto-generated if not provided)

**Response:**
```json
{
  "success": true,
  "message": "Article created successfully",
  "article": {
    "id": "uuid",
    "title": "Article Title",
    "description": "Article description...",
    "url": "https://example.com/article",
    "publication_date": "2024-04-28T10:00:00Z",
    "source_name": "News Source",
    "category": ["Technology"],
    "relevance_score": 0.85,
    "latitude": 37.7749,
    "longitude": -122.4194,
    "summary": "LLM-generated summary..."
  }
}
```

**Status Codes:**
- `201 Created`: Article created successfully
- `400 Bad Request`: Invalid input parameters
- `500 Internal Server Error`: Failed to create article

---

### Query News (Natural Language)

```http
GET /api/v1/news/query?query=<query>&lat=<latitude>&lon=<longitude>
```

**Description:** Process a natural language query using LLM to extract intents and entities, then retrieve relevant news articles using a filter chain.

**Query Parameters:**
- `query` (required): Natural language query string
- `lat` (optional): Latitude (-90 to 90), must be provided with `lon`
- `lon` (optional): Longitude (-180 to 180), must be provided with `lat`

**Example:**
```http
GET /api/v1/news/query?query=Latest technology news about AI near San Francisco&lat=37.7749&lon=-122.4194
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
      "summary": "LLM-generated summary..."
    }
  ]
}
```

**Note:** Returns a maximum of 5 articles, sorted by relevance.

**Status Codes:**
- `200 OK`: Query processed successfully
- `400 Bad Request`: Invalid query parameters
- `500 Internal Server Error`: Failed to process query

---

### Get Trending News

```http
GET /api/v1/news/trending?lat=<latitude>&lon=<longitude>&limit=<limit>
```

**Description:** Retrieve trending news articles based on location and user engagement metrics. Results are cached in Redis for performance.

**Query Parameters:**
- `lat` (required): Latitude (-90 to 90)
- `lon` (required): Longitude (-180 to 180)
- `limit` (optional): Number of articles to return (default: 10, max: 100)

**Example:**
```http
GET /api/v1/news/trending?lat=37.7749&lon=-122.4194&limit=10
```

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
      "summary": "LLM-generated summary..."
    }
  ]
}
```

**Status Codes:**
- `200 OK`: Trending articles retrieved successfully
- `400 Bad Request`: Invalid query parameters
- `500 Internal Server Error`: Failed to retrieve trending news

---

### Filter Articles

```http
GET /api/v1/news/filter?category=<category>&source=<source>&lat=<latitude>&lon=<longitude>&radius=<radius>
```

**Description:** Filter articles by category, source, or geographic location. At least one filter parameter must be provided.

**Query Parameters:**
- `category` (optional): Filter by category name
- `source` (optional): Filter by source name
- `lat` (optional): Latitude for location-based filtering (must be provided with `lon`)
- `lon` (optional): Longitude for location-based filtering (must be provided with `lat`)
- `radius` (optional): Radius in kilometers for location-based filtering (default: 50km)

**Example:**
```http
GET /api/v1/news/filter?category=Technology&source=Reuters&lat=37.7749&lon=-122.4194&radius=25
```

**Response:**
```json
{
  "articles": [
    {
      "id": "uuid",
      "title": "Filtered Article",
      "description": "Article description...",
      "url": "https://example.com/article",
      "publication_date": "2024-04-28T10:00:00Z",
      "source_name": "Reuters",
      "category": ["Technology"],
      "relevance_score": 0.90,
      "latitude": 37.7749,
      "longitude": -122.4194,
      "summary": "LLM-generated summary..."
    }
  ]
}
```

**Status Codes:**
- `200 OK`: Articles filtered successfully
- `400 Bad Request`: Invalid filter parameters or no filters provided
- `500 Internal Server Error`: Failed to filter articles

---

### Load Data from JSON

```http
POST /api/v1/news/load
Content-Type: application/json
```

**Description:** Load articles from a JSON file on the server filesystem. Articles are automatically enriched with LLM-generated summaries before insertion.

**Request Body:**
```json
{
  "filepath": "/path/to/articles.json"
}
```

**Field Requirements:**
- `filepath` (required): Absolute or relative path to the JSON file on the server

**Response (Success):**
```json
{
  "success": true,
  "message": "Data loaded successfully",
  "total_articles": 100,
  "success_count": 98,
  "error_count": 2
}
```

**Response (Validation Errors):**
```json
{
  "success": false,
  "message": "Validation failed",
  "total_articles": 100,
  "success_count": 95,
  "error_count": 5,
  "validation_errors": [
    "Article at index 5: title is required",
    "Article at index 12: invalid URL format"
  ]
}
```

**Status Codes:**
- `200 OK`: Data loaded successfully (may include validation errors)
- `400 Bad Request`: Validation failed or file not found
- `500 Internal Server Error`: Failed to load data

---

### Record User Interaction

```http
POST /api/v1/interactions/record
Content-Type: application/json
```

**Description:** Record a user interaction event (view or click) with an article. Used for computing trending scores.

**Request Body:**
```json
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

**Field Requirements:**
- `user_id` (required): Unique identifier for the user
- `article_id` (required): UUID of the article
- `event_type` (required): Must be either `"view"` or `"click"`
- `location` (required): Geographic coordinates
  - `latitude` (required): Float between -90 and 90
  - `longitude` (required): Float between -180 and 180

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

**Status Codes:**
- `200 OK`: Interaction recorded successfully
- `400 Bad Request`: Invalid request body or missing required fields
- `500 Internal Server Error`: Failed to record interaction

## Query Examples

### Category-based Query
```http
GET /api/v1/news/query?query=Show me sports news
```

### Location-based Query
```http
GET /api/v1/news/query?query=News near me&lat=37.7749&lon=-122.4194
```

### Source-based Query
```http
GET /api/v1/news/query?query=Latest articles from Reuters
```

### Complex Multi-intent Query
```http
GET /api/v1/news/query?query=Technology news about artificial intelligence from New York Times near San Francisco&lat=37.7749&lon=-122.4194
```

### Filter Examples

**Filter by Category:**
```http
GET /api/v1/news/filter?category=Technology
```

**Filter by Source:**
```http
GET /api/v1/news/filter?source=Reuters
```

**Filter by Location:**
```http
GET /api/v1/news/filter?lat=37.7749&lon=-122.4194&radius=25
```

**Combined Filters:**
```http
GET /api/v1/news/filter?category=Technology&source=Reuters&lat=37.7749&lon=-122.4194
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
├── main.go                      # Application entry point
├── src/
│   ├── controllers/
│   │   ├── article.go           # Article controller (CRUD, query, filter, trending)
│   │   ├── controllers.go       # Controller factory/container
│   │   └── user_interaction.go  # User interaction controller
│   ├── infra/
│   │   ├── config.go            # Configuration management
│   │   ├── database.go          # Database initialization (GORM)
│   │   ├── infra.go             # Infrastructure container
│   │   ├── logger.go            # Structured logger (singleton)
│   │   └── redis.go             # Redis client initialization
│   ├── middleware/
│   │   └── error_handler.go    # Centralized error handling
│   ├── models/
│   │   └── models.go           # Domain models (Article, UserEvent, Intent, etc.)
│   ├── repositories/
│   │   ├── article.go           # Article repository (data access)
│   │   ├── repositories.go      # Repository factory/container
│   │   └── user_event.go        # User event repository
│   ├── routes/
│   │   └── routes.go           # Route definitions and middleware setup
│   ├── services/
│   │   ├── article.go           # Article service (business logic)
│   │   ├── filter_chain.go     # Filter chain orchestrator
│   │   ├── filters.go          # Individual filter implementations
│   │   ├── llm.go              # LLM service (OpenAI integration)
│   │   ├── services.go         # Service factory/container
│   │   └── trending.go         # Trending news computation
│   └── types/
│       ├── article_types.go    # Article-related request/response DTOs
│       └── user_interaction_types.go  # User interaction DTOs
├── .env.example                 # Example environment variables
├── docker-compose.yml           # Docker Compose configuration
├── Dockerfile                   # Docker image definition
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
├── init.sql                     # Database initialization script
├── news_data.json               # Sample news data
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
