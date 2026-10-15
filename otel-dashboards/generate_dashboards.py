#!/usr/bin/env python3
"""Generates the Grafana dashboards of the Kinetica OpenTelemetry exporter's
metric tables into otel-dashboards/dashboards. Edit the dashboards here rather
than their JSON, then regenerate them and run the tests, from the repository
root:

    python3 otel-dashboards/generate_dashboards.py
    go test ./otel-dashboards/

The dashboards read the series and point tables (metric_series,
metric_number_point, ...) of the latest Kinetica OpenTelemetry exporter, and
need the Kinetica datasource plugin (kinetica-grafana-datasource)
PLUGIN_MIN_VERSION or later, whose features they rely on:

- dashboard variables: the schema the exporter writes to, and multi-value
  query variables (service, route, host) read from metric_series;
- $__labelMatch(labels, 'key', ${var:json}), which filters a series' labels by
  a variable, with the variable's custom "All" value * meaning no filter;
- $__timeFilter and $__timeGroup(col, $__interval) for time ranges and buckets;
- time columns recognised by type, so a bucket can be called time anywhere;
- the time_series format, which makes one series per value of the text
  columns, so panels need no transformation.
"""
import json
import sys
from pathlib import Path

PLUGIN_ID = "kinetica-grafana-datasource"
# The plugin release with $__labelMatch, the time_series format and time
# columns recognised by type
PLUGIN_MIN_VERSION = "1.1.0"
GRAFANA_MIN_VERSION = "12.3.0"

DS = {"type": PLUGIN_ID, "uid": "${DS_KINETICA}"}
# The shortest time bucket: scrape intervals are rarely shorter, and a bucket
# without samples leaves a gap
MIN_INTERVAL = "1m"
HOST = "COALESCE(NULLIF(JSON_VALUE(s.labels, '$[\"host.name\"]'), ''), 'unknown')"
SERVICE = "COALESCE(s.service_name, 'unknown')"


def table(name):
    return f"$schema.{name}"


def time_group(column):
    return f"$__timeGroup({column}, $__interval)"


def label_match(key, variable):
    """A filter on a label of metric_series by a multi-value variable"""
    return f"$__labelMatch(labels, '{key}', ${{{variable}:json}})"


def lag(col, alias):
    return f"LAG({col}) OVER (PARTITION BY {alias}.series_id ORDER BY {alias}.time_unix)"


def increase(alias, col, guard=None):
    """The increase of a cumulative column over the previous sample of its
    series; the value itself after a reset, detected on the guard column; null
    for a series' first sample, which has no previous one."""
    guard = guard or col
    previous = lag(f'{alias}.{guard}', alias)
    return (f"CASE WHEN {previous} IS NULL THEN NULL\n"
            f"                     WHEN {alias}.{guard} >= {previous}\n"
            f"                     THEN {alias}.{col} - {lag(f'{alias}.{col}', alias)}\n"
            f"                     ELSE {alias}.{col} END")


def per_second(alias, col):
    # A LONG times a decimal literal is a DECIMAL(28,6): cast to DOUBLE first
    return (f"CAST({increase(alias, col)} AS DOUBLE) * 1000\n"
            f"               / (MSECS_SINCE_EPOCH({alias}.time_unix)\n"
            f"                  - {lag(f'MSECS_SINCE_EPOCH({alias}.time_unix)', alias)})")


def series_filter(metric, filters, extra="", temporality=None, cols="series_id, service_name, labels"):
    """The series of a metric, filtered by the dashboard's variables; a
    text-search function evaluates against a single table, so the filters are
    applied here, before the series are joined to their points"""
    conditions = [f"metric_name = '{metric}'"]
    if temporality is not None:
        conditions.append(f"temporality = {temporality}")
    conditions += filters
    if extra:
        conditions.append(extra)
    where = "\n                 AND ".join(conditions)
    return f"(SELECT {cols} FROM {table('metric_series')}\n               WHERE {where}) s"


def rate_query(point_table, alias, col, metric, filters, groups, value_alias, extra_filter="", outer=None, keep=None,
               having=""):
    """Per-second rate of a cumulative column, summed per time bucket for each
    group: a sample's rate is its increase over the time since the previous
    sample of its series; a series' rate in a bucket is the average of its
    samples' rates."""
    group_cols = ", ".join(name for name, _ in groups)
    group_exprs = ",\n               ".join(f"{expr} AS {name}" for name, expr in groups)
    select_outer = outer or f"SUM(series_rate) AS {value_alias}"
    outer_cols = ", ".join(keep) if keep else group_cols
    having_clause = f"\nHAVING {having}" if having else ""
    return f"""SELECT time, {outer_cols}, {select_outer}
FROM (
    SELECT {time_group('time_unix')} AS time, series_id, {group_cols},
           AVG(sample_rate) AS series_rate
    FROM (
        SELECT {alias}.series_id, {alias}.time_unix,
               {group_exprs},
               {per_second(alias, col)} AS sample_rate
        FROM {table(point_table)} {alias}
        JOIN {series_filter(metric, filters, extra_filter, temporality=2)}
          ON s.series_id = {alias}.series_id
        WHERE $__timeFilter({alias}.time_unix)
    ) d
    WHERE sample_rate IS NOT NULL
    GROUP BY {time_group('time_unix')}, series_id, {group_cols}
) b
GROUP BY time, {outer_cols}{having_clause}
ORDER BY time, {outer_cols}"""


def gauge_query(metric, filters, groups, value_alias, extra_filter="", aggregate="SUM"):
    """A gauge or non-monotonic sum: each series averaged per time bucket, then
    aggregated across series per group."""
    group_cols = ", ".join(name for name, _ in groups)
    group_exprs = ",\n           ".join(f"{expr} AS {name}" for name, expr in groups)
    group_by = ", ".join(expr for _, expr in groups)
    return f"""SELECT time, {group_cols}, {aggregate}(series_value) AS {value_alias}
FROM (
    SELECT {time_group('p.time_unix')} AS time,
           p.series_id,
           {group_exprs},
           AVG(p.value) AS series_value
    FROM {table('metric_number_point')} p
    JOIN {series_filter(metric, filters, extra_filter)}
      ON s.series_id = p.series_id
    WHERE $__timeFilter(p.time_unix)
    GROUP BY {time_group('p.time_unix')}, p.series_id, {group_by}
) b
GROUP BY time, {group_cols}
ORDER BY time, {group_cols}"""


def schema_variable():
    return {
        "name": "schema", "label": "Schema", "type": "textbox", "query": "otel", "hide": 0,
        "current": {"text": "otel", "value": "otel"},
        "options": [{"selected": True, "text": "otel", "value": "otel"}],
        "description": "The schema the Kinetica exporter writes its metric tables to",
    }


def query_variable(name, label, sql, description):
    """A multi-value variable read from metric_series, whose All value * makes
    $__labelMatch filter nothing"""
    return {
        "name": name, "label": label, "type": "query", "description": description,
        "datasource": DS, "query": {"refId": f"{name}-values", "rawSql": sql}, "definition": sql,
        "multi": True, "includeAll": True, "allValue": "*",
        "current": {"text": ["All"], "value": ["$__all"]}, "options": [],
        "refresh": 1, "sort": 1, "regex": "", "hide": 0,
    }


def label_values_sql(key, where):
    """The values of a label across the series a condition selects"""
    value = f"JSON_VALUE(labels, '$[\"{key}\"]')"
    return (f"SELECT DISTINCT {value} AS value FROM {table('metric_series')} "
            f"WHERE {where} AND {value} <> '' ORDER BY value")


class Dashboard:
    def __init__(self, uid, title, description, intro, variables):
        self.uid, self.title, self.description = uid, title, description
        self.variables = variables
        self.panels = []
        self.y = 0
        self.x = 0
        self.add_text(intro)

    def _place(self, w, h):
        if self.x + w > 24:
            self.x, self.y = 0, self.y + self.row_h
        pos = {"x": self.x, "y": self.y, "w": w, "h": h}
        self.x += w
        self.row_h = h
        if self.x >= 24:
            self.x, self.y = 0, self.y + h
        return pos

    def add_text(self, content, h=4):
        self.row_h = h
        self.panels.append({
            "id": len(self.panels) + 1, "type": "text", "title": "", "gridPos": self._place(24, h),
            "options": {"mode": "markdown", "content": content},
        })

    def add_timeseries(self, title, description, sql, unit, w=12, h=8, stack=False):
        defaults = {"unit": unit, "custom": {"fillOpacity": 10, "showPoints": "never", "spanNulls": True}}
        if stack:
            defaults["custom"]["stacking"] = {"mode": "normal", "group": "A"}
        self.panels.append({
            "id": len(self.panels) + 1, "type": "timeseries", "title": title, "description": description,
            "datasource": DS, "gridPos": self._place(w, h), "interval": MIN_INTERVAL,
            "fieldConfig": {"defaults": defaults, "overrides": []},
            "options": {"legend": {"displayMode": "list", "placement": "bottom", "showLegend": True},
                        "tooltip": {"mode": "multi", "sort": "desc"}},
            "targets": [{"datasource": DS, "refId": "A", "format": "time_series", "rawSql": sql}],
        })

    def add_table(self, title, description, sql, w=12, h=8, overrides=None):
        self.panels.append({
            "id": len(self.panels) + 1, "type": "table", "title": title, "description": description,
            "datasource": DS, "gridPos": self._place(w, h),
            "fieldConfig": {"defaults": {}, "overrides": overrides or []},
            "options": {"showHeader": True, "cellHeight": "sm"},
            "targets": [{"datasource": DS, "refId": "A", "format": "table", "rawSql": sql}],
        })

    def json(self):
        return {
            "__requires": [
                {"type": "grafana", "id": "grafana", "name": "Grafana", "version": GRAFANA_MIN_VERSION},
                {"type": "datasource", "id": PLUGIN_ID, "name": "Kinetica", "version": PLUGIN_MIN_VERSION},
            ],
            "annotations": {"list": []},
            "description": self.description,
            "editable": True,
            "fiscalYearStartMonth": 0,
            "graphTooltip": 1,
            "links": [],
            "panels": self.panels,
            "refresh": "1m",
            "schemaVersion": 39,
            "tags": ["opentelemetry", "kinetica", "metrics"],
            "templating": {"list": [{
                "current": {}, "hide": 0, "includeAll": False, "label": "Kinetica", "multi": False,
                "name": "DS_KINETICA", "options": [], "query": PLUGIN_ID, "refresh": 1,
                "regex": "", "skipUrlSync": False, "type": "datasource",
            }, schema_variable()] + self.variables},
            "time": {"from": "now-6h", "to": "now"},
            "timepicker": {},
            "timezone": "",
            "title": self.title,
            "uid": self.uid,
            "version": 1,
            "weekStart": "",
        }


def unit_override(field, unit):
    return {"matcher": {"id": "byName", "options": field}, "properties": [{"id": "unit", "value": unit}]}


REQUIREMENTS = (f"Needs the Kinetica datasource plugin {PLUGIN_MIN_VERSION} or later, and the series and point tables "
                "(`metric_series`, `metric_*_point`) the latest Kinetica OpenTelemetry exporter writes, in the schema "
                "chosen above.")
COMMON_NOTE = (REQUIREMENTS + " Rates assume cumulative temporality "
               "(the default of most SDKs; for delta, run the collector's `deltatocumulative` processor). "
               f"Each series is averaged per time bucket, which follows the zoom level, {MIN_INTERVAL} at least.")


def http_dashboard():
    duration = "http.server.request.duration"
    service = label_match("service.name", "service")
    route = label_match("http.route", "route")
    d = Dashboard(
        "kinetica-otel-http", "OTel metrics on Kinetica / HTTP services",
        "HTTP server metrics of the OpenTelemetry semantic conventions, from the Kinetica exporter's metric tables.",
        "### HTTP services\nRequest rate, errors and duration per service, from `http.server.request.duration` "
        "(a histogram, in seconds) and `http.server.active_requests`. " + COMMON_NOTE,
        [
            query_variable(
                "service", "Service",
                f"SELECT DISTINCT service_name FROM {table('metric_series')} "
                f"WHERE metric_name = '{duration}' AND service_name IS NOT NULL ORDER BY service_name",
                "The services with HTTP server metrics"),
            query_variable(
                "route", "Route",
                label_values_sql("http.route", f"metric_name = '{duration}' AND {service}"),
                "The routes of the selected services"),
        ])

    d.add_timeseries(
        "Requests per second", "Observations of http.server.request.duration per second, per service.",
        rate_query("metric_histogram_point", "h", "data_count", duration, [service, route], [("service", SERVICE)],
                   "requests_per_second"),
        "reqps")
    d.add_timeseries(
        "5xx responses per second",
        "Requests answered with a 5xx status, found with text_match on the series' labels.",
        rate_query("metric_histogram_point", "h", "data_count", duration, [service, route], [("service", SERVICE)],
                   "errors_per_second",
                   extra_filter="text_match(labels, 'labels.http.response.status_code:/5../')"),
        "reqps")
    d.add_timeseries(
        "Error ratio", "Share of requests answered with a 5xx status, per service.",
        rate_query("metric_histogram_point", "h", "data_count", duration, [service, route],
                   [("service", SERVICE),
                    ("is_error", "CASE WHEN JSON_VALUE(s.labels, '$[\"http.response.status_code\"]') LIKE '5%' "
                                 "THEN 1 ELSE 0 END")],
                   "error_ratio",
                   outer="SUM(CASE WHEN is_error = 1 THEN series_rate ELSE 0 END) / SUM(series_rate) AS error_ratio",
                   keep=["service"], having="SUM(series_rate) > 0"),
        "percentunit")
    d.add_timeseries(
        "Mean request duration",
        "Increase of the histogram's sum over the increase of its count, per service.",
        f"""SELECT time, service, SUM(sum_increase) / SUM(count_increase) AS mean_duration
FROM (
    SELECT {time_group('h.time_unix')} AS time,
           {SERVICE} AS service,
           {increase('h', 'data_sum', guard='data_count')} AS sum_increase,
           {increase('h', 'data_count')} AS count_increase
    FROM {table('metric_histogram_point')} h
    JOIN {series_filter(duration, [service, route], temporality=2)}
      ON s.series_id = h.series_id
    WHERE $__timeFilter(h.time_unix)
) d
WHERE count_increase > 0
GROUP BY time, service
ORDER BY time, service""",
        "s")
    # http.server.active_requests has no route, so it is filtered by service only
    d.add_timeseries(
        "Active requests", "http.server.active_requests, summed over each service's series. Not filtered by route, "
                           "which the metric does not carry.",
        gauge_query("http.server.active_requests", [service], [("service", SERVICE)], "active_requests"),
        "short")
    d.add_table(
        "Busiest routes",
        "Requests, 5xx ratio and mean duration per service and route over the time range.",
        f"""SELECT service, route,
       SUM(count_increase) AS requests,
       CAST(SUM(CASE WHEN is_error = 1 THEN count_increase ELSE 0 END) AS DOUBLE) / SUM(count_increase) AS error_ratio,
       SUM(sum_increase) / SUM(count_increase) AS mean_duration
FROM (
    SELECT {SERVICE} AS service,
           COALESCE(NULLIF(JSON_VALUE(s.labels, '$["http.route"]'), ''), '(none)') AS route,
           CASE WHEN JSON_VALUE(s.labels, '$["http.response.status_code"]') LIKE '5%' THEN 1 ELSE 0 END AS is_error,
           {increase('h', 'data_sum', guard='data_count')} AS sum_increase,
           {increase('h', 'data_count')} AS count_increase
    FROM {table('metric_histogram_point')} h
    JOIN {series_filter(duration, [service, route], temporality=2)}
      ON s.series_id = h.series_id
    WHERE $__timeFilter(h.time_unix)
) d
WHERE count_increase > 0
GROUP BY service, route
ORDER BY requests DESC
LIMIT 25""",
        overrides=[unit_override("error_ratio", "percentunit"), unit_override("mean_duration", "s")])
    d.add_table(
        "Slowest exemplars",
        "The exemplars of http.server.request.duration with the largest values; look their trace_id up in "
        "the trace tables.",
        f"""SELECT e.time_unix AS point_time_unix, e.value AS duration, {SERVICE} AS service,
       JSON_VALUE(s.labels, '$["http.route"]') AS route, e.trace_id, e.span_id
FROM {table('metric_exemplar')} e
JOIN {series_filter(duration, [service, route])}
  ON s.series_id = e.series_id
WHERE $__timeFilter(e.time_unix)
ORDER BY e.value DESC
LIMIT 25""",
        overrides=[unit_override("duration", "s")])
    return d


def hosts_dashboard():
    host = label_match("host.name", "host")
    d = Dashboard(
        "kinetica-otel-hosts", "OTel metrics on Kinetica / Hosts",
        "Host metrics of the collector's hostmetrics receiver, from the Kinetica exporter's metric tables.",
        "### Hosts\nCPU, memory, filesystem, network and disk metrics of the collector's `hostmetrics` "
        "receiver, per `host.name` (set by the `resourcedetection` processor). " + COMMON_NOTE,
        [
            query_variable(
                "host", "Host", label_values_sql("host.name", "metric_name LIKE 'system.%'"),
                "The hosts with host metrics"),
        ])

    d.add_timeseries(
        "CPU utilization", "Share of CPU time not idle, across all CPUs of each host (system.cpu.time).",
        rate_query("metric_number_point", "p", "value", "system.cpu.time", [host],
                   [("host", HOST), ("state", "JSON_VALUE(s.labels, '$[\"state\"]')")], "cpu_utilization",
                   outer="SUM(CASE WHEN state <> 'idle' THEN series_rate ELSE 0 END) / SUM(series_rate) "
                         "AS cpu_utilization",
                   keep=["host"], having="SUM(series_rate) > 0"),
        "percentunit")
    d.add_timeseries(
        "Load average (1m)", "system.cpu.load_average.1m per host.",
        gauge_query("system.cpu.load_average.1m", [host], [("host", HOST)], "load_average", aggregate="AVG"),
        "short")
    d.add_timeseries(
        "Memory used", "system.memory.usage in the used state, found with text_match on the series' labels.",
        gauge_query("system.memory.usage", [host], [("host", HOST)], "used_bytes",
                    extra_filter="text_match(labels, 'labels.state:\"used\"')"),
        "bytes")
    d.add_timeseries(
        "Filesystem utilization", "Used share of each filesystem (system.filesystem.usage), per host and mountpoint.",
        f"""SELECT time, host, mountpoint,
       SUM(CASE WHEN state = 'used' THEN series_value ELSE 0 END) / SUM(series_value) AS utilization
FROM (
    SELECT {time_group('p.time_unix')} AS time,
           p.series_id,
           {HOST} AS host,
           JSON_VALUE(s.labels, '$["mountpoint"]') AS mountpoint,
           JSON_VALUE(s.labels, '$["state"]') AS state,
           AVG(p.value) AS series_value
    FROM {table('metric_number_point')} p
    JOIN {series_filter("system.filesystem.usage", [host])}
      ON s.series_id = p.series_id
    WHERE $__timeFilter(p.time_unix)
    GROUP BY {time_group('p.time_unix')}, p.series_id, {HOST},
             JSON_VALUE(s.labels, '$["mountpoint"]'), JSON_VALUE(s.labels, '$["state"]')
) b
GROUP BY time, host, mountpoint
HAVING SUM(series_value) > 0
ORDER BY time, host, mountpoint""",
        "percentunit")
    d.add_timeseries(
        "Network throughput", "Bytes per second per host and direction (system.network.io).",
        rate_query("metric_number_point", "p", "value", "system.network.io", [host],
                   [("host", HOST), ("direction", "JSON_VALUE(s.labels, '$[\"direction\"]')")],
                   "bytes_per_second"),
        "Bps")
    d.add_timeseries(
        "Disk throughput", "Bytes per second per host and direction (system.disk.io).",
        rate_query("metric_number_point", "p", "value", "system.disk.io", [host],
                   [("host", HOST), ("direction", "JSON_VALUE(s.labels, '$[\"direction\"]')")],
                   "bytes_per_second"),
        "Bps")
    return d


def inventory_dashboard():
    service = label_match("service.name", "service")
    d = Dashboard(
        "kinetica-otel-inventory", "OTel metrics on Kinetica / Inventory",
        "What the Kinetica exporter has written to its metric tables: metrics, series, labels and ingest volume.",
        "### Inventory\nThe metrics, series and labels in the metric tables, and the points written. "
        + REQUIREMENTS + " The service filter applies to "
        "everything but the points written, which the point tables do not record per service.",
        [
            query_variable(
                "service", "Service",
                f"SELECT DISTINCT service_name FROM {table('metric_series')} "
                "WHERE service_name IS NOT NULL ORDER BY service_name",
                "The services with metrics"),
        ])

    d.add_table(
        "Metrics", "Every metric, with its number of series and services.",
        f"""SELECT metric_name, metric_type, unit,
       COUNT(*) AS series,
       COUNT(DISTINCT service_name) AS services,
       MIN(first_seen) AS first_seen
FROM {table('metric_series')}
WHERE {service}
GROUP BY metric_name, metric_type, unit
ORDER BY series DESC, metric_name""",
        w=24, h=10)
    d.add_table(
        "Series per service", "Series and metrics of each service (service.name).",
        f"""SELECT COALESCE(service_name, '(none)') AS service,
       COUNT(*) AS series,
       COUNT(DISTINCT metric_name) AS metric_names
FROM {table('metric_series')}
WHERE {service}
GROUP BY COALESCE(service_name, '(none)')
ORDER BY series DESC""")
    d.add_table(
        "Label keys", "Every label key, with the number of series having it: the cardinality to watch.",
        f"""SELECT label, COUNT(*) AS series
FROM UNNEST_JSON_OBJECT(SELECT labels FROM {table('metric_series')} WHERE {service}) t(label, label_value)
GROUP BY label
ORDER BY series DESC, label""")
    points = "\nUNION ALL\n".join(
        f"""SELECT {time_group('time_unix')} AS time, '{name}' AS point_table, COUNT(*) AS points
FROM {table(point_table)}
WHERE $__timeFilter(time_unix)
GROUP BY {time_group('time_unix')}"""
        for name, point_table in [("number", "metric_number_point"), ("histogram", "metric_histogram_point"),
                                  ("exp_histogram", "metric_exp_histogram_point"),
                                  ("summary", "metric_summary_point")])
    d.add_timeseries("Points written", "Rows written to each point table per time bucket, for every service.",
                     points, "short", stack=True)
    d.add_timeseries(
        "New series per hour", "Series seen for the first time, per metric type.",
        f"""SELECT $__timeGroup(first_seen, 1h) AS time, metric_type, COUNT(*) AS new_series
FROM {table('metric_series')}
WHERE $__timeFilter(first_seen)
  AND {service}
GROUP BY $__timeGroup(first_seen, 1h), metric_type
ORDER BY time""",
        "short", stack=True)
    return d


def main(out_dir):
    out = Path(out_dir)
    out.mkdir(parents=True, exist_ok=True)
    for dashboard, name in [(http_dashboard(), "otel-metrics-http.json"),
                            (hosts_dashboard(), "otel-metrics-hosts.json"),
                            (inventory_dashboard(), "otel-metrics-inventory.json")]:
        (out / name).write_text(json.dumps(dashboard.json(), indent=2) + "\n")
        print("wrote", out / name, len(dashboard.panels), "panels")


if __name__ == "__main__":
    main(sys.argv[1] if len(sys.argv) > 1 else str(Path(__file__).parent / "dashboards"))
