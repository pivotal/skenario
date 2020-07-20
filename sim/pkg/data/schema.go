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

    simulated_duration                       big integer not null,

    origin                                   text        not null,

    traffic_pattern                          text        not null,

    cluster_launch_delay                     big integer not null,
    cluster_terminate_delay                  big integer not null,
    cluster_number_of_requests               big integer not null,

    autoscaler_tick_interval                 big integer not null
);

create table if not exists stocks
(
    id           integer primary key, -- aliases to rowid
    name         text not null,
    kind_stocked text not null
);
create unique index if not exists stocks_names_kind on stocks (name, kind_stocked);

create table if not exists entities
(
    id   integer primary key, -- aliases to rowid
    name text not null,
    kind text not null
);
create unique index if not exists entities_names_kind on entities (name, kind);

create table if not exists completed_movements
(
    id              integer primary key,  -- aliases to rowid
    occurs_at       unsigned big integer, -- unsigned int to avoid being an alias to rowid
    kind            text    not null,

    moved           integer    not null references entities (id),
    from_stock      integer    not null references stocks (id),
    to_stock        integer    not null references stocks (id),

    scenario_run_id integer not null references scenario_runs (id)
);

create table if not exists cpu_utilizations
(
	id 					integer primary key,
	cpu_utilization 	real 					not null,
	calculated_at 		unsigned big integer 	not null,

	scenario_run_id 	integer not null references scenario_runs (id)
);

create unique index if not exists move_once_per_run on completed_movements (occurs_at, scenario_run_id);

create table if not exists ignored_movements
(
    id              integer primary key,  -- aliases to rowid
    occurs_at       unsigned big integer, -- unsigned int to avoid being an alias to rowid
    kind            text    not null,

    from_stock      integer    not null references stocks (id),
    to_stock        integer    not null references stocks (id),
    reason          text       not null,

    scenario_run_id integer not null references scenario_runs (id)
);
create unique index if not exists ignore_once_per_run on ignored_movements (occurs_at, scenario_run_id);

create view if not exists stock_aggregate as
select id
     , (case
            when name like 'RequestsProcessing%' then 'RequestsProcessing'
            else name
    end) as name
     , (case
            when kind_stocked = 'Desired' then 'Replica'
            else kind_stocked
    end) as kind_stocked
from stocks
where kind_stocked in ('Request', 'Desired', 'Replica')
  and name not in ('TrafficSource', 'ReplicaSource', 'DesiredSource', 'DesiredSink', 'ReplicasLaunching', 'ReplicasTerminating', 'ReplicasTerminated')
  and name not like 'RequestsComplete%'
;
`
