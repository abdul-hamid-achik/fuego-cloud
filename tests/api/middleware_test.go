package api_test

import (
	"sync"
	"testing"

	"github.com/abdul-hamid-achik/fuego-cloud/app/api"
)

// TestRateLimiter tests the RateLimiter type directly
func TestRateLimiter(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		limiter := api.NewRateLimiter(10, 20) // 10 per second, burst 20

		for i := 0; i < 10; i++ {
			if !limiter.Allow("192.168.1.1") {
				t.Errorf("request %d should be allowed", i)
			}
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		limiter := api.NewRateLimiter(1, 2) // 1 per second, burst 2

		// Use up the burst
		limiter.Allow("192.168.1.2")
		limiter.Allow("192.168.1.2")

		// Third request should be blocked
		if limiter.Allow("192.168.1.2") {
			t.Error("expected request to be blocked after exceeding burst")
		}
	})

	t.Run("tracks different IPs separately", func(t *testing.T) {
		limiter := api.NewRateLimiter(1, 1) // 1 per second, burst 1

		// First IP uses its quota
		limiter.Allow("192.168.1.3")
		if limiter.Allow("192.168.1.3") {
			t.Error("second request from same IP should be blocked")
		}

		// Second IP should still be allowed
		if !limiter.Allow("192.168.1.4") {
			t.Error("first request from different IP should be allowed")
		}
	})

	t.Run("is thread-safe", func(t *testing.T) {
		limiter := api.NewRateLimiter(1000, 2000)

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(ip string) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					limiter.Allow(ip)
				}
			}(string(rune('A' + i%26)))
		}

		wg.Wait()
		// If we get here without panic, the test passes
	})

	t.Run("allows burst requests", func(t *testing.T) {
		limiter := api.NewRateLimiter(1, 10) // 1 per second, burst 10

		ip := "192.168.1.5"

		// Should allow all requests within burst
		for i := 0; i < 10; i++ {
			if !limiter.Allow(ip) {
				t.Errorf("request %d within burst should be allowed", i)
			}
		}
	})
}

// TestRateLimiterEdgeCases tests edge cases for the rate limiter
func TestRateLimiterEdgeCases(t *testing.T) {
	t.Run("handles empty IP", func(t *testing.T) {
		limiter := api.NewRateLimiter(10, 20)

		// Should not panic with empty IP
		if !limiter.Allow("") {
			t.Error("first request with empty IP should be allowed")
		}
	})

	t.Run("handles very long IP", func(t *testing.T) {
		limiter := api.NewRateLimiter(10, 20)

		longIP := "192.168.1.1:12345:some:extra:data:that:is:very:long"
		if !limiter.Allow(longIP) {
			t.Error("first request with long IP should be allowed")
		}
	})

	t.Run("handles IPv6 addresses", func(t *testing.T) {
		limiter := api.NewRateLimiter(10, 20)

		ipv6 := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
		if !limiter.Allow(ipv6) {
			t.Error("first request with IPv6 should be allowed")
		}
	})
}
