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

package model

import (
	"fmt"
	"github.com/knative/serving/pkg/autoscaler"

	"skenario/pkg/simulator"
)

type ScraperTicktock interface {
	simulator.ThroughStock
}

type scraperTicktock struct {
	scraperEntity simulator.Entity
	collector     *autoscaler.MetricCollector
	scraper       *autoscaler.ServiceScraper
}

func (st *scraperTicktock) Name() simulator.StockName {
	return "Scraper Ticktock"
}

func (st *scraperTicktock) KindStocked() simulator.EntityKind {
	return "KnativeScraper"
}

func (st *scraperTicktock) Count() uint64 {
	return 1
}

func (st *scraperTicktock) EntitiesInStock() []*simulator.Entity {
	return nil
}

func (st *scraperTicktock) Remove() simulator.Entity {
	return st.scraperEntity
}

func (st *scraperTicktock) Add(entity simulator.Entity) error {
	if st.scraperEntity != entity {
		return fmt.Errorf("'%+v' is different from the entity given at creation time, '%+v'", entity, st.scraperEntity)
	}
	stat, err := st.scraper.Scrape()
	if err != nil {
		panic(fmt.Errorf("could not scrape: %s", err.Error()))
	}

	if stat != nil {
		st.collector.Record(stat.Key, stat.Stat)
	}

	return nil
}

func NewScraperTicktockStock(collector *autoscaler.MetricCollector, scraper *autoscaler.ServiceScraper) ScraperTicktock {
	return &scraperTicktock{
		scraperEntity: simulator.NewEntity("Scraper", "KnativeScraper"),
		collector:     collector,
		scraper:       scraper,
	}
}
