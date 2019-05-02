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
var RunningCountQuery = `
select
    occurs_at
  , kind
  , (select name from entities where id = moved) moved

  , sum(case
      when exists(select id from stocks where name = 'RequestsBuffered' and id = from_stock and id = to_stock) then 0
      when exists(select id from stocks where name = 'RequestsBuffered' and id = from_stock) then -1
      when exists(select id from stocks where name = 'RequestsBuffered' and id = to_stock) then 1
      else 0 end)
    over summation as requests_buffered
  , sum(case
      when exists(select id from stocks where name like 'RequestsProcessing%' and id = from_stock) then -1
      when exists(select id from stocks where name like 'RequestsProcessing%' and id = to_stock) then 1
      else 0 end)
    over summation as requests_processing
  , sum(case
      when exists(select id from stocks where name like 'RequestsComplete%' and id = from_stock) then -1
      when exists(select id from stocks where name like 'RequestsComplete%' and id = to_stock) then 1
      else 0 end)
    over summation as requests_completed
  , sum(case (select id from stocks where name    = 'ReplicasDesired'   ) when from_stock then -1   when to_stock then 1  else 0 end) over summation as replicas_desired
  , sum(case (select id from stocks where name    = 'ReplicasLaunching'   ) when from_stock then -1   when to_stock then 1  else 0 end) over summation as replicas_launching
  , sum(case (select id from stocks where name    = 'ReplicasActive'      ) when from_stock then -1   when to_stock then 1  else 0 end) over summation as replicas_active
  , sum(case (select id from stocks where name    = 'ReplicasTerminated'  ) when from_stock then -1   when to_stock then 1  else 0 end) over summation as replicas_terminated

from completed_movements
where scenario_run_id = ?
and kind not in ('start_to_running', 'autoscaler_tick', 'running_to_halted')
window summation as (order by occurs_at asc rows unbounded preceding)
order by occurs_at
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