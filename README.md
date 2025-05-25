# KubeDepot

A web service that provides kubeconfig YAML files on request.

## Features

- List all available kubeconfigs
- Get a specific kubeconfig
- Merge multiple kubeconfigs

## Installation

1. Clone the repository
2. Run `go mod tidy` to fetch all dependencies
3. Build the application: `go build -o ./kubedepot ./cmd/kubedepot/`

## Configuration

The application can be configured using environment variables:

- `CONFIGS_DIR`: Directory containing kubeconfig files (default: `./configs`)
- `PORT`: HTTP server port (default: `8080`)
- `DEBUG`: Enable debug mode (default: `false`)

## Usage

### Starting the Server

```bash
# Start with default settings
./kubedepot

# Or with custom settings
CONFIGS_DIR=/path/to/configs PORT=9090 ./kubedepot
```

### API Endpoints

#### List All Configs

```
GET /json/list
GET /yaml/list
```

Returns a list of all available kubeconfigs in either JSON or YAML format.

#### Get Configs

```
GET /json/get?name=<config-name>
GET /yaml/get?name=<config-name>
GET /yaml/get?name=<config-name>&name=<config-name2>
```

Returns kubeconfig(s) in either JSON or YAML format. You can specify multiple `name` parameters to merge configs.

If no `name` parameter is provided, all available configs will be merged.

#### Web Interface

```
GET /
```

Provides a simple web interface to browse and download available kubeconfigs.

## Storage

Kubeconfig files are stored as YAML files in the configured `CONFIGS_DIR`. File names should have `.yaml` extension.
