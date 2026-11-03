package spec

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

func generateLargeSpec(operationsCount int) []byte {
	var sb strings.Builder
	sb.WriteString(`openapi: 3.1.0
info:
  title: Benchmark Massive OpenAPI Specification
  version: 2.0.0
servers:
  - url: http://localhost:4010
paths:
`)

	for i := 1; i <= operationsCount; i++ {
		sb.WriteString(fmt.Sprintf(`  /resource_%d:
    get:
      summary: Retrieve resource %d
      operationId: getResource%d
      responses:
        '200':
          description: Successful operation
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Entity%d'
`, i, i, i, (i%20)+1))
	}

	sb.WriteString("components:\n  schemas:\n")
	for i := 1; i <= 20; i++ {
		sb.WriteString(fmt.Sprintf(`    Entity%d:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        score:
          type: integer
          minimum: 0
          maximum: 100
`, i))
	}

	return []byte(sb.String())
}

func BenchmarkLargeSpecParsing(b *testing.B) {
	data := generateLargeSpec(250) // 250 endpoints + 20 schemas ~ 3,500 lines of OpenAPI
	parser := NewParser()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseBytes(data)
		if err != nil {
			b.Fatalf("failed to parse: %v", err)
		}
	}
}

func TestParsingPerformance(t *testing.T) {
	data := generateLargeSpec(300)
	parser := NewParser()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	start := time.Now()
	spec, err := parser.ParseBytes(data)
	elapsed := time.Since(start)

	runtime.ReadMemStats(&m2)
	memAllocMB := float64(m2.Alloc-m1.Alloc) / (1024 * 1024)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(spec.Endpoints) != 300 {
		t.Errorf("expected 300 endpoints, got %d", len(spec.Endpoints))
	}

	t.Logf("Parsed %d operations in %v (Allocated memory: %.2f MB)", len(spec.Endpoints), elapsed, memAllocMB)
}
