# Multi-AI Chat Interface

This Python script provides a command-line interface for chatting with multiple AI models, including Claude and Gemini. It allows users to interact with different language models, switch between them, and manage conversations.

## Features

- Support for multiple AI models (Claude and Gemini)
- Easy model switching during conversation
- Conversation history management
- Color-coded output for better readability

## Prerequisites

- Python 2.7 or higher
- `requests` library

## Setup

1. Clone this repository or download the script.
2. Install the required library:
   ```sh
   pip install requests
   ```
3. Set up the following environment variables:
   - `MULTI_AI_CLAUDE_API_KEY`: Your Claude API key
   - `MULTI_AI_CLAUDE_EP`: Claude API endpoint
   - `MULTI_AI_GEMINI_CREDS`: Gemini API endpoint

## Usage

Run the script:

```sh
python multi_ai_chat.py
```

- Choose a model by entering its corresponding number.
- Type your messages and press Enter to send.
- Special commands:
  - `exit` or `quit`: End the chat session
  - `change_model`: Switch to a different AI model
  - `clear_convo`: Clear the conversation history

## Supported Models

- Claude 3 Haiku
- Claude 3 Sonnet
- Gemini 1.5 Flash

Note: Available models depend on your API credentials.

## License

[Specify your license here]

## Contributing

[Add contribution guidelines if applicable]

## Acknowledgments

- Anthropic for Claude API
- Google for Gemini API

```
