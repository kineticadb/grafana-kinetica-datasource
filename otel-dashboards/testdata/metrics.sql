-- The metric tables of the Kinetica OpenTelemetry exporter, as its
-- sql/datamodel.sql creates them, and metrics whose dashboard results are
-- known, for TestDashboardsLive. Replace {{schema}} to load it by hand.
--
-- The metrics, from 2026-10-12 10:00:00 UTC (m0) and a minute later (m1):
--
--   - checkout on host h1: http.server.request.duration, a cumulative
--     histogram, on /api/v1/cart with status 200 (count 10 then 30, sum 1
--     then 5) and 500 (count 0 then 10, sum 0 then 3, an exemplar of 0.9),
--     and on /api/v1/items/{id} with status 200 (count 100 then 160, sum 10
--     then 16); http.server.active_requests, 4 then 6
--   - checkout-v2 and Checkout: http.server.active_requests, 1 at m0
--   - host h1, with no service: system.cpu.time, idle 0 then 45 and user 0
--     then 15; system.cpu.load_average.1m, 1.5 then 2.5; system.memory.usage,
--     used 1000 and free 3000; system.filesystem.usage on /, used 30 and free
--     70; system.network.io received 0 then 6000; system.disk.io written 0
--     then 12000
--   - host h2, with no service: system.cpu.load_average.1m, 0.5 then 0.7
--
-- The exporter hashes a series' identity into its series_id; here they are
-- simply numbered.

CREATE TABLE {{schema}}.metric_series
(
    series_id LONG NOT NULL,
    metric_name VARCHAR (256, dict) NOT NULL,
    metric_type VARCHAR (16, dict) NOT NULL,
    unit VARCHAR (64, dict),
    description VARCHAR (512),
    temporality INT8,
    is_monotonic BOOLEAN,
    service_name VARCHAR (128, dict),
    scope_name VARCHAR (256, dict),
    scope_version VARCHAR (64, dict),
    labels JSON (text_search),
    resource_attributes JSON,
    scope_attributes JSON,
    attributes JSON,
    first_seen TIMESTAMP NOT NULL,
    PRIMARY KEY (series_id),
    SHARD KEY (series_id)
)
TEXT INDEX (labels) WITH OPTIONS (analyzer = 'keyword')
USING TABLE PROPERTIES (NO_ERROR_IF_EXISTS = TRUE);

CREATE TABLE {{schema}}.metric_number_point
(
    series_id LONG NOT NULL,
    time_unix TIMESTAMP NOT NULL,
    start_time_unix TIMESTAMP,
    value DOUBLE NOT NULL,
    flags INT,
    PRIMARY KEY (series_id, time_unix),
    SHARD KEY (series_id)
) USING TABLE PROPERTIES (NO_ERROR_IF_EXISTS = TRUE);

CREATE TABLE {{schema}}.metric_histogram_point
(
    series_id LONG NOT NULL,
    time_unix TIMESTAMP NOT NULL,
    start_time_unix TIMESTAMP,
    data_count LONG NOT NULL,
    data_sum DOUBLE,
    data_min DOUBLE,
    data_max DOUBLE,
    explicit_bounds JSON,
    bucket_counts JSON,
    flags INT,
    PRIMARY KEY (series_id, time_unix),
    SHARD KEY (series_id)
) USING TABLE PROPERTIES (NO_ERROR_IF_EXISTS = TRUE);

CREATE TABLE {{schema}}.metric_exp_histogram_point
(
    series_id LONG NOT NULL,
    time_unix TIMESTAMP NOT NULL,
    start_time_unix TIMESTAMP,
    data_count LONG NOT NULL,
    data_sum DOUBLE,
    data_min DOUBLE,
    data_max DOUBLE,
    scale INT NOT NULL,
    zero_count LONG,
    zero_threshold DOUBLE,
    positive_offset INT,
    positive_bucket_counts JSON,
    negative_offset INT,
    negative_bucket_counts JSON,
    flags INT,
    PRIMARY KEY (series_id, time_unix),
    SHARD KEY (series_id)
) USING TABLE PROPERTIES (NO_ERROR_IF_EXISTS = TRUE);

CREATE TABLE {{schema}}.metric_summary_point
(
    series_id LONG NOT NULL,
    time_unix TIMESTAMP NOT NULL,
    start_time_unix TIMESTAMP,
    data_count LONG NOT NULL,
    data_sum DOUBLE,
    quantiles JSON,
    quantile_values JSON,
    flags INT,
    PRIMARY KEY (series_id, time_unix),
    SHARD KEY (series_id)
) USING TABLE PROPERTIES (NO_ERROR_IF_EXISTS = TRUE);

CREATE TABLE {{schema}}.metric_exemplar
(
    series_id LONG NOT NULL,
    point_time_unix TIMESTAMP NOT NULL,
    time_unix TIMESTAMP NOT NULL,
    value DOUBLE,
    trace_id VARCHAR (32),
    span_id VARCHAR (16),
    filtered_attributes JSON,
    SHARD KEY (series_id)
) USING TABLE PROPERTIES (NO_ERROR_IF_EXISTS = TRUE);

-- Labels are the resource and data point attributes, every value a string;
-- temporality 2 is cumulative, 0 a gauge's
INSERT INTO {{schema}}.metric_series
    (series_id, metric_name, metric_type, unit, temporality, is_monotonic, service_name,
     labels, resource_attributes, scope_attributes, attributes, first_seen)
VALUES
    (1, 'http.server.request.duration', 'histogram', 's', 2, 0, 'checkout',
     '{"host.name":"h1","http.response.status_code":"200","http.route":"/api/v1/cart","service.name":"checkout"}',
     '{"host.name":"h1","service.name":"checkout"}', '{}',
     '{"http.response.status_code":200,"http.route":"/api/v1/cart"}', '2026-10-12 10:00:00'),
    (2, 'http.server.request.duration', 'histogram', 's', 2, 0, 'checkout',
     '{"host.name":"h1","http.response.status_code":"500","http.route":"/api/v1/cart","service.name":"checkout"}',
     '{"host.name":"h1","service.name":"checkout"}', '{}',
     '{"http.response.status_code":500,"http.route":"/api/v1/cart"}', '2026-10-12 10:00:00'),
    (3, 'http.server.request.duration', 'histogram', 's', 2, 0, 'checkout',
     '{"host.name":"h1","http.response.status_code":"200","http.route":"/api/v1/items/{id}","service.name":"checkout"}',
     '{"host.name":"h1","service.name":"checkout"}', '{}',
     '{"http.response.status_code":200,"http.route":"/api/v1/items/{id}"}', '2026-10-12 10:00:00'),
    (4, 'http.server.active_requests', 'sum', '{request}', 2, 0, 'checkout',
     '{"host.name":"h1","service.name":"checkout"}',
     '{"host.name":"h1","service.name":"checkout"}', '{}', '{}', '2026-10-12 10:00:00'),
    (5, 'http.server.active_requests', 'sum', '{request}', 2, 0, 'checkout-v2',
     '{"service.name":"checkout-v2"}', '{"service.name":"checkout-v2"}', '{}', '{}', '2026-10-12 10:00:00'),
    (6, 'http.server.active_requests', 'sum', '{request}', 2, 0, 'Checkout',
     '{"service.name":"Checkout"}', '{"service.name":"Checkout"}', '{}', '{}', '2026-10-12 10:00:00'),
    (7, 'system.cpu.time', 'sum', 's', 2, 1, NULL,
     '{"cpu":"cpu0","host.name":"h1","state":"idle"}', '{"host.name":"h1"}', '{}',
     '{"cpu":"cpu0","state":"idle"}', '2026-10-12 10:00:00'),
    (8, 'system.cpu.time', 'sum', 's', 2, 1, NULL,
     '{"cpu":"cpu0","host.name":"h1","state":"user"}', '{"host.name":"h1"}', '{}',
     '{"cpu":"cpu0","state":"user"}', '2026-10-12 10:00:00'),
    (9, 'system.cpu.load_average.1m', 'gauge', '{thread}', 0, 0, NULL,
     '{"host.name":"h1"}', '{"host.name":"h1"}', '{}', '{}', '2026-10-12 10:00:00'),
    (10, 'system.memory.usage', 'sum', 'By', 2, 0, NULL,
     '{"host.name":"h1","state":"used"}', '{"host.name":"h1"}', '{}', '{"state":"used"}', '2026-10-12 10:00:00'),
    (11, 'system.memory.usage', 'sum', 'By', 2, 0, NULL,
     '{"host.name":"h1","state":"free"}', '{"host.name":"h1"}', '{}', '{"state":"free"}', '2026-10-12 10:00:00'),
    (12, 'system.filesystem.usage', 'sum', 'By', 2, 0, NULL,
     '{"host.name":"h1","mountpoint":"/","state":"used"}', '{"host.name":"h1"}', '{}',
     '{"mountpoint":"/","state":"used"}', '2026-10-12 10:00:00'),
    (13, 'system.filesystem.usage', 'sum', 'By', 2, 0, NULL,
     '{"host.name":"h1","mountpoint":"/","state":"free"}', '{"host.name":"h1"}', '{}',
     '{"mountpoint":"/","state":"free"}', '2026-10-12 10:00:00'),
    (14, 'system.network.io', 'sum', 'By', 2, 1, NULL,
     '{"device":"eth0","direction":"receive","host.name":"h1"}', '{"host.name":"h1"}', '{}',
     '{"device":"eth0","direction":"receive"}', '2026-10-12 10:00:00'),
    (15, 'system.disk.io', 'sum', 'By', 2, 1, NULL,
     '{"device":"sda","direction":"write","host.name":"h1"}', '{"host.name":"h1"}', '{}',
     '{"device":"sda","direction":"write"}', '2026-10-12 10:00:00'),
    (16, 'system.cpu.load_average.1m', 'gauge', '{thread}', 0, 0, NULL,
     '{"host.name":"h2"}', '{"host.name":"h2"}', '{}', '{}', '2026-10-12 10:00:00');

INSERT INTO {{schema}}.metric_histogram_point
    (series_id, time_unix, start_time_unix, data_count, data_sum, explicit_bounds, bucket_counts, flags)
VALUES
    (1, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 10, 1, '[0.1,0.5,1]', '[5,3,1,1]', 0),
    (1, '2026-10-12 10:01:00', '2026-10-12 09:00:00', 30, 5, '[0.1,0.5,1]', '[15,9,4,2]', 0),
    (2, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 0, 0, '[0.1,0.5,1]', '[0,0,0,0]', 0),
    (2, '2026-10-12 10:01:00', '2026-10-12 09:00:00', 10, 3, '[0.1,0.5,1]', '[0,5,4,1]', 0),
    (3, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 100, 10, '[0.1,0.5,1]', '[60,30,8,2]', 0),
    (3, '2026-10-12 10:01:00', '2026-10-12 09:00:00', 160, 16, '[0.1,0.5,1]', '[96,48,13,3]', 0);

INSERT INTO {{schema}}.metric_exemplar
    (series_id, point_time_unix, time_unix, value, trace_id, span_id, filtered_attributes)
VALUES
    (2, '2026-10-12 10:01:00', '2026-10-12 10:01:00', 0.9, 'ab000000000000000000000000000001', 'cd00000000000001', '{}');

-- Gauges and the series of other services have no start time
INSERT INTO {{schema}}.metric_number_point (series_id, time_unix, start_time_unix, value, flags)
VALUES
    (4, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 4, 0),
    (4, '2026-10-12 10:01:00', '2026-10-12 09:00:00', 6, 0),
    (5, '2026-10-12 10:00:00', NULL, 1, 0),
    (6, '2026-10-12 10:00:00', NULL, 1, 0),
    (7, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 0, 0),
    (7, '2026-10-12 10:01:00', '2026-10-12 09:00:00', 45, 0),
    (8, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 0, 0),
    (8, '2026-10-12 10:01:00', '2026-10-12 09:00:00', 15, 0),
    (9, '2026-10-12 10:00:00', NULL, 1.5, 0),
    (9, '2026-10-12 10:01:00', NULL, 2.5, 0),
    (10, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 1000, 0),
    (11, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 3000, 0),
    (12, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 30, 0),
    (13, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 70, 0),
    (14, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 0, 0),
    (14, '2026-10-12 10:01:00', '2026-10-12 09:00:00', 6000, 0),
    (15, '2026-10-12 10:00:00', '2026-10-12 09:00:00', 0, 0),
    (15, '2026-10-12 10:01:00', '2026-10-12 09:00:00', 12000, 0),
    (16, '2026-10-12 10:00:00', NULL, 0.5, 0),
    (16, '2026-10-12 10:01:00', NULL, 0.7, 0);
