package org

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrganizationService is a mock implementation of the organization service
type MockOrganizationService struct {
	mock.Mock
}

func (m *MockOrganizationService) CreateOrganization(ctx context.Context, req CreateOrganizationRequest) (*MockOrganization, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MockOrganization), args.Error(1)
}

func (m *MockOrganizationService) ListOrganizations(ctx context.Context, userID string) ([]*MockOrganization, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]*MockOrganization), args.Error(1)
}

func (m *MockOrganizationService) GetOrganization(ctx context.Context, userID, orgID string) (*MockOrganization, error) {
	args := m.Called(ctx, userID, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MockOrganization), args.Error(1)
}

func (m *MockOrganizationService) UpdateOrganization(ctx context.Context, userID, orgID string, req UpdateOrganizationRequest) (*MockOrganization, error) {
	args := m.Called(ctx, userID, orgID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MockOrganization), args.Error(1)
}

func (m *MockOrganizationService) DeleteOrganization(ctx context.Context, orgID string) error {
	args := m.Called(ctx, orgID)
	return args.Error(0)
}

// Test helper functions
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func createTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

// TestCreateOrganizationHandlerWithJSONData tests the CreateOrganization handler using JSON test data
func TestCreateOrganizationHandlerWithJSONData(t *testing.T) {
	// Load test data
	testDataPath := filepath.Join("test_data")
	loader := NewTestLoader(testDataPath)
	testData, err := loader.LoadTestData("create_organization.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	for _, testCase := range testData.TestCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Setup mocks
			mockService := new(MockOrganizationService)
			logger := logrus.New()

			// Setup handler
			handler := NewOrganizationHandler(mockService, logger)

			// Setup test data
			if testCase.Setup != nil && len(testCase.Setup.ExistingOrganizations) > 0 {
				// Setup existing organizations if needed
				existingOrgs := CreateMockOrganizationsFromSetup(testCase.Setup.ExistingOrganizations)
				for _, org := range existingOrgs {
					mockService.On("GetOrganizationByName", mock.Anything, org.Name).Return(org, nil)
				}
			}

			// Setup expected behavior based on test case
			expectedStatus := GetIntValue(testCase.Expected, "status_code")
			expectedSuccess := GetBoolValue(testCase.Expected, "success")

			if expectedSuccess {
				// Success case - expect organization creation
				expectedOrg := &MockOrganization{
					ID:          uuid.New(),
					Name:        GetStringValue(testCase.Input, "name"),
					Description: GetStringValue(testCase.Input, "description"),
					OwnerID:     uuid.MustParse(GetStringValue(testCase.Input, "user_id")),
				}

				mockService.On("CreateOrganization", mock.Anything, mock.AnythingOfType("CreateOrganizationRequest")).
					Return(expectedOrg, nil)
			} else {
				// Error case - expect error
				errorMsg := GetStringValue(testCase.Expected, "error")
				mockService.On("CreateOrganization", mock.Anything, mock.AnythingOfType("CreateOrganizationRequest")).
					Return(nil, errors.New(errorMsg))
			}

			// Create request
			body, _ := json.Marshal(testCase.Input)
			req := httptest.NewRequest("POST", "/organizations", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Create gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Execute handler
			handler.CreateOrganization(c, nil, nil, nil, nil, nil)

			// Assertions
			assert.Equal(t, expectedStatus, w.Code)

			if expectedSuccess {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "data")
			} else {
				assert.Contains(t, w.Body.String(), "error")
			}

			// Verify mocks
			mockService.AssertExpectations(t)
		})
	}
}

// TestListOrganizationsHandlerWithJSONData tests the ListOrganizations handler using JSON test data
func TestListOrganizationsHandlerWithJSONData(t *testing.T) {
	// Load test data
	testDataPath := filepath.Join("test_data")
	loader := NewTestLoader(testDataPath)
	testData, err := loader.LoadTestData("list_organizations.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}

	for _, testCase := range testData.TestCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// Setup mocks
			mockService := new(MockOrganizationService)
			logger := logrus.New()

			// Setup handler
			handler := NewOrganizationHandler(mockService, logger)

			// Setup test data
			if testCase.Setup != nil && len(testCase.Setup.ExistingOrganizations) > 0 {
				// Setup existing organizations
				existingOrgs := CreateMockOrganizationsFromSetup(testCase.Setup.ExistingOrganizations)
				userID := GetStringValue(testCase.Input, "user_id")
				
				// Filter organizations by owner
				var userOrgs []*MockOrganization
				for _, org := range existingOrgs {
					if org.OwnerID.String() == userID {
						userOrgs = append(userOrgs, org)
					}
				}

				mockService.On("ListOrganizations", mock.Anything, userID).Return(userOrgs, nil)
			} else {
				// No setup data - return empty list
				userID := GetStringValue(testCase.Input, "user_id")
				mockService.On("ListOrganizations", mock.Anything, userID).Return([]*MockOrganization{}, nil)
			}

			// Setup expected behavior
			expectedStatus := GetIntValue(testCase.Expected, "status_code")
			expectedSuccess := GetBoolValue(testCase.Expected, "success")

			if !expectedSuccess {
				// Error case
				errorMsg := GetStringValue(testCase.Expected, "error")
				userID := GetStringValue(testCase.Input, "user_id")
				mockService.On("ListOrganizations", mock.Anything, userID).Return(nil, errors.New(errorMsg))
			}

			// Create request
			req := httptest.NewRequest("GET", "/organizations", nil)

			// Create response recorder
			w := httptest.NewRecorder()

			// Create gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Execute handler
			handler.ListOrganizations(c, nil, nil, nil, nil, nil)

			// Assertions
			assert.Equal(t, expectedStatus, w.Code)

			if expectedSuccess {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "data")
			} else {
				assert.Contains(t, w.Body.String(), "error")
			}

			// Verify mocks
			mockService.AssertExpectations(t)
		})
	}
}

// TestGetOrganizationHandler tests the GetOrganization handler
func TestGetOrganizationHandler(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		setupMocks     func(*MockOrganizationService)
		expectedStatus int
		expectedError  bool
	}{
		{
			name:  "Success - Get organization",
			orgID: uuid.New().String(),
			setupMocks: func(mockService *MockOrganizationService) {
				expectedOrg := &MockOrganization{
					ID:          uuid.New(),
					Name:        "Test Organization",
					Description: "Test organization description",
					OwnerID:     uuid.New(),
				}
				mockService.On("GetOrganization", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(expectedOrg, nil)
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:  "Error - Invalid UUID",
			orgID: "invalid-uuid",
			setupMocks: func(mockService *MockOrganizationService) {
				// No mocks needed as it should fail validation
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:  "Error - Service error",
			orgID: uuid.New().String(),
			setupMocks: func(mockService *MockOrganizationService) {
				mockService.On("GetOrganization", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).
					Return(nil, errors.New("service error"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockService := new(MockOrganizationService)
			logger := logrus.New()

			// Setup handler
			handler := NewOrganizationHandler(mockService, logger)

			// Setup mocks
			tt.setupMocks(mockService)

			// Create request
			req := httptest.NewRequest("GET", "/organizations/"+tt.orgID, nil)

			// Create response recorder
			w := httptest.NewRecorder()

			// Create gin context
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: tt.orgID}}

			// Execute handler
			handler.GetOrganization(c, nil, nil, nil, nil, nil)

			// Assertions
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError {
				assert.Contains(t, w.Body.String(), "error")
			} else {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "data")
			}

			// Verify mocks
			mockService.AssertExpectations(t)
		})
	}
}
