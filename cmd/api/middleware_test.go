package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func addAuthorization(
	t *testing.T,
	request *http.Request,
	tokenMaker Maker,
	authorizationType string,
	username string,
	duration time.Duration,
) {
	token, payload, err := tokenMaker.CreateToken(username, duration)
	if err != nil {
		t.Fatal(err)
	}
	if payload == nil {
		t.Fatal("payload is nil")
	}

	authorizationHeader := fmt.Sprintf("%s %s", authorizationType, token)
	request.Header.Set(string(authorizationHeaderKey), authorizationHeader)
}
func TestMiddlewareAuthWithRequestHeader(t *testing.T) {
	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "should pass with valid token",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker Maker) {
				addAuthorization(t, request, tokenMaker, "Bearer", "user", time.Hour)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				if recorder.Code != http.StatusOK {
					t.Fatalf("expected status code %d, got %d", http.StatusOK, recorder.Code)
				}
			},
		},
		{
			name: "should fail with invalid token",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker Maker) {
				addAuthorization(t, request, tokenMaker, "Bearer", "user", time.Hour)
				request.Header.Set(string(authorizationHeaderKey), "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMSJ9.c7SpmftjdwaJH6gNkoyxrjxgTrX9tXgWK3ZZ8mAvJIY")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				if recorder.Code != http.StatusUnauthorized {
					t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, recorder.Code)
				}
			},
		},
		{
			name: "should fail with missing token",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker Maker) {
				addAuthorization(t, request, tokenMaker, "Bearer", "user", time.Hour)
				request.Header.Del(string(authorizationHeaderKey))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				if recorder.Code != http.StatusUnauthorized {
					t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, recorder.Code)
				}
			},
		},
		{
			name: "should fail with invalid token payload",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker Maker) {
				addAuthorization(t, request, tokenMaker, "Bearer", "user", time.Hour)
				request.Header.Set(string(authorizationHeaderKey), "Basic invalid-token")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				if recorder.Code != http.StatusUnauthorized {
					t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, recorder.Code)
				}
			},
		},
		{
			name: "should fail with expired token",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker Maker) {
				addAuthorization(t, request, tokenMaker, "Bearer", "user", time.Microsecond)
				time.Sleep(time.Second)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				if recorder.Code != http.StatusUnauthorized {
					t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, recorder.Code)
				}
			},
		},
		{
			name: "should fail with invalid token type",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker Maker) {
				addAuthorization(t, request, tokenMaker, "Bearer", "user", time.Hour)
				request.Header.Set(string(authorizationHeaderKey), "Basic invalid-token")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				if recorder.Code != http.StatusUnauthorized {
					t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, recorder.Code)
				}
			},
		},
		{
			name: "invalid authorization header",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker Maker) {
				request.Header.Set(string(authorizationHeaderKey), "invalid-token")
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				if recorder.Code != http.StatusUnauthorized {
					t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, recorder.Code)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := newTestServer(t)
			tokenMaker, err := NewPasetoMaker("nigkjtvbrhugwpgaqbemmvnqbtywfrcq")
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest("GET", "/", nil)
			recorder := httptest.NewRecorder()
			tc.setupAuth(t, request, tokenMaker)
			app.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			})).ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestMiddlewarePermissions(t *testing.T) {
	testCases := []struct {
		name          string
		request       func() *http.Request
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "denied with the user does not exist",
			request: func() *http.Request {
				request := httptest.NewRequest("GET", "/", nil)
				payload := &Payload{
					Username: "user",
				}
				ctx := context.WithValue(request.Context(), authorizationPayloadKey, payload)
				reqWithPayload := request.WithContext(ctx)
				return reqWithPayload
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				if recorder.Code != http.StatusUnauthorized {
					t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, recorder.Code)
				}
			},
		},
		{
			name: "denied if payload is empty",
			request: func() *http.Request {
				request := httptest.NewRequest("GET", "/", nil)
				payload := &Payload{}
				ctx := context.WithValue(request.Context(), authorizationPayloadKey, payload)
				reqWithPayload := request.WithContext(ctx)
				return reqWithPayload
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				if recorder.Code != http.StatusUnauthorized {
					t.Fatalf("expected status code %d, got %d", http.StatusUnauthorized, recorder.Code)
				}
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := newTestServer(t)
			recorder := httptest.NewRecorder()
			app.MiddlewarePermissionChecker(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			})).ServeHTTP(recorder, tc.request())
			tc.checkResponse(t, recorder)
		})
	}
}
