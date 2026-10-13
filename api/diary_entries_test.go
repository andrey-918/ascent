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
	"strings"
	"testing"

	"github.com/google/uuid"
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
	otherEntry, err := dataStore.DiaryEntries.Create(t.Context(), otherUser.Id, "Чужая запись")
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

	doRequest := func(method, path, body string) *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken.Raw)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response
	}

	secondOwnedEntry, err := dataStore.DiaryEntries.Create(t.Context(), user.Id, "Вторая моя запись")
	require.NoError(t, err)
	firstPageResponse := doRequest(http.MethodGet, "/diary-entries?limit=1&offset=0", "")
	require.Equal(t, http.StatusOK, firstPageResponse.Code)
	var firstPage struct {
		Data []store.DiaryEntry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(firstPageResponse.Body.Bytes(), &firstPage))
	require.Len(t, firstPage.Data, 1)

	secondPageResponse := doRequest(http.MethodGet, "/diary-entries?limit=1&offset=1", "")
	require.Equal(t, http.StatusOK, secondPageResponse.Code)
	var secondPage struct {
		Data []store.DiaryEntry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(secondPageResponse.Body.Bytes(), &secondPage))
	require.Len(t, secondPage.Data, 1)
	require.NotEqual(t, firstPage.Data[0].ID, secondPage.Data[0].ID)
	require.ElementsMatch(t, []uuid.UUID{created.Data.ID, secondOwnedEntry.ID}, []uuid.UUID{
		firstPage.Data[0].ID,
		secondPage.Data[0].ID,
	})

	// Invalid and out-of-range pagination parameters are client errors.
	for _, path := range []string{
		"/diary-entries?limit=abc",
		"/diary-entries?limit=0",
		"/diary-entries?limit=101",
		"/diary-entries?offset=-1",
		"/diary-entries?offset=abc",
	} {
		response := doRequest(http.MethodGet, path, "")
		require.Equalf(t, http.StatusBadRequest, response.Code, "path: %s", path)
	}

	// The default page is the first 20 entries; an offset beyond the end is empty.
	emptyPageResponse := doRequest(http.MethodGet, "/diary-entries?limit=20&offset=100", "")
	require.Equal(t, http.StatusOK, emptyPageResponse.Code)
	var emptyPage struct {
		Data []store.DiaryEntry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(emptyPageResponse.Body.Bytes(), &emptyPage))
	require.NotNil(t, emptyPage.Data)
	require.Empty(t, emptyPage.Data)

	unauthenticatedRequest := httptest.NewRequest(http.MethodGet, "/diary-entries", nil)
	unauthenticatedResponse := httptest.NewRecorder()
	handler.ServeHTTP(unauthenticatedResponse, unauthenticatedRequest)
	require.Equal(t, http.StatusUnauthorized, unauthenticatedResponse.Code)
	require.Equal(t, "application/json; charset=utf-8", unauthenticatedResponse.Header().Get("Content-Type"))
	var unauthorizedBody struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(unauthenticatedResponse.Body.Bytes(), &unauthorizedBody))
	require.Equal(t, http.StatusText(http.StatusUnauthorized), unauthorizedBody.Message)

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

	// A user cannot retrieve or delete another user's entry.
	foreignEntryPath := "/diary-entries/" + otherEntry.ID.String()
	require.Equal(t, http.StatusNotFound, doRequest(http.MethodGet, foreignEntryPath, "").Code)
	foreignDeleteResponse := doRequest(http.MethodDelete, foreignEntryPath, "")
	require.Equal(t, http.StatusNotFound, foreignDeleteResponse.Code)
	_, err = dataStore.DiaryEntries.ById(t.Context(), otherEntry.ID, otherUser.Id)
	require.NoError(t, err, "foreign entry must remain after rejected deletion")

	// Invalid UUIDs and valid but missing IDs have distinct client-facing cases.
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		require.Equal(t, http.StatusBadRequest, doRequest(method, "/diary-entries/not-a-uuid", "").Code)
		require.Equal(t, http.StatusNotFound, doRequest(method, "/diary-entries/"+uuid.NewString(), "").Code)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/diary-entries/"+created.Data.ID.String(), nil)
	deleteRequest.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken.Raw)
	deleteResponse := httptest.NewRecorder()
	handler.ServeHTTP(deleteResponse, deleteRequest)

	require.Equal(t, http.StatusNoContent, deleteResponse.Code)
	require.Empty(t, deleteResponse.Body.String())

	// Verify that the entry is deleted
	verifyDeleteRequest := httptest.NewRequest(http.MethodGet, "/diary-entries/"+created.Data.ID.String(), nil)
	verifyDeleteRequest.Header.Set("Authorization", "Bearer "+tokenPair.AccessToken.Raw)
	verifyDeleteResponse := httptest.NewRecorder()
	handler.ServeHTTP(verifyDeleteResponse, verifyDeleteRequest)

	require.Equal(t, http.StatusNotFound, verifyDeleteResponse.Code)
}
