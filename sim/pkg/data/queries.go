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
var RunningTallyQuery = `
with running_tally as (
select
	  occurs_at
	, sa.name            as stock_name
	, sa.kind_stocked
	, sum(case
		   when from_stock = to_stock then 0
		   when from_stock = sa.id then -1
		   when to_stock = sa.id then 1
		 end)
	  over summation as tally
	from completed_movements join stock_aggregate sa on sa.id in (from_stock, to_stock)
	where kind not in ('start_to_running', 'autoscaler_tick', 'running_to_halted', 'metrics_tick', 'send_metrics_to_pipeline', 'send_metrics_to_sink')
	and scenario_run_id = ?
    window summation as (partition by sa.name order by occurs_at asc rows unbounded preceding)
)
select occurs_at
     , stock_name
     , kind_stocked
     , tally
from running_tally
union
select -- fake up final values so that plot of replicas doesn't clip
    simulated_duration as occurs_at
     , stock_name
     , kind_stocked
     , tally
from running_tally, scenario_runs
where scenario_runs.id = ?
  and kind_stocked = 'Replica'
group by stock_name
having max(occurs_at)
order by occurs_at asc, stock_name asc
;
`

// language=sql
var ResponseTimesQuery = `
select
    min(occurs_at) as arrived_at
  , max(occurs_at) as completed_at
  , max(occurs_at) - min(occurs_at) as response_time
from completed_movements
where moved in (select id from entities where entities.kind = 'Request')
  and scenario_run_id = ?
group by moved
order by arrived_at
;
`

// language=sql
var CPUUtilizationQuery = `
select
    max(cpu_utilization)
  , calculated_at
from cpu_utilizations
where scenario_run_id = ?
group by calculated_at
order by calculated_at
;
`

// language=sql
var RequestsPerSecondQuery = `
select
    occurs_at / 1000000000        as occurs_at_second
  , count(occurs_at / 1000000000) as arrivals
from completed_movements
where kind = 'arrive_at_routing_stock'
and scenario_run_id = ?
group by occurs_at_second
;
`
