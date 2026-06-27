# SurrealDB data source plugin

The SurrealDB datasource plugin enables you to query and visualize SurrealDB data directly within Grafana, offering seamless integration and exploration of SurrealDB datasets.

## SurrealDB version compatibility

This version of the datasource targets **SurrealDB 3.x** (it connects using the SurrealDB Go SDK v1.x over the WebSocket RPC protocol). Use a SurrealDB 3.x server for full functionality.

## ⚠️ This plugin is currently experimental

This means that while we believe in its potential and are enthusiastic about its development, **we are not yet ready to make a long-term commitment to maintaining it indefinitely**. The plugin is still under active development and may contain bugs. We do not recommend using this plugin in production environments.

## What are Grafana data source plugins?

Grafana supports a wide range of data sources, including Prometheus, MySQL, and even Datadog. There’s a good chance you can already visualize metrics from the systems you have set up. In some cases, though, you already have an in-house metrics solution that you’d like to add to your Grafana dashboards. Grafana Data Source Plugins enables integrating such solutions with Grafana.

## Usage

### Installation

Please refer to our [Data Source Management documentation](https://grafana.com/docs/grafana/latest/administration/data-source-management/) for more information on installing the plugin to an instance of Grafana.

### Configuration

[Add a data source](https://grafana.com/docs/grafana/latest/datasources/add-a-data-source/) by filling in the following fields:

### Basic fields

| Field           | Description                                                                                                       |
| --------------- | ----------------------------------------------------------------------------------------------------------------- |
| Endpoint URL    | The **full** address of the SurrealDB RPC endpoint to connect to, e.g. `ws://localhost:8000/rpc`                  |
| Namespace       | The [namespace](https://docs.surrealdb.com/docs/surrealql/statements/define/namespace) to use for the connection. |
| Database name   | The name of the database to connect to.                                                                           |

### Authentication fields

| Field                | Description                                                                                                     |
| -------------------- | --------------------------------------------------------------------------------------------------------------- |
| Authentication level | The level the system user authenticates at: **Root** (default), **Namespace**, or **Database**. See below.     |
| Username             | Your SurrealDB username                                                                                         |
| Password             | Your SurrealDB password                                                                                         |
| Access               | The [access method](https://surrealdb.com/docs/surrealql/statements/define/access) to use for record-level authentication. (Optional) |

SurrealDB determines the authentication level from the credentials you sign in with, so choose the **Authentication level** that matches your user:

- **Root** — a root user (`DEFINE USER ... ON ROOT`). Signs in with only username/password and can query any namespace/database.
- **Namespace** — a namespace user (`DEFINE USER ... ON NAMESPACE`). Signs in scoped to the configured **Namespace**.
- **Database** — a database user (`DEFINE USER ... ON DATABASE`). Signs in scoped to the configured **Namespace** and **Database**. Recommended for least-privilege access.

For [record/scope access](https://surrealdb.com/docs/surrealql/statements/define/access), leave the level as Root and set the **Access** field to your access method name. In every case the configured Namespace and Database are selected for your queries.

**We strongly recommend that you make your queries with a user account that has read-only access.** This practice not only safeguards your data but also helps maintain system integrity.

### Querying

The query editor allows you to write SurrealQL queries. For more information about writing SurrealQL queries, please refer to [SurrealDB's documentation](https://docs.surrealdb.com/docs/surrealql/overview).

In this version, only a SurrealQL Editor is provided to write queries with. A Query Builder UI is planned for a later version of the plugin.

#### Time macros

The following macros are expanded before a query is sent to SurrealDB, so dashboard time ranges can drive your queries. They emit SurrealDB datetime literals (`d'...'`):

| Macro | Expands to |
| ----- | ---------- |
| `$__timeFrom(col)` | `col >= d'<from>'` |
| `$__timeTo(col)` | `col <= d'<to>'` |
| `$__timeFilter(col)` | `col >= d'<from>' AND col <= d'<to>'` |
| `$__timeGroup(col, unit)` | `time::group(col, 'unit')` |

For example, a time-series query might look like:

```surql
SELECT time::group(created_at, 'hour') AS time, count() AS value
FROM events
WHERE $__timeFilter(created_at)
GROUP BY time
ORDER BY time
```

## Development

This project requires **at least Node.js v20** and **at least Go 1.23**.

Version management configuration for Node.js is provided for [`volta`](https://volta.sh/). It is recommended that you have this installed to automatically switch between Node.js versions when you enter the project directory. This allows for more deterministic and reproducible builds, which makes debugging easier.

You use `volta` to configure the project to use the latest LTS version of Node.js by running:

```bash
volta pin node@lts
```

You can run this command again to update the version.

### Getting started

#### Backend

1. Update [Grafana plugin SDK for Go](https://grafana.com/docs/grafana/latest/developers/plugins/backend/grafana-plugin-sdk-for-go/) dependency to the latest minor version:

   ```bash
   go get -u github.com/grafana/grafana-plugin-sdk-go
   go mod tidy
   ```

2. Build backend plugin binaries for Linux, Windows and Darwin:

   ```bash
   mage -v
   ```

3. List all available Mage targets for additional commands:

   ```bash
   mage -l
   ```

#### Frontend

1. Install dependencies

   ```bash
   npm install
   ```

2. Build plugin in development mode and run in watch mode

   ```bash
   npm run dev
   ```

3. Build plugin in production mode

   ```bash
   npm run build
   ```

4. Run the tests (using Jest)

   ```bash
   # Runs the tests and watches for changes, requires git init first
   npm run test

   # Exits after running all the tests
   npm run test:ci
   ```

5. Spin up a Grafana instance and run the plugin inside it (using Docker)

   ```bash
   npm run server
   ```

6. Run the E2E tests (using Playwright)

   ```bash
   # Spins up a Grafana instance first that we tests against
   npm run server

   # Starts the tests
   npm run e2e
   ```

7. Run the linter

   ```bash
   npm run lint

   # or

   npm run lint:fix
   ```
