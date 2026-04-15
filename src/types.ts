import { DataSourceJsonData } from '@grafana/data';
import { DataQuery } from '@grafana/schema';

export interface KineticaQuery extends DataQuery {
  rawSql: string;
  builder?: KineticaQueryBuilder; 
}

export const defaultQuery: Partial<KineticaQuery> = {
  rawSql: 'SELECT * FROM ITER',
};

export interface KineticaDataSourceOptions extends DataSourceJsonData {
  url?: string;
  username?: string;
}

export interface KineticaSecureJsonData {
  password?: string;
}

export interface KineticaSelect {
  column: string;
  table?: string;
  aggregate?: string; 
  alias?: string;     
}

export interface KineticaFilter {
  key: string;
  operator: string;
  value: string;
  logic?: 'AND' | 'OR'; // New: Supports chaining filters
}

export interface KineticaOrderBy {
  column: string;
  direction: 'ASC' | 'DESC';
}

// New: Structured Join Condition
export interface KineticaJoinCondition {
    logic: 'AND' | 'OR';
    left: string;
    operator: string;
    right: string;
}

export interface KineticaJoin {
  type: string;       
  schema: string;     
  table: string;      
  alias?: string;     
  on?: string; // Legacy support
  conditions?: KineticaJoinCondition[]; // New: List of conditions
}

export type KineticaSetOperator = 'UNION' | 'UNION ALL' | 'INTERSECT' | 'INTERSECT ALL' | 'EXCEPT' | 'EXCEPT ALL';

export interface KineticaSetOperation {
  operator: KineticaSetOperator;
  query: KineticaQueryBuilder;
}

export interface KineticaQueryBuilder {
  schema: string;
  table: string;
  alias?: string;
  
  selects?: KineticaSelect[]; 
  columns?: string[]; 

  distinct?: boolean;
  
  joins?: KineticaJoin[];

  timeColumn?: string;
  filters?: KineticaFilter[];
  
  groupBy?: string[];
  having?: KineticaFilter[];
  
  orderBy?: KineticaOrderBy[];
  limit?: number;
  offset?: number;

  setOperations?: KineticaSetOperation[];
}
