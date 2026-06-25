import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface SurrealQuery extends DataQuery {
  rawSql: string;
}

export const DEFAULT_QUERY: Partial<SurrealQuery> = {
  rawSql: 'SELECT * FROM surreal LIMIT 10',
};

/**
 * These are options configured for each DataSource instance
 */
export interface SurrealDataSourceOptions extends DataSourceJsonData {
  database?: string;
  endpoint?: string;
  namespace?: string;
  username?: string;
  /** SurrealDB 2.x/3.x access method, used for record-level authentication. */
  access?: string;
  /**
   * @deprecated SCOPE was removed in SurrealDB 3.0 in favour of `access`. Kept
   * so configurations saved by older plugin versions still load; it is ignored.
   */
  scope?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface SurrealSecureJsonData {
  password?: string;
}
