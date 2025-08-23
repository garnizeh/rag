# Project Plan: AI-Powered Engineer Context and Profile System

## 1. Overview

This document outlines a comprehensive plan for building an intelligent system that manages and understands software engineers' professional contexts. The system ingests natural language activity logs, leverages Large Language Models (LLMs) to extract and structure contextual information, and maintains dynamic profiles for engineers.

### Key Features
- **Automatic Context Extraction**: Transforms informal activity logs into structured data about projects, teams, and collaborations
- **Interactive Learning**: Asks clarifying questions to resolve ambiguities and improve accuracy
- **Dynamic Profile Generation**: Creates and updates professional summaries based on current context
- **Asynchronous Processing**: Handles AI inference in the background for responsive user experience

## 2. Core Concepts & Data Model

### Engineer Context Model
A comprehensive, structured representation of an engineer's professional environment, including:
- **Team Affiliation**: Current team or department
- **Active Projects**: Projects currently worked on or recently completed
- **Collaborators**: Colleagues regularly worked with
- **Skills & Technologies**: Demonstrated technical competencies
- **Work Patterns**: Inferred from activity patterns and descriptions

### Activity Ingestion Pipeline
Engineers provide informal, natural language updates about their work activities:
- Examples: "finished refactoring the auth module", "paired with Jane on the new API", "deployed microservice to production"
- No rigid format required - the AI handles natural language variations
- Activities serve as the primary data source for context inference

### AI-Powered Inference Engine
An asynchronous processing system that:
- **Entity Extraction**: Identifies people, projects, technologies, and teams from activity descriptions
- **Relationship Mapping**: Understands connections between entities (e.g., "worked with X on Y")
- **Context Evolution**: Tracks changes in the engineer's professional landscape over time
- **Confidence Assessment**: Evaluates certainty of inferences to determine when clarification is needed

### Interactive Clarification System
When the AI encounters ambiguous information:
- Generates targeted questions for the engineer
- Examples: "Is 'Project Phoenix' a new project or part of an existing initiative?"
- Maintains a feedback loop to continuously improve accuracy
- Learns from responses to reduce future ambiguities

### Dynamic Profile Synthesis
Generates professional summaries by:
- Analyzing the complete Engineer Context Model
- Creating concise, professional narratives
- Updating profiles as context evolves
- Maintaining consistency across profile versions

## 3. Technical Architecture

### System Components

#### Backend API Server (Go)
- **Framework**: Standard library with gorilla/mux for routing
- **Authentication**: JWT-based authentication for engineer identification
- **Rate Limiting**: Protect against abuse of AI inference endpoints
- **Logging**: Structured logging for debugging and monitoring
- **Health Checks**: System status and dependency health monitoring

#### Database Layer
- **Primary Database**: SQLite with sqlc for type-safe SQL queries
- **Migration System**: Database versioning and schema evolution using goose library
- **Connection Pooling**: Efficient database connection management
- **Backup Strategy**: Regular automated backups of engineer data

#### AI Integration Layer
- **LLM Provider**: Local Ollama instance for privacy and control
- **Model Management**: Support for multiple models (deepseek-r1:32b, llama3, etc.)
- **Prompt Engineering**: Templated prompts with version control
- **Fallback Mechanisms**: Handle AI service unavailability gracefully
- **Response Validation**: Ensure AI outputs conform to expected schemas

#### Background Processing
- **Job Queue**: Asynchronous processing of activities and profile updates
- **Worker Pool**: Concurrent processing of AI inference tasks
- **Retry Logic**: Handle transient failures in AI processing
- **Dead Letter Queue**: Manage permanently failed processing attempts

### Deployment Architecture
- **Containerization**: Docker containers for consistent deployment
- **Environment Configuration**: Separate configs for dev/staging/production
- **Monitoring**: Application metrics and alerting
- **Scaling**: Horizontal scaling capabilities for API and workers

## 4. Database Schema Design

The database serves as the single source of truth for all engineer data and system state.

### Core Tables

```sql
-- Engineers table: Basic information and current context
CREATE TABLE engineers (
    id                 INTEGER PRIMARY KEY,
    name               TEXT NOT NULL UNIQUE,
    email              TEXT UNIQUE,
    created_at         INTEGER NOT NULL DEFAULT (unixepoch() * 1000), -- Unix milliseconds UTC
    updated_at         INTEGER NOT NULL DEFAULT (unixepoch() * 1000), -- Unix milliseconds UTC
    
    -- AI-maintained context fields (updated by inference engine)
    current_team       TEXT,
    collaborators      TEXT, -- JSON array: ["Jane Doe", "Bob Smith"]
    projects           TEXT, -- JSON array: ["Project Phoenix", "API Refactor"]
    skills             TEXT, -- JSON array: ["Go", "PostgreSQL", "Kubernetes"]
    work_patterns      TEXT  -- JSON object with inferred patterns
);

-- Activity logs: Raw input data from engineers
CREATE TABLE raw_activities (
    id          INTEGER PRIMARY KEY,
    engineer_id INTEGER NOT NULL,
    content     TEXT NOT NULL,
    created_at  INTEGER NOT NULL DEFAULT (unixepoch() * 1000), -- Unix milliseconds UTC
    processed_at INTEGER, -- Unix milliseconds UTC when AI processing completed
    processing_status TEXT DEFAULT 'pending', -- pending, processing, completed, failed
    
    FOREIGN KEY (engineer_id) REFERENCES engineers (id) ON DELETE CASCADE
);

-- Generated profiles: AI-synthesized professional summaries
CREATE TABLE engineer_profiles (
    id          INTEGER PRIMARY KEY,
    engineer_id INTEGER NOT NULL UNIQUE,
    profile_text TEXT NOT NULL,
    version     INTEGER NOT NULL DEFAULT 1,
    created_at  INTEGER NOT NULL DEFAULT (unixepoch() * 1000), -- Unix milliseconds UTC
    updated_at  INTEGER NOT NULL DEFAULT (unixepoch() * 1000), -- Unix milliseconds UTC
    
    FOREIGN KEY (engineer_id) REFERENCES engineers (id) ON DELETE CASCADE
);

-- AI clarification questions and responses
CREATE TABLE ai_questions (
    id              INTEGER PRIMARY KEY,
    engineer_id     INTEGER NOT NULL,
    question_text   TEXT NOT NULL,
    context_data    TEXT, -- JSON with relevant context for the question
    created_at      INTEGER NOT NULL DEFAULT (unixepoch() * 1000), -- Unix milliseconds UTC
    
    -- Response tracking
    is_answered     BOOLEAN NOT NULL DEFAULT 0,
    answer_text     TEXT,
    answered_at     INTEGER, -- Unix milliseconds UTC
    
    -- Processing metadata
    question_type   TEXT, -- 'entity_clarification', 'project_scope', 'team_change', etc.
    priority        INTEGER DEFAULT 1, -- 1=low, 2=medium, 3=high
    
    FOREIGN KEY (engineer_id) REFERENCES engineers (id) ON DELETE CASCADE
);

-- Processing jobs: Track background AI tasks
CREATE TABLE processing_jobs (
    id              INTEGER PRIMARY KEY,
    job_type        TEXT NOT NULL, -- 'activity_processing', 'profile_generation'
    engineer_id     INTEGER NOT NULL,
    related_id      INTEGER, -- ID of related record (activity, question, etc.)
    status          TEXT NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
    error_message   TEXT,
    created_at      INTEGER NOT NULL DEFAULT (unixepoch() * 1000), -- Unix milliseconds UTC
    started_at      INTEGER, -- Unix milliseconds UTC
    completed_at    INTEGER, -- Unix milliseconds UTC
    retry_count     INTEGER DEFAULT 0,
    
    FOREIGN KEY (engineer_id) REFERENCES engineers (id) ON DELETE CASCADE
);
```

### Indexes for Performance
```sql
CREATE INDEX idx_activities_engineer_created ON raw_activities(engineer_id, created_at);
CREATE INDEX idx_questions_engineer_answered ON ai_questions(engineer_id, is_answered);
CREATE INDEX idx_jobs_status_created ON processing_jobs(status, created_at);
CREATE INDEX idx_engineers_updated ON engineers(updated_at);
CREATE INDEX idx_profiles_updated ON engineer_profiles(updated_at);
```

## 5. API Design

RESTful API providing comprehensive access to system functionality.

### Authentication & Authorization
All endpoints require JWT authentication with engineer-specific scoping.

### Core Endpoints

#### Activity Management
```http
POST /v1/activities
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
    "activity_text": "Deployed the final changes for Project Phoenix with Jane Doe.",
    "timestamp": 1724421000000 // Optional Unix milliseconds UTC, defaults to current time
}

Response: 201 Created
{
    "id": 123,
    "status": "queued_for_processing",
    "estimated_processing_time": "30s"
}
```

```http
GET /v1/activities?limit=50&offset=0
Authorization: Bearer <jwt_token>

Response: 200 OK
{
    "activities": [
        {
            "id": 123,
            "content": "Deployed the final changes...",
            "created_at": 1724421000000, // Unix milliseconds UTC
            "processing_status": "completed"
        }
    ],
    "total": 150,
    "has_more": true
}
```

#### Profile Management
```http
GET /v1/engineers/{id}/profile
Authorization: Bearer <jwt_token>

Response: 200 OK
{
    "engineer_id": 1,
    "profile_text": "Senior Backend Engineer with expertise in Go and distributed systems...",
    "version": 3,
    "updated_at": 1724421000000, // Unix milliseconds UTC
    "context": {
        "current_team": "Core Services",
        "projects": ["Project Phoenix", "API Refactor"],
        "collaborators": ["Jane Doe", "Bob Smith"],
        "skills": ["Go", "PostgreSQL", "Kubernetes"]
    }
}
```

```http
POST /v1/engineers/{id}/profile/regenerate
Authorization: Bearer <jwt_token>

Response: 202 Accepted
{
    "job_id": "job_456",
    "estimated_completion": "60s"
}
```

#### Question & Answer System
```http
GET /v1/engineers/{id}/questions?answered=false
Authorization: Bearer <jwt_token>

Response: 200 OK
{
    "questions": [
        {
            "id": 789,
            "question_text": "Is 'Project Phoenix' a new project or part of an existing initiative?",
            "question_type": "entity_clarification",
            "priority": 2,
            "created_at": 1724421000000, // Unix milliseconds UTC
            "context": {
                "activity": "Deployed the final changes for Project Phoenix",
                "existing_projects": ["API Refactor", "Database Migration"]
            }
        }
    ]
}
```

```http
POST /v1/questions/{id}/answer
Content-Type: application/json
Authorization: Bearer <jwt_token>

{
    "answer_text": "Project Phoenix is a new microservice project for payment processing"
}

Response: 200 OK
{
    "question_id": 789,
    "status": "answered",
    "processing_queued": true
}
```

#### System Status & Monitoring
```http
GET /v1/system/health
Response: 200 OK
{
    "status": "healthy",
    "components": {
        "database": "healthy",
        "ai_service": "healthy",
        "job_queue": "healthy"
    },
    "version": "1.0.0"
}
```

```http
GET /v1/engineers/{id}/processing-status
Authorization: Bearer <jwt_token>

Response: 200 OK
{
    "pending_activities": 2,
    "unanswered_questions": 1,
    "profile_last_updated": 1724421000000, // Unix milliseconds UTC
    "next_update_estimate": 1724422800000  // Unix milliseconds UTC
}
```

### Error Handling
Consistent error response format:
```json
{
    "error": {
        "code": "VALIDATION_ERROR",
        "message": "Activity text cannot be empty",
        "details": {
            "field": "activity_text",
            "constraint": "min_length"
        }
    }
}
```

## 6. AI Processing Workflow

### Activity Processing Pipeline

#### 1. Activity Ingestion & Validation
- Engineer submits activity via API
- Validate activity content and format
- Store in `raw_activities` table with `pending` status
- Queue background processing job
- Return immediate response to user

#### 2. Context Retrieval & Preparation
```go
type ProcessingContext struct {
    Engineer     Engineer
    CurrentContext ContextModel
    RecentActivities []Activity
    PendingQuestions []Question
}
```

#### 3. AI Inference & Analysis
**Primary Prompt Template:**
```
System: You are an expert AI assistant that maintains structured professional context for software engineers. Analyze activities to extract entities and relationships while maintaining data consistency.

Context Schema:
- Team: String (current team/department)
- Projects: Array of project names
- Collaborators: Array of colleague names  
- Skills: Array of technologies/competencies
- Work Patterns: Object with inferred behavioral patterns

Current Context:
{json_context}

Recent Activities (for context):
{recent_activities}

New Activity to Process:
"{activity_content}"

Instructions:
1. Identify new entities (people, projects, technologies)
2. Determine context updates needed
3. Assess confidence level for each inference
4. Generate clarifying questions for low-confidence items

Output Format:
{
    "confidence": "high|medium|low",
    "context_updates": {
        "team": "updated_value_or_null",
        "projects": ["new_or_updated_projects"],
        "collaborators": ["new_collaborators"],
        "skills": ["newly_demonstrated_skills"]
    },
    "clarifying_questions": [
        {
            "type": "entity_clarification|project_scope|team_change",
            "question": "Clear, specific question text",
            "context": "Additional context for the question"
        }
    ],
    "reasoning": "Brief explanation of inference logic"
}
```

#### 4. Response Processing & Context Updates
- Parse AI response and validate JSON structure
- High confidence updates: Apply directly to engineer context
- Medium/Low confidence items: Generate clarifying questions
- Update `raw_activities` status to `completed`
- Trigger profile regeneration if context changed significantly

#### 5. Error Handling & Retry Logic
```go
type ProcessingJob struct {
    MaxRetries    int
    BackoffFactor time.Duration
    ErrorHandlers map[string]ErrorHandler
}

// Retry strategies:
// - AI service unavailable: Exponential backoff
// - Invalid response format: Log and mark as failed
// - Rate limiting: Delay and retry
// - Context conflicts: Generate clarification question
```

### Profile Generation Workflow

#### 1. Trigger Conditions
- Significant context changes (new project, team change)
- Manual regeneration request
- Scheduled periodic updates (configurable)
- After answering clarifying questions

#### 2. Profile Synthesis Process
**Profile Generation Prompt:**
```
System: Generate a professional summary for a software engineer based on their current context and work patterns.

Engineer Context:
{complete_context}

Recent Activity Summary:
{activity_summary}

Requirements:
- 2-3 paragraph professional summary
- Highlight current role and responsibilities  
- Emphasize demonstrated skills and technologies
- Mention key projects and collaborations
- Maintain professional tone and accuracy
- Focus on recent and relevant information

Output a professional profile text suitable for internal directories or project assignments.
```

#### 3. Profile Versioning & History
- Maintain version history for profile changes
- Track what context changes triggered updates
- Enable rollback to previous versions if needed

### Question Resolution Workflow

#### 1. Question Generation Strategy
- Prioritize questions by impact on context accuracy
- Group related questions to reduce engineer burden
- Use natural language that's easy to understand
- Provide sufficient context for informed answers

#### 2. Answer Processing
- Parse engineer responses for structured data
- Update context based on clarifications
- Mark questions as resolved
- Trigger follow-up processing if needed

#### 3. Learning & Improvement
- Track question types and success rates
- Adjust confidence thresholds based on feedback
- Improve prompt engineering based on common clarifications

## 7. Implementation Plan

### Phase 1: Foundation (Weeks 1-2)
- [ ] Set up Go project structure with proper modules
- [ ] Implement database schema and migration system
- [ ] Create basic API server with authentication
- [ ] Set up Ollama integration and model management
- [ ] Implement core data models and repository patterns

### Phase 2: Core Processing (Weeks 3-4)
- [ ] Build activity ingestion pipeline
- [ ] Implement AI inference engine with prompt templates
- [ ] Create background job processing system
- [ ] Develop context update logic and validation
- [ ] Add basic error handling and retry mechanisms

### Phase 3: Intelligence Features (Weeks 5-6)
- [ ] Implement question generation and answer processing
- [ ] Build profile synthesis system
- [ ] Add confidence assessment and learning capabilities
- [ ] Create context conflict resolution logic
- [ ] Implement activity pattern analysis

### Phase 4: User Experience (Weeks 7-8)
- [ ] Complete API endpoint implementation
- [ ] Add comprehensive validation and error handling
- [ ] Implement real-time status updates
- [ ] Create monitoring and health check systems
- [ ] Add rate limiting and security hardening

### Phase 5: Production Readiness (Weeks 9-10)
- [ ] Performance optimization and load testing
- [ ] Comprehensive test suite (unit, integration, E2E)
- [ ] Documentation and API specifications
- [ ] Deployment automation and monitoring
- [ ] Security audit and penetration testing

## 8. Technical Considerations

### Performance & Scalability
- **AI Processing**: Queue-based async processing to handle AI latency
- **Database**: Proper indexing and query optimization for growing datasets
- **Caching**: Redis for frequently accessed profiles and context data
- **Rate Limiting**: Protect against abuse while allowing normal usage patterns

### Security & Privacy
- **Data Protection**: Encrypt sensitive activity data at rest
- **Access Control**: Role-based permissions for different user types
- **AI Privacy**: Local Ollama deployment ensures data doesn't leave infrastructure
- **Audit Logging**: Track all data access and modifications

### Monitoring & Observability
- **Metrics**: Track AI processing times, accuracy rates, question resolution
- **Alerting**: Monitor system health, processing queues, error rates
- **Logging**: Structured logging for debugging and analysis
- **Dashboards**: Real-time visibility into system performance

### Configuration Management
```go
type Config struct {
    Server struct {
        Port         int
        ReadTimeout  time.Duration
        WriteTimeout time.Duration
    }
    Database struct {
        Path            string
        MaxConnections  int
        MigrationPath   string
    }
    AI struct {
        OllamaURL       string
        DefaultModel    string
        RequestTimeout  time.Duration
        MaxRetries      int
    }
    Processing struct {
        WorkerCount     int
        QueueSize       int
        RetryDelays     []time.Duration
    }
}
```

## 9. Success Metrics

### Technical Metrics
- **Processing Latency**: < 30 seconds for activity processing
- **AI Accuracy**: > 85% confidence in entity extraction
- **Question Resolution**: < 24 hours average response time
- **System Uptime**: 99.9% availability
- **Error Rate**: < 1% of processing jobs fail

### User Experience Metrics
- **Profile Accuracy**: User satisfaction scores > 4/5
- **Question Quality**: < 10% of questions marked as unclear
- **Context Completeness**: > 90% of profiles have complete team/project data
- **Engagement**: Average response rate to questions > 80%

### Business Impact Metrics
- **Onboarding Time**: Reduce new engineer context gathering by 70%
- **Project Assignment**: Improve project-engineer matching accuracy
- **Knowledge Sharing**: Increase cross-team collaboration insights
- **Skill Tracking**: Better visibility into team capabilities and gaps

## 10. Future Enhancements

### Advanced AI Features
- **Sentiment Analysis**: Understand engineer satisfaction and workload
- **Skill Gap Analysis**: Identify learning opportunities and training needs
- **Project Risk Assessment**: Flag potential issues based on activity patterns
- **Automated Recommendations**: Suggest collaborations and project assignments

### Integration Capabilities
- **Version Control**: Integrate with Git for automatic activity detection
- **Calendar Integration**: Infer meetings and collaboration patterns
- **Slack/Teams**: Process communication data for additional context
- **JIRA/Linear**: Correlate tickets with activity descriptions

### Advanced Analytics
- **Team Dynamics**: Analyze collaboration patterns and effectiveness
- **Productivity Insights**: Identify factors that impact engineer performance
- **Knowledge Networks**: Map expertise distribution across the organization
- **Predictive Modeling**: Forecast project timelines and resource needs

### Deployment Options
- **Multi-tenant SaaS**: Support multiple organizations
- **Enterprise On-premise**: Complete control for security-conscious organizations
- **Hybrid Cloud**: Flexible deployment models
- **Mobile Applications**: Native apps for activity submission and profile viewing