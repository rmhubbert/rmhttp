package rmhttp

import (
	"net/http/httptest"
	"testing"
)

// This file benchmarks CaptureWriter performance.
// Run with: go test -bench=Benchmark_CaptureWriter -benchmem ./...

func Benchmark_CaptureWriter_Write_SmallChunks(b *testing.B) {
	w := httptest.NewRecorder()
	chunk := make([]byte, 64) // 64-byte chunks

	b.ResetTimer()
	for b.Loop() {
		cw := NewCaptureWriter(w)
		for range 1000 { // 64KB total
			_, _ = cw.Write(chunk)
		}
		_ = cw.Body()
	}
}

func Benchmark_CaptureWriter_Write_MediumChunks(b *testing.B) {
	w := httptest.NewRecorder()
	chunk := make([]byte, 1024) // 1KB chunks

	b.ResetTimer()
	for b.Loop() {
		cw := NewCaptureWriter(w)
		for range 1000 { // 1MB total
			_, _ = cw.Write(chunk)
		}
		_ = cw.Body()
	}
}

func Benchmark_CaptureWriter_Write_LargeChunks(b *testing.B) {
	w := httptest.NewRecorder()
	chunk := make([]byte, 8192) // 8KB chunks

	b.ResetTimer()
	for b.Loop() {
		cw := NewCaptureWriter(w)
		for range 100 { // 800KB total
			_, _ = cw.Write(chunk)
		}
		_ = cw.Body()
	}
}

func Benchmark_CaptureWriter_Write_SingleChunk(b *testing.B) {
	w := httptest.NewRecorder()
	chunk := make([]byte, 1024*1024) // 1MB single chunk

	b.ResetTimer()
	for b.Loop() {
		cw := NewCaptureWriter(w)
		_, _ = cw.Write(chunk)
		_ = cw.Body()
	}
}

func Benchmark_CaptureWriter_PassThroughFalse(b *testing.B) {
	w := httptest.NewRecorder()
	chunk := make([]byte, 1024)

	b.ResetTimer()
	for b.Loop() {
		cw := NewCaptureWriter(w)
		cw.PassThrough = false
		for range 1000 {
			_, _ = cw.Write(chunk)
		}
	}
}

func Benchmark_CaptureWriter_WriteHeader(b *testing.B) {
	w := httptest.NewRecorder()

	b.ResetTimer()
	for b.Loop() {
		cw := NewCaptureWriter(w)
		cw.WriteHeader(200)
	}
}
