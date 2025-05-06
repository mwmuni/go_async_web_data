# Async Web Data

A proof of concept application demonstrating asynchronous pinging and fetching in Go.

## Overview

This project showcases concurrent operations in Go using goroutines to:

1. Asynchronously ping multiple websites
2. Asynchronously fetch content from those websites
3. Display performance metrics in a stylised terminal UI

## Features

- Concurrent network operations using Go's goroutines
- Performance measurement and comparison
- Sorting of results by response time and body size
- Clean terminal interface with colour-coded output
- Configuration via YAML file

## Requirements

- Go 1.16 or higher
- Internet connectivity to reach the configured websites

## Configuration

The application reads website URLs from a `websites.yaml` file in the following format:

```yaml
websites:
  - name: "Google"
    url: "https://www.google.com"
  - name: "GitHub"
    url: "https://www.github.com"
  # Add more websites as needed
```

## Usage

1. Clone the repository
2. Customise the `websites.yaml` file if desired
3. Run the application:

```bash
go run main.go
```

## How It Works

The application:

1. Loads website configurations from `websites.yaml`
2. Launches goroutines to ping each website simultaneously
3. Collects ping results through a channel
4. Launches goroutines to fetch content from each website simultaneously
5. Collects fetch results through a channel
6. Displays performance metrics in a formatted table

This demonstrates the power of Go's concurrency primitives by parallelising network operations that would traditionally be performed sequentially.

## Dependencies

- [github.com/charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [github.com/goccy/go-yaml](https://github.com/goccy/go-yaml) - YAML parsing
- [github.com/prometheus-community/pro-bing](https://github.com/prometheus-community/pro-bing) - ICMP pinging

## License

MIT
