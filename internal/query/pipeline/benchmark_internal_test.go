package pipeline

import (
	"fmt"
	"testing"
)

func BenchmarkQueryEntries(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		entries := generateTestEntries(size)

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "entries",
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkQueryMessages(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		entries := generateTestEntries(size)

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "messages",
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkQueryTools(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		entries := generateTestEntries(size)

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "tools",
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkQueryToolsWithFilter(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		entries := generateTestEntries(size)

		b.Run(fmt.Sprintf("size_%d_filter_name", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "tools",
				Filter: FilterSpec{
					ToolName: "Read",
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("size_%d_filter_status", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "tools",
				Filter: FilterSpec{
					ToolStatus: "error",
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("size_%d_filter_both", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "tools",
				Filter: FilterSpec{
					ToolName:   "Read",
					ToolStatus: "error",
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkQueryWithAggregate(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		entries := generateTestEntries(size)

		b.Run(fmt.Sprintf("size_%d_count", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "tools",
				Aggregate: AggregateSpec{
					Function: "count",
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("size_%d_count_by_field", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "tools",
				Aggregate: AggregateSpec{
					Function: "count",
					Field:    "tool_name",
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("size_%d_group_by_status", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "tools",
				Aggregate: AggregateSpec{
					Function: "count",
					Field:    "status",
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkQueryFilterAndAggregate(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		entries := generateTestEntries(size)

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "tools",
				Filter: FilterSpec{
					ToolStatus: "error",
				},
				Aggregate: AggregateSpec{
					Function: "count",
					Field:    "tool_name",
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkQueryUserMessages(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		entries := generateTestEntries(size)

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "messages",
				Filter: FilterSpec{
					Role: "user",
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkQueryAssistantMessages(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		entries := generateTestEntries(size)

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "messages",
				Filter: FilterSpec{
					Role: "assistant",
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkQueryBySession(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		entries := generateTestEntries(size)

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "entries",
				Filter: FilterSpec{
					SessionID: "session-5",
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkQueryByGitBranch(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		entries := generateTestEntries(size)

		b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
			params := QueryParams{
				Resource: "entries",
				Filter: FilterSpec{
					GitBranch: "feature/branch",
				},
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := Query(entries, params)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkResourceSelection(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		entries := generateTestEntries(size)

		b.Run(fmt.Sprintf("size_%d_entries", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := SelectResource(entries, "entries")
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("size_%d_messages", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := SelectResource(entries, "messages")
				if err != nil {
					b.Fatal(err)
				}
			}
		})

		b.Run(fmt.Sprintf("size_%d_tools", size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := SelectResource(entries, "tools")
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkFilterApplication(b *testing.B) {
	entries := generateTestEntries(1000)
	tools, _ := SelectResource(entries, "tools")

	b.Run("no_filter", func(b *testing.B) {
		filter := FilterSpec{}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ApplyFilter(tools, filter)
		}
	})

	b.Run("single_field_filter", func(b *testing.B) {
		filter := FilterSpec{
			ToolName: "Read",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ApplyFilter(tools, filter)
		}
	})

	b.Run("multi_field_filter", func(b *testing.B) {
		filter := FilterSpec{
			ToolName:   "Read",
			ToolStatus: "error",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ApplyFilter(tools, filter)
		}
	})
}

func BenchmarkAggregation(b *testing.B) {
	entries := generateTestEntries(1000)
	tools, _ := SelectResource(entries, "tools")

	b.Run("count_all", func(b *testing.B) {
		agg := AggregateSpec{
			Function: "count",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ApplyAggregate(tools, agg)
		}
	})

	b.Run("count_by_field", func(b *testing.B) {
		agg := AggregateSpec{
			Function: "count",
			Field:    "tool_name",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = ApplyAggregate(tools, agg)
		}
	})
}
