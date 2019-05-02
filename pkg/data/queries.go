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
 *
 */

package data

// language=sql
var RunningTallyQuery = `select occurs_at
     , stock_name
     , kind_stocked
     , tally
from running_tallies
where scenario_run_id = ?
union
select -- fake up final values so that plot of replicas doesn't clip
    simulated_duration as occurs_at
     , stock_name
     , kind_stocked
     , tally
from running_tallies
         join scenario_runs on running_tallies.scenario_run_id = scenario_runs.id
where scenario_run_id = ?
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
var RequestsPerSecondQuery = `
select
    occurs_at / 1000000000        as occurs_at_second
  , count(occurs_at / 1000000000) as arrivals
from completed_movements
where kind = 'arrive_at_buffer'
and scenario_run_id = ?
group by occurs_at_second
;
`