package api_test

import (
	"ascent/api"
	"ascent/fixtures"
	"ascent/store"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGoalsHandler(t *testing.T) {
	env := fixtures.NewTestEnv(t)
	cleanup := env.SetupDb(t)
	t.Cleanup(func() {
		cleanup(t)
	})

	dataStore := store.New(env.Db)
	user, err := dataStore.Users.CreateUser(t.Context(), "goals-api@test.com", "testingpassword")
	require.NoError(t, err)
	otherUser, err := dataStore.Users.CreateUser(t.Context(), "other-goals-api@test.com", "testingpassword")
	require.NoError(t, err)

	jwtManager := api.NewJWTManager(env.Config)
	tokenPair, err := jwtManager.GenerateTokenPair(user.Id)
	require.NoError(t, err)
	server := api.New(env.Config, slog.Default(), dataStore, jwtManager)
	handler := server.Handler()

	doRequest := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		request.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken.Raw)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	for _, body := range []string{
		`{"name":"","target_amount_kopecks":100}`,
		`{"name":"Emergency fund","target_amount_kopecks":0}`,
		`{"name":"   ","target_amount_kopecks":100}`,
	} {
		require.Equal(t, http.StatusBadRequest, doRequest(http.MethodPost, "/goals", body).Code)
	}

	createResponse := doRequest(http.MethodPost, "/goals", `{"name":"Emergency fund","target_amount_kopecks":5000000}`)
	require.Equal(t, http.StatusCreated, createResponse.Code)
	var created struct {
		Data store.Goal `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createResponse.Body.Bytes(), &created))
	require.Equal(t, user.Id, created.Data.UserID)
	require.Equal(t, "Emergency fund", created.Data.Name)
	require.EqualValues(t, 5_000_000, created.Data.TargetAmountKopecks)
	require.Zero(t, created.Data.CurrentAmountKopecks)
	require.Equal(t, "RUB", created.Data.Currency)
	require.Equal(t, "active", created.Data.Status)

	listResponse := doRequest(http.MethodGet, "/goals", "")
	require.Equal(t, http.StatusOK, listResponse.Code)
	var listed struct {
		Data []store.Goal `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listResponse.Body.Bytes(), &listed))
	require.Len(t, listed.Data, 1)
	require.Equal(t, created.Data.ID, listed.Data[0].ID)

	getResponse := doRequest(http.MethodGet, "/goals/"+created.Data.ID.String(), "")
	require.Equal(t, http.StatusOK, getResponse.Code)
	var found struct {
		Data store.Goal `json:"data"`
	}
	require.NoError(t, json.Unmarshal(getResponse.Body.Bytes(), &found))
	require.Equal(t, created.Data.ID, found.Data.ID)

	foreignGoal, err := dataStore.Goals.Create(t.Context(), otherUser.Id, "Other goal", 100)
	require.NoError(t, err)
	require.Equal(t, http.StatusNotFound, doRequest(http.MethodGet, "/goals/"+foreignGoal.ID.String(), "").Code)
	require.Equal(t, http.StatusBadRequest, doRequest(http.MethodGet, "/goals/not-a-uuid", "").Code)
	require.Equal(t, http.StatusNotFound, doRequest(http.MethodGet, "/goals/"+uuid.NewString(), "").Code)
}
