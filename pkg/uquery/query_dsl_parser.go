/* Copyright 2022 Zinc Labs Inc. and Contributors
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package uquery

import (
	"fmt"

	"github.com/strivewrt/bluge"
	"github.com/strivewrt/bluge/analysis"
	"github.com/strivewrt/bluge/search"

	"github.com/zincsearch/zincsearch/pkg/config"
	"github.com/zincsearch/zincsearch/pkg/errors"
	"github.com/zincsearch/zincsearch/pkg/meta"
	"github.com/zincsearch/zincsearch/pkg/uquery/aggregation"
	"github.com/zincsearch/zincsearch/pkg/uquery/fields"
	"github.com/zincsearch/zincsearch/pkg/uquery/highlight"
	"github.com/zincsearch/zincsearch/pkg/uquery/query"
	"github.com/zincsearch/zincsearch/pkg/uquery/sort"
	"github.com/zincsearch/zincsearch/pkg/uquery/source"
)

// ParseQueryDSL parse query DSL and return searchRequest
func ParseQueryDSL(q *meta.ZincQuery, mappings *meta.Mappings, analyzers map[string]*analysis.Analyzer) (bluge.SearchRequest, error) {
	// parse size
	if q.Size > config.Global.MaxResults {
		q.Size = config.Global.MaxResults
	}

	// parse query
	query, err := query.Query(q.Query, mappings, analyzers)
	if err != nil {
		return nil, err
	}
	if query == nil {
		return nil, errors.New(errors.ErrorTypeNotImplemented, fmt.Sprintf("[%s] query doesn't support", q.Query))
	}

	// create search request
	request := bluge.NewTopNSearch(q.Size, query).WithStandardAggregations()

	// parse highlight
	if q.Highlight != nil {
		_ = highlight.Request(q.Highlight)
		request.IncludeLocations()
	}

	// parse from
	if q.From > 0 {
		request.SetFrom(q.From)
	}

	// parse explain
	if q.Explain {
		request.ExplainScores()
	}

	// parse aggregations
	if q.Aggregations != nil {
		if err := aggregation.Request(request, q.Aggregations, mappings); err != nil {
			return nil, err
		}
	}

	// parse fields
	if q.Fields != nil {
		if v, ok := q.Fields.([]interface{}); ok {
			if q.Fields, err = fields.Request(v); err != nil {
				return nil, err
			}
		}
	}

	// parse source
	if q.Source, err = source.Request(q.Source); err != nil {
		return nil, err
	}

	// parse sort
	if q.Sort != nil {
		if q.Sort, err = sort.Request(q.Sort); err != nil {
			return nil, err
		}
		if q.Sort != nil {
			request.SortByCustom(q.Sort.(search.SortOrder))
		}
	}

	// pagenation
	// TODO: search after PIT support

	return request, nil
}
