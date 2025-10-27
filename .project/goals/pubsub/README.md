# Pubsub - Elixir Publisher-Subscriber Pattern

This project demonstrates a publisher-subscriber pattern implementation in
Elixir using GenServers and Registry for event management.

## Overview

The system consists of:

- **Publisher**: Publishes articles to the event bus
- **Event Bus**: Manages the pub-sub communication using Registry
- **Subscribers**:
  - **Markdown Subscriber**: Formats articles as Markdown files
  - **HTML Subscriber**: Formats articles as HTML files

## Architecture

```
+-------------+    +-------------+    +-----------------+
|  Publisher  |--->|             +--->|  Markdown Sub   |
+-------------|    |  Event Bus  |    +-----------------+
                   |             |    +-----------------+
                   |             +--->|    HTML Sub     |
                   +-------------+    +-----------------+
```

## Components

### Article Structure

Articles contain:

- `id`: Unique identifier
- `title`: Article title
- `content`: Article content
- `author`: Author name
- `published_at`: Publication timestamp

### Publisher

The `Pubsub.Publisher` GenServer:

- Automatically publishes a "Hello World" article on startup
- Provides functions to publish custom articles
- Uses the event bus to broadcast to all subscribers

### Event Bus

The `Pubsub.EventBus` GenServer:

- Uses Registry for subscriber management
- Handles event distribution to all registered subscribers
- Supports multiple event types

### Subscribers

1. **Markdown Subscriber** (`Pubsub.MarkdownSubscriber`):
   - Formats articles as Markdown
   - Saves to `article_{id}_markdown.md` files

2. **HTML Subscriber** (`Pubsub.HtmlSubscriber`):
   - Formats articles as HTML with CSS styling
   - Saves to `article_{id}_html.html` files

## Running the Application

If you're having problems to get it working, use `mise exec -- <command>` instead.
Replace the bash command bellow into `<command>`.

### Prerequisites

Make sure you have Elixir installed on your system.

### Setup

```bash
# Navigate to the pubsub directory
cd pubsub

# Install dependencies
mix deps.get

# Compile the project
mix compile
```

### Running

```bash
mix run demo.esx
```

### Testing

```bash
# Run all tests
mix test

# Run tests with coverage
mix test --cover
```

## Features

- **Supervision Tree**: All processes are supervised for fault tolerance
- **Registry-based Pub-Sub**: Uses Elixir's Registry for efficient event
  distribution
- **Multiple Output Formats**: Generates both Markdown and HTML versions
- **Automatic Publishing**: Publishes a sample article on startup
- **JSON Serialization**: Articles can be serialized to JSON
- **File Output**: Generated content is saved to files for inspection

## Example Output

When an article is published, you'll see files created:

- `article_hello-world-001_markdown.md` - Markdown formatted version
- `article_hello-world-001_html.html` - HTML formatted version

## Project Structure

```
pubsub/
├── lib/
│   ├── pubsub/
│   │   ├── application.ex          # Application supervisor
│   │   ├── event_bus.ex           # Event bus GenServer
│   │   ├── article.ex             # Article data structure
│   │   ├── publisher.ex           # Publisher GenServer
│   │   ├── markdown_subscriber.ex # Markdown formatter
│   │   └── html_subscriber.ex     # HTML formatter
│   └── pubsub.ex                  # Main module
├── config/                        # Configuration files
├── test/                          # Test files
└── mix.exs                        # Project configuration
```

This implementation showcases Elixir's strengths in building concurrent,
fault-tolerant systems with clean separation of concerns.
