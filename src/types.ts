import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface SurrealQuery extends DataQuery {
  rawSql: string;
}

export const DEFAULT_QUERY: Partial<SurrealQuery> = {
  rawSql: 'SELECT * FROM surreal LIMIT 10',
};

/** The level at which a SurrealDB system user authenticates. */
export type SurrealAuthScope = 'root' | 'namespace' | 'database';

/**
 * These are options configured for each DataSource instance
 */
export interface SurrealDataSourceOptions extends DataSourceJsonData {
  database?: string;
  endpoint?: string;
  namespace?: string;
  username?: string;
  /** Authentication level for the system user (defaults to root). */
  authScope?: SurrealAuthScope;
  /** SurrealDB access method, used for record-level authentication. */
  access?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface SurrealSecureJsonData {
  password?: string;
}
