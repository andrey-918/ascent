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

	"github.com/stretchr/testify/require"
)

func TestDiaryEntriesHandler(t *testing.T) {
	env := fixtures.NewTestEnv(t)
	cleanup := env.SetupDb(t)
	t.Cleanup(func() {
		cleanup(t)
	})

	dataStore := store.New(env.Db)
	user, err := dataStore.Users.CreateUser(t.Context(), "diary@test.com", "testingpassword")
	require.NoError(t, err)
	server := api.New(env.Config, slog.Default(), dataStore, api.NewJWTManager(env.Config))
	handler := server.Handler()
	tokenPair, err := api.NewJWTManager(env.Config).GenerateTokenPair(user.Id)
	require.NoError(t, err)

	createRequest := httptest.NewRequest(http.MethodPost, "/diary-entries", bytes.NewBufferString(`{"content":"Сегодня отложил 100 ₽"}`))
	createRequest.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken.Raw)
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)

	require.Equal(t, http.StatusCreated, createResponse.Code)
	var created struct {
		Data store.DiaryEntry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(createResponse.Body.Bytes(), &created))
	require.Equal(t, user.Id, created.Data.UserID)
	require.Equal(t, "Сегодня отложил 100 ₽", created.Data.Content)

	otherUser, err := dataStore.Users.CreateUser(t.Context(), "other@test.com", "testingpassword")
	require.NoError(t, err)
	_, err = dataStore.DiaryEntries.Create(t.Context(), otherUser.Id, "Чужая запись")
	require.NoError(t, err)

	listRequest := httptest.NewRequest(http.MethodGet, "/diary-entries", nil)
	listRequest.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken.Raw)
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)

	require.Equal(t, http.StatusOK, listResponse.Code)
	var listed struct {
		Data []store.DiaryEntry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listResponse.Body.Bytes(), &listed))
	require.Len(t, listed.Data, 1)
	require.Equal(t, created.Data.ID, listed.Data[0].ID)

	unauthenticatedRequest := httptest.NewRequest(http.MethodGet, "/diary-entries", nil)
	unauthenticatedResponse := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticatedResponse, unauthenticatedRequest)
	require.Equal(t, http.StatusUnauthorized, unauthenticatedResponse.Code)

	requestById := httptest.NewRequest(http.MethodGet, "/diary-entries/"+created.Data.ID.String(), nil)
	requestById.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken.Raw)
	responseById := httptest.NewRecorder()
	handler.ServeHTTP(responseById, requestById)

	require.Equal(t, http.StatusOK, responseById.Code)
	var entryById struct {
		Data store.DiaryEntry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(responseById.Body.Bytes(), &entryById))
	require.Equal(t, created.Data.ID, entryById.Data.ID)

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/diary-entries/"+created.Data.ID.String(), nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken.Raw)
	deleteResponse := httptest.NewRecorder()
	handler.ServeHTTP(deleteResponse, deleteRequest)

	require.Equal(t, http.StatusNoContent, deleteResponse.Code)

	// Verify that the entry is deleted
	verifyDeleteRequest := httptest.NewRequest(http.MethodGet, "/diary-entries/"+created.Data.ID.String(), nil)
	verifyDeleteRequest.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken.Raw)
	verifyDeleteResponse := httptest.NewRecorder()
	handler.ServeHTTP(verifyDeleteResponse, verifyDeleteRequest)

	require.Equal(t, http.StatusNotFound, verifyDeleteResponse.Code)
}
