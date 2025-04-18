package pkg

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	//go:embed example/jwt.txt
	token string
	//go:embed example/slots.json
	slotsJson string
)

type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

func TestNewClient(t *testing.T) {
	type args struct {
		conf Config
		jwt  string
	}
	tests := []struct {
		name      string
		args      args
		wantErr   bool
		wantPanic bool
	}{
		{
			name: "Without JWT",
			args: args{
				conf: Config{},
				jwt:  "",
			},
			wantErr: true,
		},
		{
			name: "With expired JWT",
			args: args{
				conf: Config{},
				jwt:  token,
			},
			wantPanic: true,
		},
		{
			name: "With valid JWT",
			args: args{
				conf: Config{},
				jwt:  generateToken(Config{}),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantPanic {
				assert.Panics(t, func() { NewClient(tt.args.conf, tt.args.jwt) }, "NewClient() should panic")
			} else {
				_, err := NewClient(tt.args.conf, tt.args.jwt)
				assert.Equal(t, tt.wantErr, err != nil, "NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_client_Slots(t *testing.T) {

	c := client{
		conf:   Config{},
		JWT:    generateToken(Config{}),
		logger: log.New(log.Writer(), "test: ", log.LstdFlags),
		client: NewTestClient(func(req *http.Request) *http.Response {
			// Test request parameters
			return &http.Response{
				StatusCode: 200,
				// Send response to be tested
				Body: io.NopCloser(bytes.NewBufferString(slotsJson)),
				// Must be set to non-nil value or it panics
				Header: make(http.Header),
			}
		}),
	}

	var slots []Slot
	json.Unmarshal([]byte(slotsJson), &slots)

	tests := []struct {
		name    string
		client  *http.Client
		wantErr bool
	}{
		{
			name: "Have slots",
			client: NewTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString(slotsJson)),
					Header:     make(http.Header),
				}
			}),
			wantErr: false,
		},
		{
			name: "Empty slots",
			client: NewTestClient(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(bytes.NewBufferString("[]")),
					Header:     make(http.Header),
				}
			}),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Slots(context.Background(), "2025-03-03")

			assert.Equal(t, tt.wantErr, err != nil, "error = %v, wantErr %v", err, tt.wantErr)

			if err == nil {
				assert.IsType(t, []Slot{}, got, "Slots() got = %v, want %v", got, slots)
			}

			if len(got) > 0 {
				assert.IsType(t, Slot{}, got[0], "Slots() got = %v, want %v", got, slots)
			}
		})
	}
}
