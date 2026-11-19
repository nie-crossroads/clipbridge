package parser_test

import (
	"errors"
	"testing"

	"github.com/nie-crossroads/clipbridge/internal/domain"
	"github.com/nie-crossroads/clipbridge/internal/parser"
	"github.com/nie-crossroads/clipbridge/internal/parser/bilibili"
)

func TestShareLinkSelection(t *testing.T) {
	registry := parser.NewRegistry(bilibili.New())
	cases := []struct {
		name, input, filter, wantPath, wantError string
	}{
		{"share text", "复制链接 https://www.bilibili.com/video/BV1GJ411x7h7/?p=2）；打开观看", "", "/video/BV1GJ411x7h7/", ""},
		{"short URL", "https://b23.tv/FhXm130", "bilibili", "/FhXm130", ""},
		{"different filter", "https://b23.tv/FhXm130", "douyin", "", "UNSUPPORTED_URL"},
		{"suffix lookalike", "https://www.bilibili.com.example/video/BV1GJ411x7h7", "", "", "UNSUPPORTED_URL"},
		{"unrecognized subdomain", "https://other.bilibili.com/video/BV1GJ411x7h7", "", "", "UNSUPPORTED_URL"},
		{"credentials", "https://local@b23.tv/FhXm130", "", "", "INVALID_INPUT"},
		{"explicit port", "https://b23.tv:443/FhXm130", "", "", "INVALID_INPUT"},
		{"platform without adapter", "https://v.douyin.com/example/", "", "", "PLATFORM_UNAVAILABLE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adapter, address, _, err := registry.Select(tc.input, tc.filter)
			if tc.wantError != "" {
				var problem *domain.Problem
				if !errors.As(err, &problem) || problem.Code != tc.wantError {
					t.Fatalf("error = %v, want %s", err, tc.wantError)
				}
				return
			}
			if err != nil || adapter == nil || address.Path != tc.wantPath {
				t.Fatalf("address = %v, error = %v, want %s", address, err, tc.wantPath)
			}
		})
	}
}
