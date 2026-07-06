# IdeaBoard

IdeaBoard is a real-time collaborative brainstorming application. It features a Go backend with WebSocket support for live updates and a React (Vite) frontend. Additionally, it integrates with the Gemini API to intelligently synthesize and generate new ideas based on the current board content.

## Features

- **Real-Time Collaboration**: Ideas are instantly broadcasted to all connected clients using WebSockets.
- **AI Synthesis**: Leverages Google's Gemini API to brainstorm and suggest up to 3 new related ideas based on what's already on the board.
- **RESTful API**: Manage boards and ideas via simple HTTP endpoints.
- **Modern Frontend**: Built with React, TypeScript, and Vite for a fast and responsive user experience.
- **Go Backend**: High-performance backend using Go and `gorilla/websocket`.

## Project Structure

- `/`: Go backend (main server, handlers, WebSocket hub, and Gemini integration).
- `/client`: React frontend built with Vite.

## Prerequisites

- [Go](https://golang.org/) 1.22 or higher
- [Node.js](https://nodejs.org/) (for the frontend)
- A Gemini API key (for AI synthesis features)

## Setup & Running

### Backend

1. Navigate to the root directory.
2. Create a `.env` file and add your Gemini API key:
   ```env
   GEMINI_API_KEY=your_api_key_here
   PORT=8090
   ```
3. Install dependencies:
   ```bash
   go mod tidy
   ```
4. Run the server:
   ```bash
   go run main.go
   ```
   The backend will start on `http://localhost:8090`.

### Frontend

1. Navigate to the `client` directory:
   ```bash
   cd client
   ```
2. Install dependencies:
   ```bash
   npm install
   ```
3. Start the development server:
   ```bash
   npm run dev
   ```

## API Endpoints

- `GET /health`: Health check endpoint.
- `GET /api/boards/{boardID}/ideas`: Fetch all ideas for a specific board.
- `POST /api/boards/{boardID}/ideas`: Submit a new idea.
  - Body: `{"text": "idea description", "created_by": "username"}`
- `POST /api/boards/{boardID}/ideas/synthesize`: Trigger AI to brainstorm new ideas based on the board's current content.
- `GET /ws`: WebSocket endpoint for real-time updates.

## WebSocket Events

The WebSocket connection broadcasts events when ideas are added or updated:
- `IDEA_ADDED`: Dispatched when a new idea is submitted.
- `IDEA_UPDATED`: Dispatched when an existing idea is modified (e.g., duplicated or merged).
