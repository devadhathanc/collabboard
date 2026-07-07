# IdeaBoard

IdeaBoard is a real-time collaborative brainstorming application. It features a Go backend with WebSocket support for live updates and a React (Vite) frontend. It uses a **locally fine-tuned AI model** running via [Ollama](https://ollama.com) to intelligently synthesize and suggest new ideas — **no internet or API key required**.

## Features

- **Real-Time Collaboration** — Ideas are instantly broadcasted to all connected clients using WebSockets.
- **Offline AI Synthesis** — Uses a fine-tuned Gemma 2B model (via Ollama) to brainstorm and suggest up to 3 new related ideas based on what's already on the board. No cloud API needed.
- **RESTful API** — Manage boards and ideas via simple HTTP endpoints.
- **Modern Frontend** — Built with React, TypeScript, and Vite.
- **Go Backend** — High-performance backend using Go and `gorilla/websocket`.

## Project Structure

```
ideaboard/
├── main.go              # Server entrypoint
├── board/               # In-memory board & idea store
├── handlers/            # REST API handlers
├── ollama/              # Local AI client (Ollama)
├── ws/                  # WebSocket hub
├── client/              # React + Vite frontend
└── models/              # Local GGUF model files (gitignored)
    ├── ideaboard-q8.gguf
    └── Modelfile
```

## Prerequisites

- [Go](https://golang.org/) 1.22+
- [Node.js](https://nodejs.org/) (for the frontend)
- [Ollama](https://ollama.com/download) (for local AI synthesis)

## Setup & Running

### 1. Install Ollama

```bash
brew install ollama
```

### 2. Load the Fine-Tuned Model

Place the `ideaboard-q8.gguf` file inside the `models/` directory, then run:

```bash
cd models/
ollama create ideaboard -f Modelfile
```

### 3. Start Ollama

```bash
ollama serve
```

### 4. Backend

From the project root:

```bash
go mod tidy
go run main.go
```

The backend will start on `http://localhost:8090`. You should see:

```
[ollama] connected — using local model "ideaboard"
IdeaBoard server listening on :8090
```

### 5. Frontend

```bash
cd client
npm install
npm run dev
```

## API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `GET` | `/api/boards/{boardID}/ideas` | List all ideas on a board |
| `POST` | `/api/boards/{boardID}/ideas` | Submit a new idea |
| `POST` | `/api/boards/{boardID}/ideas/synthesize` | Trigger AI to suggest 3 new ideas |
| `GET` | `/ws` | WebSocket for real-time updates |

### Submit Idea — Request Body
```json
{ "text": "remote work", "created_by": "username" }
```

### Synthesize — Example Response
```json
{
  "added": 3,
  "ideas": [
    { "text": "Virtual water cooler time", "created_by": "AI" },
    { "text": "Flexible time zones for global teams", "created_by": "AI" },
    { "text": "Project management tools for async workflows", "created_by": "AI" }
  ]
}
```

## WebSocket Events

| Event | Trigger |
|---|---|
| `IDEA_ADDED` | A new idea is submitted |
| `IDEA_UPDATED` | An existing idea is modified |

## AI Model

The synthesis feature runs a **fine-tuned Gemma 2B** model locally via Ollama.

| Property | Detail |
|---|---|
| Base model | `google/gemma-2-2b-it` |
| Fine-tuning | QLoRA on brainstorming datasets (Dolly 15K + DevQuasar brainstorm + custom IdeaBoard examples) |
| Format | GGUF Q8 (~2.8 GB) |
| Runtime | Ollama (localhost:11434) |
| API key needed | ❌ None |
| Internet needed | ❌ None |

The `gemini/` package acts as a drop-in client — the rest of the codebase is unchanged.
