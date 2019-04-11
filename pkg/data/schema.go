/*
 * Copyright (C) 2019-Present Pivotal Software, Inc. All rights reserved.
 *
 * This program and the accompanying materials are made available under the terms
 * of the Apache License, Version 2.0 (the "License”); you may not use this file
 * except in compliance with the License. You may obtain a copy of the License at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package data

// language=sql
var Schema = `create table if not exists scenario_runs
(
    id                                       integer primary key, -- aliases to rowid

    recorded                                 text        not null,

    origin                                   text        not null,

    traffic_pattern                          text        not null,

    cluster_launch_delay                     big integer not null,
    cluster_terminate_delay                  big integer not null,
    cluster_number_of_requests               big integer not null,

    autoscaler_tick_interval                 big integer not null,
    autoscaler_stable_window                 big integer not null,
    autoscaler_panic_window                  big integer not null,
    autoscaler_scale_to_zero_grace_period    big integer not null,
    autoscaler_target_concurrency_default    real        not null,
    autoscaler_target_concurrency_percentage real        not null,
    autoscaler_max_scale_up_rate             real        not null
);

create table if not exists stocks
(
    name            text primary key,
    kind_stocked    text    not null,

    scenario_run_id integer not null references scenario_runs (id)
);

create table if not exists entities
(
    name            text primary key,
    kind            text    not null,

    scenario_run_id integer not null references scenario_runs (id)
);

create table if not exists completed_movements
(
    occurs_at       unsigned big integer primary key, -- unsigned int to avoid being an alias to rowid
    kind            text    not null,
    moved           text    not null references entities (name),
    from_stock      text    not null references stocks (name),
    to_stock        text    not null references stocks (name),

    scenario_run_id integer not null references scenario_runs (id)
);

create table if not exists ignored_movements
(
    occurs_at       unsigned big integer primary key, -- unsigned int to avoid being an alias to rowid
    kind            text    not null,
    from_stock      text    not null references stocks (name),
    to_stock        text    not null references stocks (name),
    reason          text    not null,

    scenario_run_id integer not null references scenario_runs (id)
)
`
