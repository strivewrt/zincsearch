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

package query

import (
	"fmt"
	"strings"

	"github.com/strivewrt/bluge"
	"github.com/strivewrt/bluge/analysis"

	"github.com/zincsearch/zincsearch/pkg/errors"
	"github.com/zincsearch/zincsearch/pkg/meta"
	zincanalysis "github.com/zincsearch/zincsearch/pkg/uquery/analysis"
	"github.com/zincsearch/zincsearch/pkg/zutils"
)

func MultiMatchQuery(query map[string]interface{}, mappings *meta.Mappings, analyzers map[string]*analysis.Analyzer) (bluge.Query, error) {
	value := new(meta.MultiMatchQuery)
	value.Boost = -1.0
	for k, v := range query {
		k := strings.ToLower(k)
		switch k {
		case "query":
			value.Query = v.(string)
		case "analyzer":
			value.Analyzer = v.(string)
		case "fields":
			if vv, ok := v.([]interface{}); ok {
				for _, vvv := range vv {
					value.Fields = append(value.Fields, vvv.(string))
				}
			}
		case "boost":
			value.Boost = v.(float64)
		case "type":
			value.Type = v.(string)
		case "operator":
			value.Operator = v.(string)
		case "minimum_should_match":
			value.MinimumShouldMatch = v
		default:
			// return nil, errors.New(errors.ErrorTypeParsingException, fmt.Sprintf("[multi_match] unknown field [%s]", k))
		}
	}

	var zer *analysis.Analyzer
	if value.Analyzer != "" {
		zer, _ = zincanalysis.QueryAnalyzer(analyzers, value.Analyzer)
	}

	var operator bluge.MatchQueryOperator = bluge.MatchQueryOperatorOr
	if value.Operator != "" {
		op := strings.ToUpper(value.Operator)
		switch op {
		case "OR":
			operator = bluge.MatchQueryOperatorOr
		case "AND":
			operator = bluge.MatchQueryOperatorAnd
		default:
			return nil, errors.New(errors.ErrorTypeIllegalArgumentException, fmt.Sprintf("[multi_match] unknown operator %s", op))
		}
	}

	subq := bluge.NewBooleanQuery()
	if value.MinimumShouldMatch != nil {
		minValue, err := zutils.CalculateMin(len(value.Fields), value.MinimumShouldMatch)
		if err != nil {
			return nil, errors.New(errors.ErrorTypeXContentParseException, fmt.Sprintf("[multi_match] unsupported MinimumShouldMatch value: %v", err))
		}
		subq.SetMinShould(minValue) // lgtm[go/hardcoded-credentials]
	}
	if value.Boost >= 0 {
		subq.SetBoost(value.Boost)
	}
	for _, field := range value.Fields {
		subqq := bluge.NewMatchQuery(value.Query).SetField(field).SetOperator(operator)
		if zer != nil {
			subqq.SetAnalyzer(zer)
		} else {
			indexZer, searchZer := zincanalysis.QueryAnalyzerForField(analyzers, mappings, field)
			if zer == nil && searchZer != nil {
				zer = searchZer
			}
			if zer == nil && indexZer != nil {
				zer = indexZer
			}
			subqq.SetAnalyzer(zer)
		}
		subq.AddShould(subqq)
	}

	return subq, nil
}
