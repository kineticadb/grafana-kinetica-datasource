import { KineticaQueryBuilder, KineticaFilter } from './types';

// Helper: Clean name for comparison (strip quotes)
const cleanName = (name: string) => {
    if (!name) {return '';}
    const parts = name.split('.');
    return parts[parts.length - 1].replace(/"/g, '');
};

// Helper: Smart Quoting
const quoteId = (val: string) => {
    if (!val) {return '';}
    if (val.startsWith('"') || val.startsWith("'") || !isNaN(parseFloat(val))) {return val;} 
    if (val.includes('.')) {
        return val.split('.').map(p => `"${p}"`).join('.');
    }
    return `"${val}"`;
};

function generateSingleSelect(builder: KineticaQueryBuilder): string {
  if (!builder.table) {
    return '';
  }

  // --- 0. RESOLVE ALIAS MAP ---
  const aliasMap = new Map<string, string>();
  
  const mainTableName = cleanName(builder.table);
  const mainTableAlias = builder.alias || mainTableName;
  if (mainTableAlias !== mainTableName) {
      aliasMap.set(mainTableName, mainTableAlias);
  }

  if (builder.joins) {
      builder.joins.forEach(j => {
          if (j.table) {
              const jName = cleanName(j.table);
              const jAlias = j.alias || jName;
              if (jAlias !== jName && !aliasMap.has(jName)) {
                  aliasMap.set(jName, jAlias);
              }
          }
      });
  }

  const resolveAlias = (identifier: string): string => {
      if (!identifier || !identifier.includes('.')) {return identifier;}
      const parts = identifier.split('.');
      const ref = parts[0].replace(/"/g, '');
      if (aliasMap.has(ref)) {
          return `${aliasMap.get(ref)}.${parts.slice(1).join('.')}`;
      }
      return identifier;
  };

  // --- 1. SELECT ---
  const distinctClause = builder.distinct ? 'DISTINCT ' : '';

  let select = '*';

  if (builder.selects && builder.selects.length > 0) {
    // Filter out selects with empty column names
    const validSelects = builder.selects.filter(s => s.column && s.column.trim() !== '');

    if (validSelects.length > 0) {
      select = validSelects.map(s => {
         const colName = `"${s.column}"`;
         let ref = s.table;
         if (ref && aliasMap.has(ref)) { ref = aliasMap.get(ref); }

         const tableName = ref ? `"${ref}".` : '';
         const fullyQualified = `${tableName}${colName}`;

         let expr = fullyQualified;
         if (s.aggregate) { expr = `${s.aggregate}(${fullyQualified})`; }
         if (s.alias) { expr += ` AS "${s.alias}"`; }
         return expr;
      }).join(', ');
    }
  } else if (builder.columns && builder.columns.length > 0) {
    select = builder.columns.map(c => `"${c}"`).join(', ');
  }

  // --- 2. FROM ---
  let finalSchema = builder.schema;
  let finalTable = builder.table;
  if (finalSchema && finalTable.startsWith(`${finalSchema}.`)) {
     finalTable = finalTable.substring(finalSchema.length + 1);
  }
  
  let from = `"${finalSchema}"."${finalTable}"`;
  if (builder.alias) { from += ` AS "${builder.alias}"`; }

  // --- 3. JOINS ---
  if (builder.joins && builder.joins.length > 0) {
      builder.joins.forEach(j => {
          if (j.table) {
              const type = j.type || 'JOIN';
              let jSchema = j.schema || finalSchema;
              let jTable = j.table;
              if (jSchema && jTable.startsWith(`${jSchema}.`)) {
                  jTable = jTable.substring(jSchema.length + 1);
              }
              const quotedJoinTable = `"${jSchema}"."${jTable}"`;
              const alias = j.alias ? ` AS "${j.alias}"` : '';
              
              let onClause = '';
              if (j.conditions && j.conditions.length > 0) {
                  const exprs = j.conditions.map((c, i) => {
                      const logic = i === 0 ? '' : ` ${c.logic} `;
                      const left = quoteId(resolveAlias(c.left));
                      const right = quoteId(resolveAlias(c.right));
                      return `${logic}${left} ${c.operator} ${right}`;
                  });
                  onClause = ` ON ${exprs.join('')}`;
              } else if (j.on) {
                  onClause = ` ON ${j.on}`;
              }
              from += ` ${type} JOIN ${quotedJoinTable}${alias}${onClause}`;
          }
      });
  }

  // --- 4. WHERE / HAVING helpers ---
  const buildConditions = (filters?: KineticaFilter[], timeCol?: string): string[] => {
    const exprs: string[] = [];
    if (timeCol) {
        exprs.push(`$__timeFilter(${quoteId(resolveAlias(timeCol))})`);
    }
    if (filters && filters.length > 0) {
      filters.forEach((f, i) => {
        if (f.key && f.operator && f.value) {
          const isNumber = !isNaN(parseFloat(f.value));
          const val = isNumber ? f.value : `'${f.value}'`;
          const quotedKey = quoteId(resolveAlias(f.key));
          
          let logic = 'AND';
          if (f.logic) { logic = f.logic; }
          
          if (exprs.length > 0) { exprs.push(`${logic} ${quotedKey} ${f.operator} ${val}`); } 
          else { exprs.push(`${quotedKey} ${f.operator} ${val}`); }
        }
      });
    }
    return exprs;
  };

  const whereExprs = buildConditions(builder.filters, builder.timeColumn);
  const whereClause = whereExprs.length > 0 ? `WHERE ${whereExprs.join(' ')}` : '';

  let groupByClause = '';
  if (builder.groupBy && builder.groupBy.length > 0) {
    const groups = builder.groupBy.map(c => quoteId(resolveAlias(c))).join(', ');
    groupByClause = `GROUP BY ${groups}`;
  }

  const havingExprs = buildConditions(builder.having);
  const havingClause = havingExprs.length > 0 ? `HAVING ${havingExprs.join(' ')}` : '';

  let orderByClause = '';
  if (builder.orderBy && builder.orderBy.length > 0) {
    const orders = builder.orderBy.map(o => `${quoteId(resolveAlias(o.column))} ${o.direction}`).join(', ');
    orderByClause = `ORDER BY ${orders}`;
  }

  let limitClause = '';
  if (builder.limit) {
    limitClause = (builder.offset && builder.offset > 0) 
        ? `LIMIT ${builder.offset}, ${builder.limit}` 
        : `LIMIT ${builder.limit}`;
  } else {
      if (!builder.setOperations || builder.setOperations.length === 0) {
         limitClause = 'LIMIT 1000';
      }
  }

  // Change: Join with space instead of newline to ensure single-line output
  return [
    `SELECT ${distinctClause}${select}`,
    `FROM ${from}`,
    whereClause,
    groupByClause,
    havingClause,
    orderByClause,
    limitClause
  ].filter(Boolean).join(' ');
}

export function generateSql(builder: KineticaQueryBuilder): string {
  const baseSql = generateSingleSelect(builder);
  if (!baseSql) {return '';}

  if (builder.setOperations && builder.setOperations.length > 0) {
    const setSqls = builder.setOperations.map(op => {
        const subQuerySql = generateSql(op.query); 
        if (!subQuerySql) {return '';}
        return `${op.operator} ${subQuerySql}`;
    }).filter(Boolean);

    if (setSqls.length > 0) {
        // Change: Join set operations with space instead of newline
        return [baseSql, ...setSqls].join(' ');
    }
  }

  return baseSql;
}
