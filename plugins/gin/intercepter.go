// Licensed to Apache Software Foundation (ASF) under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Apache Software Foundation (ASF) licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package gin

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/apache/skywalking-go/plugins/core/operator"
	"github.com/apache/skywalking-go/plugins/core/tracing"
)

type HTTPInterceptor struct {
}

func (h *HTTPInterceptor) BeforeInvoke(invocation operator.Invocation) error {
	context := invocation.Args()[0].(*gin.Context)
	if traceIgnore("/fib/*", context.Request.RequestURI) || traceIgnore("/world", context.Request.RequestURI) {
		fmt.Println("Before  trace ignore,url = ", context.Request.RequestURI)
		return nil
	}
	s, err := tracing.CreateEntrySpan(
		fmt.Sprintf("%s:%s", context.Request.Method, context.Request.URL.Path), func(headerKey string) (string, error) {
			return context.Request.Header.Get(headerKey), nil
		},
		tracing.WithLayer(tracing.SpanLayerHTTP),
		tracing.WithTag(tracing.TagHTTPMethod, context.Request.Method),
		tracing.WithTag(tracing.TagURL, context.Request.Host+context.Request.URL.Path),
		tracing.WithComponent(5006))
	if err != nil {
		return err
	}
	invocation.SetContext(s)
	return nil
}

func (h *HTTPInterceptor) AfterInvoke(invocation operator.Invocation, result ...interface{}) error {
	context := invocation.Args()[0].(*gin.Context)
	if traceIgnore("/fib/*", context.Request.RequestURI) || traceIgnore("/world", context.Request.RequestURI) {
		fmt.Println("After  trace ignore,url = ", context.Request.RequestURI)
		return nil
	}
	if invocation.GetContext() == nil {
		return nil
	}
	span := invocation.GetContext().(tracing.Span)
	span.Tag(tracing.TagStatusCode, fmt.Sprintf("%d", context.Writer.Status()))
	if len(context.Errors) > 0 {
		span.Error(context.Errors.String())
	}
	span.End()
	return nil
}

func traceIgnore(pattern, url string) bool {
	pattern1 := `^\*.*\*$` // * 字符开头和结尾   *example*
	pattern2 := `.*\*$`    // * 字符开头   example*  ==>  ^example
	pattern3 := `^\*.*`    // * 字符结尾   *example  ==>  example$

	matched, err := regexp.MatchString(pattern1, pattern)
	if err == nil && matched {
		return strings.Contains(url, pattern[1:len(pattern)-1])
	}

	matched, err = regexp.MatchString(pattern2, pattern)
	if err == nil && matched {
		return strings.HasPrefix(url, pattern[:len(pattern)-1])
	}

	matched, err = regexp.MatchString(pattern3, pattern)
	if err == nil && matched {
		return strings.HasSuffix(url, pattern[1:])
	}

	return url == pattern
}
