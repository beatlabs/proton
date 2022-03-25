package cmd

import (
	"testing"

	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

func TestParseOffsets(t *testing.T) {
	tests := []struct {
		name               string
		given              []string
		startTime, endTime int64
	}{
		{
			name:      "no offsets specified",
			given:     []string{},
			startTime: sarama.OffsetOldest,
			endTime:   sarama.OffsetNewest,
		},
		{
			name:      "start offset specified",
			given:     []string{"s@24"},
			startTime: 24,
			endTime:   sarama.OffsetNewest,
		},
		{
			name:      "end offset specified",
			given:     []string{"e@42"},
			startTime: sarama.OffsetOldest,
			endTime:   42,
		},
		{
			name:      "both offsets specified",
			given:     []string{"s@24", "e@42"},
			startTime: 24,
			endTime:   42,
		},
		{
			name:      "multiple offsets specified",
			given:     []string{"s@24", "e@42", "s@123", "e@321"},
			startTime: 24,
			endTime:   42,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given
			// when
			r1, r2 := parseOffsets(test.given)

			// then
			assert.Equal(t, test.startTime, r1)
			assert.Equal(t, test.endTime, r2)
		})
	}
}
