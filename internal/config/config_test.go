package config

import (
	"reflect"
	"testing"
)

func TestLoad_PanicsWithoutJWTSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected Load to panic when JWT_SECRET is empty")
		}
	}()
	Load()
}

func TestLoad_UsesJWTSecretFromEnv(t *testing.T) {
	t.Setenv("JWT_SECRET", "from-env")
	if got := Load().JWT.SecretKey; got != "from-env" {
		t.Fatalf("SecretKey = %q, want from-env", got)
	}
}

func TestLoad_CORSAllowedOrigins(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want []string
	}{
		{"default dev origins", "", []string{"http://localhost:4325", "http://localhost:4322"}},
		{"comma list trimmed", " https://kinetix.events , https://admin.kinetix.events ,", []string{"https://kinetix.events", "https://admin.kinetix.events"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", "x")
			t.Setenv("CORS_ALLOWED_ORIGINS", tt.env)
			if got := Load().Server.CORSAllowedOrigins; !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("CORSAllowedOrigins = %#v, want %#v", got, tt.want)
			}
		})
	}
}
